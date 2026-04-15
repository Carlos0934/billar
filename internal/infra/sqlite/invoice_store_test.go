package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/Carlos0934/billar/internal/core"
)

func TestInvoiceStoreCreateDraft(t *testing.T) {
	t.Parallel()

	store, err := Open("")
	if err != nil {
		t.Fatalf("Open(): %v", err)
	}
	defer store.Close()

	customerID, agreementID := seedCustomerAndAgreement(t, store)

	entry := &core.TimeEntry{
		ID:                 "te_001",
		ServiceAgreementID: agreementID,
		CustomerProfileID:  customerID,
		Description:        "Work",
		Hours:              mustHours(15000),
		Billable:           true,
		Date:               time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC),
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
	if err := NewTimeEntryStore(store).Save(context.Background(), entry); err != nil {
		t.Fatalf("save entry: %v", err)
	}

	rate, _ := core.NewMoney(10000, "USD")
	line, _ := core.NewInvoiceLine(core.InvoiceLineParams{
		InvoiceID:          "inv_seed",
		ServiceAgreementID: agreementID,
		TimeEntryID:        entry.ID,
		UnitRate:           rate,
	})

	invoice, _ := core.NewInvoice(core.InvoiceParams{
		CustomerID: customerID,
		Status:     core.InvoiceStatusDraft,
		Currency:   "USD",
		Lines:      []core.InvoiceLine{line},
	})

	invStore := NewInvoiceStore(store)
	if err := invStore.CreateDraft(context.Background(), &invoice, []*core.TimeEntry{entry}); err != nil {
		t.Fatalf("CreateDraft() error = %v", err)
	}

	got, err := invStore.GetByID(context.Background(), invoice.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got.Status != core.InvoiceStatusDraft {
		t.Fatalf("Status = %q, want draft", got.Status)
	}
	if got.CustomerID != customerID {
		t.Fatalf("CustomerID = %q, want %q", got.CustomerID, customerID)
	}
	if len(got.Lines) != 1 {
		t.Fatalf("len(Lines) = %d, want 1", len(got.Lines))
	}
	if got.Lines[0].TimeEntryID != entry.ID {
		t.Fatalf("Line TimeEntryID = %q, want %q", got.Lines[0].TimeEntryID, entry.ID)
	}
}

func TestInvoiceStoreGetByID_NotFound(t *testing.T) {
	t.Parallel()

	store, err := Open("")
	if err != nil {
		t.Fatalf("Open(): %v", err)
	}
	defer store.Close()

	invStore := NewInvoiceStore(store)
	_, err = invStore.GetByID(context.Background(), "inv_nonexistent")
	if err == nil {
		t.Fatal("GetByID() error = nil, want not found")
	}
}

func TestInvoiceStoreUpdate(t *testing.T) {
	t.Parallel()

	store, err := Open("")
	if err != nil {
		t.Fatalf("Open(): %v", err)
	}
	defer store.Close()

	customerID, agreementID := seedCustomerAndAgreement(t, store)

	entry := &core.TimeEntry{
		ID:                 "te_001",
		ServiceAgreementID: agreementID,
		CustomerProfileID:  customerID,
		Description:        "Work",
		Hours:              mustHours(15000),
		Billable:           true,
		Date:               time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC),
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
	if err := NewTimeEntryStore(store).Save(context.Background(), entry); err != nil {
		t.Fatalf("save entry: %v", err)
	}

	rate, _ := core.NewMoney(10000, "USD")
	line, _ := core.NewInvoiceLine(core.InvoiceLineParams{
		InvoiceID:          "inv_seed",
		ServiceAgreementID: agreementID,
		TimeEntryID:        entry.ID,
		UnitRate:           rate,
	})

	invoice, _ := core.NewInvoice(core.InvoiceParams{
		CustomerID: customerID,
		Status:     core.InvoiceStatusDraft,
		Currency:   "USD",
		Lines:      []core.InvoiceLine{line},
	})

	invStore := NewInvoiceStore(store)
	if err := invStore.CreateDraft(context.Background(), &invoice, []*core.TimeEntry{entry}); err != nil {
		t.Fatalf("CreateDraft() error = %v", err)
	}

	issuedAt := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	if err := invoice.Issue("INV-2026-0001", issuedAt); err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	if err := invStore.Update(context.Background(), &invoice); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	got, err := invStore.GetByID(context.Background(), invoice.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got.Status != core.InvoiceStatusIssued {
		t.Fatalf("Status = %q, want issued", got.Status)
	}
	if got.InvoiceNumber != "INV-2026-0001" {
		t.Fatalf("InvoiceNumber = %q, want INV-2026-0001", got.InvoiceNumber)
	}
}

func TestInvoiceStoreDelete(t *testing.T) {
	t.Parallel()

	store, err := Open("")
	if err != nil {
		t.Fatalf("Open(): %v", err)
	}
	defer store.Close()

	customerID, agreementID := seedCustomerAndAgreement(t, store)

	entry := &core.TimeEntry{
		ID:                 "te_001",
		ServiceAgreementID: agreementID,
		CustomerProfileID:  customerID,
		Description:        "Work",
		Hours:              mustHours(15000),
		Billable:           true,
		Date:               time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC),
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
	if err := NewTimeEntryStore(store).Save(context.Background(), entry); err != nil {
		t.Fatalf("save entry: %v", err)
	}

	rate, _ := core.NewMoney(10000, "USD")
	line, _ := core.NewInvoiceLine(core.InvoiceLineParams{
		InvoiceID:          "inv_seed",
		ServiceAgreementID: agreementID,
		TimeEntryID:        entry.ID,
		UnitRate:           rate,
	})

	invoice, _ := core.NewInvoice(core.InvoiceParams{
		CustomerID: customerID,
		Status:     core.InvoiceStatusDraft,
		Currency:   "USD",
		Lines:      []core.InvoiceLine{line},
	})

	invStore := NewInvoiceStore(store)
	if err := invStore.CreateDraft(context.Background(), &invoice, []*core.TimeEntry{entry}); err != nil {
		t.Fatalf("CreateDraft() error = %v", err)
	}

	if err := invStore.Delete(context.Background(), invoice.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err = invStore.GetByID(context.Background(), invoice.ID)
	if err == nil {
		t.Fatal("GetByID() after delete error = nil, want not found")
	}

	// Time entry should be unlocked (invoice_id = NULL).
	teStore := NewTimeEntryStore(store)
	gotEntry, err := teStore.GetByID(context.Background(), entry.ID)
	if err != nil {
		t.Fatalf("Get entry after delete: %v", err)
	}
	if gotEntry.InvoiceID != "" {
		t.Fatalf("Entry InvoiceID = %q, want empty after draft delete", gotEntry.InvoiceID)
	}
}

func TestInvoiceStoreSoftDiscard(t *testing.T) {
	t.Parallel()

	store, err := Open("")
	if err != nil {
		t.Fatalf("Open(): %v", err)
	}
	defer store.Close()

	customerID, agreementID := seedCustomerAndAgreement(t, store)

	entry := &core.TimeEntry{
		ID:                 "te_001",
		ServiceAgreementID: agreementID,
		CustomerProfileID:  customerID,
		Description:        "Work",
		Hours:              mustHours(15000),
		Billable:           true,
		Date:               time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC),
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
	if err := NewTimeEntryStore(store).Save(context.Background(), entry); err != nil {
		t.Fatalf("save entry: %v", err)
	}

	rate, _ := core.NewMoney(10000, "USD")
	line, _ := core.NewInvoiceLine(core.InvoiceLineParams{
		InvoiceID:          "inv_seed",
		ServiceAgreementID: agreementID,
		TimeEntryID:        entry.ID,
		UnitRate:           rate,
	})

	invoice, _ := core.NewInvoice(core.InvoiceParams{
		CustomerID: customerID,
		Status:     core.InvoiceStatusDraft,
		Currency:   "USD",
		Lines:      []core.InvoiceLine{line},
	})

	invStore := NewInvoiceStore(store)
	if err := invStore.CreateDraft(context.Background(), &invoice, []*core.TimeEntry{entry}); err != nil {
		t.Fatalf("CreateDraft() error = %v", err)
	}

	// Issue the invoice.
	issuedAt := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	if err := invoice.Issue("INV-2026-0001", issuedAt); err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	if err := invStore.Update(context.Background(), &invoice); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Soft-discard (issued path).
	discardedAt := time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC)
	if err := invoice.Discard(discardedAt); err != nil {
		t.Fatalf("Discard() error = %v", err)
	}
	if err := invStore.Update(context.Background(), &invoice); err != nil {
		t.Fatalf("Update() soft-discard error = %v", err)
	}

	// Invoice should still exist with status=discarded.
	got, err := invStore.GetByID(context.Background(), invoice.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got.Status != core.InvoiceStatusDiscarded {
		t.Fatalf("Status = %q, want discarded", got.Status)
	}
	if got.InvoiceNumber != "INV-2026-0001" {
		t.Fatalf("InvoiceNumber = %q, want INV-2026-0001", got.InvoiceNumber)
	}

	// Time entry should still be locked to the invoice.
	teStore := NewTimeEntryStore(store)
	gotEntry, err := teStore.GetByID(context.Background(), entry.ID)
	if err != nil {
		t.Fatalf("Get entry after soft-discard: %v", err)
	}
	if gotEntry.InvoiceID != invoice.ID {
		t.Fatalf("Entry InvoiceID = %q, want %q (should remain locked)", gotEntry.InvoiceID, invoice.ID)
	}
}

func TestInvoiceStoreSoftDiscard_NextIssueGetsNextNumber(t *testing.T) {
	t.Parallel()

	store, err := Open("")
	if err != nil {
		t.Fatalf("Open(): %v", err)
	}
	defer store.Close()

	customerID, agreementID := seedCustomerAndAgreement(t, store)

	entry := &core.TimeEntry{
		ID:                 "te_001",
		ServiceAgreementID: agreementID,
		CustomerProfileID:  customerID,
		Description:        "Work",
		Hours:              mustHours(15000),
		Billable:           true,
		Date:               time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC),
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
	if err := NewTimeEntryStore(store).Save(context.Background(), entry); err != nil {
		t.Fatalf("save entry: %v", err)
	}

	rate, _ := core.NewMoney(10000, "USD")
	line, _ := core.NewInvoiceLine(core.InvoiceLineParams{
		InvoiceID:          "inv_seed",
		ServiceAgreementID: agreementID,
		TimeEntryID:        entry.ID,
		UnitRate:           rate,
	})

	invStore := NewInvoiceStore(store)
	seqStore := NewInvoiceSequenceStore(store)

	// Create and issue first invoice.
	invoice1, _ := core.NewInvoice(core.InvoiceParams{
		CustomerID: customerID,
		Status:     core.InvoiceStatusDraft,
		Currency:   "USD",
		Lines:      []core.InvoiceLine{line},
	})
	if err := invStore.CreateDraft(context.Background(), &invoice1, []*core.TimeEntry{entry}); err != nil {
		t.Fatalf("CreateDraft() error = %v", err)
	}
	num1, err := seqStore.Next(context.Background())
	if err != nil {
		t.Fatalf("first Next() error = %v", err)
	}
	if err := invoice1.Issue(num1, time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	if err := invStore.Update(context.Background(), &invoice1); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if num1 != "INV-2026-0001" {
		t.Fatalf("first number = %q, want INV-2026-0001", num1)
	}

	// Soft-discard the issued invoice.
	if err := invoice1.Discard(time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("Discard() error = %v", err)
	}
	if err := invStore.Update(context.Background(), &invoice1); err != nil {
		t.Fatalf("soft-discard Update() error = %v", err)
	}

	// Create and issue second invoice — must get INV-2026-0002, not reuse 0001.
	entry2 := &core.TimeEntry{
		ID:                 "te_002",
		ServiceAgreementID: agreementID,
		CustomerProfileID:  customerID,
		Description:        "Work 2",
		Hours:              mustHours(15000),
		Billable:           true,
		Date:               time.Date(2026, 4, 11, 0, 0, 0, 0, time.UTC),
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
	if err := NewTimeEntryStore(store).Save(context.Background(), entry2); err != nil {
		t.Fatalf("save entry2: %v", err)
	}
	line2, _ := core.NewInvoiceLine(core.InvoiceLineParams{
		InvoiceID:          "inv_seed",
		ServiceAgreementID: agreementID,
		TimeEntryID:        entry2.ID,
		UnitRate:           rate,
	})
	invoice2, _ := core.NewInvoice(core.InvoiceParams{
		CustomerID: customerID,
		Status:     core.InvoiceStatusDraft,
		Currency:   "USD",
		Lines:      []core.InvoiceLine{line2},
	})
	if err := invStore.CreateDraft(context.Background(), &invoice2, []*core.TimeEntry{entry2}); err != nil {
		t.Fatalf("CreateDraft() second error = %v", err)
	}
	num2, err := seqStore.Next(context.Background())
	if err != nil {
		t.Fatalf("second Next() error = %v", err)
	}
	if num2 != "INV-2026-0002" {
		t.Fatalf("second number = %q, want INV-2026-0002 (must not reuse 0001)", num2)
	}
	if err := invoice2.Issue(num2, time.Date(2026, 4, 12, 12, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("Issue() second error = %v", err)
	}
	if err := invStore.Update(context.Background(), &invoice2); err != nil {
		t.Fatalf("Update() second error = %v", err)
	}

	// Verify first invoice still exists as discarded.
	got1, err := invStore.GetByID(context.Background(), invoice1.ID)
	if err != nil {
		t.Fatalf("GetByID() first error = %v", err)
	}
	if got1.Status != core.InvoiceStatusDiscarded {
		t.Fatalf("first invoice status = %q, want discarded", got1.Status)
	}
	if got1.InvoiceNumber != "INV-2026-0001" {
		t.Fatalf("first invoice number = %q, want INV-2026-0001", got1.InvoiceNumber)
	}
}

func TestInvoiceStoreDelete_IsAtomic(t *testing.T) {
	t.Parallel()

	store, err := Open("")
	if err != nil {
		t.Fatalf("Open(): %v", err)
	}
	defer store.Close()

	customerID, agreementID := seedCustomerAndAgreement(t, store)

	entry := &core.TimeEntry{
		ID:                 "te_001",
		ServiceAgreementID: agreementID,
		CustomerProfileID:  customerID,
		Description:        "Work",
		Hours:              mustHours(15000),
		Billable:           true,
		Date:               time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC),
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
	if err := NewTimeEntryStore(store).Save(context.Background(), entry); err != nil {
		t.Fatalf("save entry: %v", err)
	}

	rate, _ := core.NewMoney(10000, "USD")
	line, _ := core.NewInvoiceLine(core.InvoiceLineParams{
		InvoiceID:          "inv_seed",
		ServiceAgreementID: agreementID,
		TimeEntryID:        entry.ID,
		UnitRate:           rate,
	})

	invoice, _ := core.NewInvoice(core.InvoiceParams{
		CustomerID: customerID,
		Status:     core.InvoiceStatusDraft,
		Currency:   "USD",
		Lines:      []core.InvoiceLine{line},
	})

	invStore := NewInvoiceStore(store)
	if err := invStore.CreateDraft(context.Background(), &invoice, []*core.TimeEntry{entry}); err != nil {
		t.Fatalf("CreateDraft() error = %v", err)
	}

	// Verify entry is locked.
	teStore := NewTimeEntryStore(store)
	gotEntry, err := teStore.GetByID(context.Background(), entry.ID)
	if err != nil {
		t.Fatalf("Get entry before delete: %v", err)
	}
	if gotEntry.InvoiceID != invoice.ID {
		t.Fatalf("Entry InvoiceID = %q, want %q (should be locked)", gotEntry.InvoiceID, invoice.ID)
	}

	// Delete should unlock entries and remove invoice atomically.
	if err := invStore.Delete(context.Background(), invoice.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify invoice is gone.
	_, err = invStore.GetByID(context.Background(), invoice.ID)
	if err == nil {
		t.Fatal("GetByID() after delete error = nil, want not found")
	}

	// Verify entry is unlocked.
	gotEntry, err = teStore.GetByID(context.Background(), entry.ID)
	if err != nil {
		t.Fatalf("Get entry after delete: %v", err)
	}
	if gotEntry.InvoiceID != "" {
		t.Fatalf("Entry InvoiceID = %q, want empty (should be unlocked)", gotEntry.InvoiceID)
	}
}

// seedCustomerAndAgreement creates a minimal customer profile and service agreement for testing.
func seedCustomerAndAgreement(t *testing.T, store *Store) (customerID, agreementID string) {
	t.Helper()

	db := store.DB()

	// Create legal entity.
	_, err := db.Exec(`INSERT INTO legal_entities (id, type, legal_name, created_at, updated_at) VALUES (?, 'company', 'Test Co', ?, ?)`,
		"le_test", time.Now().UTC().UnixNano(), time.Now().UTC().UnixNano())
	if err != nil {
		t.Fatalf("seed legal entity: %v", err)
	}

	// Create customer profile.
	_, err = db.Exec(`INSERT INTO customer_profiles (id, legal_entity_id, status, default_currency, created_at, updated_at) VALUES (?, ?, 'active', 'USD', ?, ?)`,
		"cus_test", "le_test", time.Now().UTC().UnixNano(), time.Now().UTC().UnixNano())
	if err != nil {
		t.Fatalf("seed customer: %v", err)
	}

	// Create service agreement.
	_, err = db.Exec(`INSERT INTO service_agreements (id, customer_profile_id, name, billing_mode, hourly_rate, currency, active, created_at, updated_at) VALUES (?, ?, 'Support', 'hourly', 10000, 'USD', 1, ?, ?)`,
		"sa_test", "cus_test", time.Now().UTC().UnixNano(), time.Now().UTC().UnixNano())
	if err != nil {
		t.Fatalf("seed agreement: %v", err)
	}

	return "cus_test", "sa_test"
}

func TestNewInvoiceStore_NilStore(t *testing.T) {
	t.Parallel()

	got := NewInvoiceStore(nil)
	if got != nil {
		t.Fatalf("NewInvoiceStore(nil) = %v, want nil", got)
	}
}

func TestInvoiceStoreCreateDraft_NilInvoice(t *testing.T) {
	t.Parallel()

	store, err := Open("")
	if err != nil {
		t.Fatalf("Open(): %v", err)
	}
	defer store.Close()

	invStore := NewInvoiceStore(store)
	err = invStore.CreateDraft(context.Background(), nil, nil)
	if err == nil {
		t.Fatal("CreateDraft(nil) error = nil, want error")
	}
}

func TestInvoiceStoreUpdate_NilInvoice(t *testing.T) {
	t.Parallel()

	store, err := Open("")
	if err != nil {
		t.Fatalf("Open(): %v", err)
	}
	defer store.Close()

	invStore := NewInvoiceStore(store)
	err = invStore.Update(context.Background(), nil)
	if err == nil {
		t.Fatal("Update(nil) error = nil, want error")
	}
}

func TestInvoiceStoreGetByID_MultipleLines(t *testing.T) {
	t.Parallel()

	store, err := Open("")
	if err != nil {
		t.Fatalf("Open(): %v", err)
	}
	defer store.Close()

	customerID, agreementID := seedCustomerAndAgreement(t, store)

	entry1 := &core.TimeEntry{
		ID:                 "te_ml_1",
		ServiceAgreementID: agreementID,
		CustomerProfileID:  customerID,
		Description:        "Work 1",
		Hours:              mustHours(15000),
		Billable:           true,
		Date:               time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC),
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
	entry2 := &core.TimeEntry{
		ID:                 "te_ml_2",
		ServiceAgreementID: agreementID,
		CustomerProfileID:  customerID,
		Description:        "Work 2",
		Hours:              mustHours(20000),
		Billable:           true,
		Date:               time.Date(2026, 4, 9, 0, 0, 0, 0, time.UTC),
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
	teStore := NewTimeEntryStore(store)
	for _, e := range []*core.TimeEntry{entry1, entry2} {
		if err := teStore.Save(context.Background(), e); err != nil {
			t.Fatalf("save entry: %v", err)
		}
	}

	rate, _ := core.NewMoney(10000, "USD")
	line1, _ := core.NewInvoiceLine(core.InvoiceLineParams{
		InvoiceID:          "inv_ml_seed",
		ServiceAgreementID: agreementID,
		TimeEntryID:        entry1.ID,
		UnitRate:           rate,
	})
	line2, _ := core.NewInvoiceLine(core.InvoiceLineParams{
		InvoiceID:          "inv_ml_seed",
		ServiceAgreementID: agreementID,
		TimeEntryID:        entry2.ID,
		UnitRate:           rate,
	})

	invoice, _ := core.NewInvoice(core.InvoiceParams{
		CustomerID: customerID,
		Status:     core.InvoiceStatusDraft,
		Currency:   "USD",
		Lines:      []core.InvoiceLine{line1, line2},
	})

	invStore := NewInvoiceStore(store)
	if err := invStore.CreateDraft(context.Background(), &invoice, []*core.TimeEntry{entry1, entry2}); err != nil {
		t.Fatalf("CreateDraft() error = %v", err)
	}

	got, err := invStore.GetByID(context.Background(), invoice.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if len(got.Lines) != 2 {
		t.Fatalf("len(Lines) = %d, want 2", len(got.Lines))
	}
}

func TestInvoiceStoreDelete_AlreadyGone(t *testing.T) {
	t.Parallel()

	store, err := Open("")
	if err != nil {
		t.Fatalf("Open(): %v", err)
	}
	defer store.Close()

	invStore := NewInvoiceStore(store)
	// Deleting a non-existent invoice should not error (SQL UPDATE/DELETE with 0 rows is fine).
	if err := invStore.Delete(context.Background(), "inv_does_not_exist"); err != nil {
		t.Fatalf("Delete(nonexistent) error = %v, want nil", err)
	}
}

func mustHours(amount int64) core.Hours {
	h, err := core.NewHours(amount)
	if err != nil {
		panic(err)
	}
	return h
}
