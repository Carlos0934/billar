package sqlite

import (
	"context"
	"strings"
	"sync"
	"testing"
)

func TestInvoiceSequenceStore_FirstCallReturnsOne(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	seq := &InvoiceSequenceStore{db: store.DB()}

	got, err := seq.nextForYear(context.Background(), 2026)
	if err != nil {
		t.Fatalf("nextForYear() error = %v", err)
	}
	if got != "INV-2026-0001" {
		t.Fatalf("nextForYear() = %q, want INV-2026-0001", got)
	}
}

func TestInvoiceSequenceStore_SequentialAllocation(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	seq := &InvoiceSequenceStore{db: store.DB()}

	ctx := context.Background()

	first, err := seq.nextForYear(ctx, 2026)
	if err != nil {
		t.Fatalf("first call error = %v", err)
	}
	if first != "INV-2026-0001" {
		t.Fatalf("first call = %q, want INV-2026-0001", first)
	}

	second, err := seq.nextForYear(ctx, 2026)
	if err != nil {
		t.Fatalf("second call error = %v", err)
	}
	if second != "INV-2026-0002" {
		t.Fatalf("second call = %q, want INV-2026-0002", second)
	}
}

func TestInvoiceSequenceStore_SpecFormattingExamples(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	seq := &InvoiceSequenceStore{db: store.DB()}

	ctx := context.Background()

	// Spec example: sequence 42 → INV-2026-0042
	for i := 0; i < 41; i++ {
		if _, err := seq.nextForYear(ctx, 2026); err != nil {
			t.Fatalf("advance iteration %d error = %v", i, err)
		}
	}

	got, err := seq.nextForYear(ctx, 2026)
	if err != nil {
		t.Fatalf("nextForYear() error = %v", err)
	}
	if got != "INV-2026-0042" {
		t.Fatalf("nextForYear() = %q, want INV-2026-0042", got)
	}

	// Spec example: sequence 10500 → INV-2026-10500
	for i := 42; i < 10499; i++ {
		if _, err := seq.nextForYear(ctx, 2026); err != nil {
			t.Fatalf("advance iteration %d error = %v", i, err)
		}
	}

	got, err = seq.nextForYear(ctx, 2026)
	if err != nil {
		t.Fatalf("nextForYear() error = %v", err)
	}
	if got != "INV-2026-10500" {
		t.Fatalf("nextForYear() = %q, want INV-2026-10500", got)
	}
}

func TestInvoiceSequenceStore_LargeSequenceFormatting(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	seq := &InvoiceSequenceStore{db: store.DB()}

	ctx := context.Background()

	// Advance to sequence 42
	for i := 0; i < 42; i++ {
		if _, err := seq.nextForYear(ctx, 2026); err != nil {
			t.Fatalf("advance iteration %d error = %v", i, err)
		}
	}

	got, err := seq.nextForYear(ctx, 2026)
	if err != nil {
		t.Fatalf("nextForYear() error = %v", err)
	}
	if got != "INV-2026-0043" {
		t.Fatalf("nextForYear() = %q, want INV-2026-0043", got)
	}

	// Advance to sequence 10500
	for i := 43; i < 10500; i++ {
		if _, err := seq.nextForYear(ctx, 2026); err != nil {
			t.Fatalf("advance iteration %d error = %v", i, err)
		}
	}

	got, err = seq.nextForYear(ctx, 2026)
	if err != nil {
		t.Fatalf("nextForYear() error = %v", err)
	}
	if got != "INV-2026-10501" {
		t.Fatalf("nextForYear() = %q, want INV-2026-10501", got)
	}
}

func TestInvoiceSequenceStore_YearRollover(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	seq := &InvoiceSequenceStore{db: store.DB()}

	ctx := context.Background()

	// Allocate some numbers for 2026
	for i := 0; i < 5; i++ {
		if _, err := seq.nextForYear(ctx, 2026); err != nil {
			t.Fatalf("2026 allocation %d error = %v", i, err)
		}
	}

	// First call for 2027 should start at 1
	got, err := seq.nextForYear(ctx, 2027)
	if err != nil {
		t.Fatalf("2027 first call error = %v", err)
	}
	if got != "INV-2027-0001" {
		t.Fatalf("2027 first call = %q, want INV-2027-0001", got)
	}

	// 2026 sequence should remain unchanged at 5
	var count int
	err = store.DB().QueryRowContext(ctx, "SELECT next_seq - 1 FROM invoice_sequences WHERE year = 2026").Scan(&count)
	if err != nil {
		t.Fatalf("query 2026 sequence error = %v", err)
	}
	if count != 5 {
		t.Fatalf("2026 allocated count = %d, want 5", count)
	}
}

func TestInvoiceSequenceStore_ConcurrentAllocation(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	seq := &InvoiceSequenceStore{db: store.DB()}

	const goroutines = 50
	results := make([]string, goroutines)
	var wg sync.WaitGroup
	var mu sync.Mutex
	errs := make([]error, 0)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			num, err := seq.nextForYear(context.Background(), 2026)
			if err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
				return
			}
			results[idx] = num
		}(i)
	}

	wg.Wait()

	if len(errs) > 0 {
		t.Fatalf("concurrent errors: %v", errs[0])
	}

	// Verify exactly 50 unique numbers
	seen := make(map[string]bool)
	for _, num := range results {
		if num == "" {
			t.Fatal("empty result in concurrent allocation")
		}
		if seen[num] {
			t.Fatalf("duplicate number: %q", num)
		}
		seen[num] = true
	}
	if len(seen) != goroutines {
		t.Fatalf("got %d unique numbers, want %d", len(seen), goroutines)
	}

	// Verify gap-free: sequence should be exactly goroutines
	var total int
	err := store.DB().QueryRowContext(context.Background(), "SELECT next_seq - 1 FROM invoice_sequences WHERE year = 2026").Scan(&total)
	if err != nil {
		t.Fatalf("query total error = %v", err)
	}
	if total != goroutines {
		t.Fatalf("total allocated = %d, want %d", total, goroutines)
	}
}

func TestInvoiceSequenceStore_NextUsesCurrentYear(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	seq := &InvoiceSequenceStore{db: store.DB()}

	ctx := context.Background()

	// Next(ctx) derives the year from the system clock.
	// Verify it returns a properly formatted string for the current year.
	got, err := seq.Next(ctx)
	if err != nil {
		t.Fatalf("Next() error = %v", err)
	}

	// The result must match INV-YYYY-0001 for the first call.
	// We verify format structure rather than exact year to avoid clock coupling.
	if !strings.HasPrefix(got, "INV-") {
		t.Fatalf("Next() = %q, want prefix INV-", got)
	}
	if !strings.HasSuffix(got, "-0001") {
		t.Fatalf("Next() = %q, want suffix -0001", got)
	}

	// Second call must be sequential.
	got2, err := seq.Next(ctx)
	if err != nil {
		t.Fatalf("Next() second call error = %v", err)
	}
	if !strings.HasSuffix(got2, "-0002") {
		t.Fatalf("Next() second call = %q, want suffix -0002", got2)
	}
}

func TestInvoiceSequenceStore_PersistenceFailure(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	seq := &InvoiceSequenceStore{db: store.DB()}

	// Close the DB to simulate failure
	if err := store.db.Close(); err != nil {
		t.Fatalf("close db error = %v", err)
	}

	_, err := seq.nextForYear(context.Background(), 2026)
	if err == nil {
		t.Fatal("nextForYear() error = nil, want error on closed DB")
	}
}
