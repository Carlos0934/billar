package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// InvoiceSequenceStore persists global yearly invoice sequences in SQLite.
// It produces gap-free, monotonically increasing numbers formatted as INV-YYYY-NNNN.
type InvoiceSequenceStore struct {
	db *sql.DB
}

// NewInvoiceSequenceStore constructs an InvoiceSequenceStore from an open Store.
func NewInvoiceSequenceStore(store *Store) *InvoiceSequenceStore {
	if store == nil {
		return nil
	}
	return &InvoiceSequenceStore{db: store.DB()}
}

// Next allocates and returns the next invoice number for the current UTC year.
// The year is derived from the system clock at call time.
func (s *InvoiceSequenceStore) Next(ctx context.Context) (string, error) {
	return s.nextForYear(ctx, time.Now().UTC().Year())
}

// nextForYear allocates and returns the next invoice number for the given year.
// This method is unexported but accessible within the sqlite package for testing.
func (s *InvoiceSequenceStore) nextForYear(ctx context.Context, year int) (string, error) {
	var seq int
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO invoice_sequences (year, next_seq) VALUES (?, 2)
		 ON CONFLICT(year) DO UPDATE SET next_seq = next_seq + 1
		 RETURNING next_seq - 1`,
		year,
	).Scan(&seq)
	if err != nil {
		return "", fmt.Errorf("allocate invoice sequence for %d: %w", year, err)
	}
	return fmt.Sprintf("INV-%04d-%04d", year, seq), nil
}
