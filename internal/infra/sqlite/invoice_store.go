package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Carlos0934/billar/internal/app"
	"github.com/Carlos0934/billar/internal/core"
)

// InvoiceStore persists Invoice entities and their lines in SQLite.
type InvoiceStore struct {
	db *sql.DB
}

// NewInvoiceStore constructs an InvoiceStore from an open Store.
func NewInvoiceStore(store *Store) *InvoiceStore {
	if store == nil {
		return nil
	}
	return &InvoiceStore{db: store.DB()}
}

// CreateDraft inserts an invoice, its lines, and links the time entries
// to the invoice in a single transaction.
func (s *InvoiceStore) CreateDraft(ctx context.Context, invoice *core.Invoice, entries []*core.TimeEntry) error {
	if s == nil || s.db == nil {
		return errors.New("invoice sqlite store is required")
	}
	if invoice == nil {
		return errors.New("invoice is required")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert invoice.
	_, err = tx.ExecContext(ctx, `
INSERT INTO invoices (id, invoice_number, customer_id, status, currency, period_start, period_end, due_date, notes, issued_at, discarded_at, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		invoice.ID,
		invoice.InvoiceNumber,
		invoice.CustomerID,
		string(invoice.Status),
		invoice.Currency,
		timeToNano(invoice.PeriodStart),
		timeToNano(invoice.PeriodEnd),
		timeToNano(invoice.DueDate),
		invoice.Notes,
		timeToNano(invoice.IssuedAt),
		timeToNano(invoice.DiscardedAt),
		invoice.CreatedAt.UTC().UnixNano(),
		invoice.UpdatedAt.UTC().UnixNano(),
	)
	if err != nil {
		return fmt.Errorf("insert invoice: %w", err)
	}

	entryByID := make(map[string]*core.TimeEntry, len(entries))
	for _, entry := range entries {
		entryByID[entry.ID] = entry
	}
	// Insert lines.
	for _, line := range invoice.Lines {
		if entry := entryByID[line.TimeEntryID]; entry != nil {
			if line.Description == "" {
				line.Description = entry.Description
			}
			if line.QuantityMin == 0 {
				line.QuantityMin = int64(entry.Hours) * 60 / 10000
			}
		}
		_, err = tx.ExecContext(ctx, `
INSERT INTO invoice_lines (id, invoice_id, service_agreement_id, time_entry_id, description, quantity_min, unit_rate_amount, unit_rate_currency)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			line.ID,
			invoice.ID,
			line.ServiceAgreementID,
			nullString(line.TimeEntryID),
			line.Description,
			line.QuantityMin,
			line.UnitRate.Amount,
			line.UnitRate.Currency,
		)
		if err != nil {
			return fmt.Errorf("insert invoice line: %w", err)
		}
	}

	// Lock time entries to this invoice.
	for _, entry := range entries {
		_, err = tx.ExecContext(ctx, `UPDATE time_entries SET invoice_id = ?, updated_at = ? WHERE id = ?`,
			invoice.ID,
			entry.UpdatedAt.UTC().UnixNano(),
			entry.ID,
		)
		if err != nil {
			return fmt.Errorf("lock time entry %s: %w", entry.ID, err)
		}
	}

	return tx.Commit()
}

// GetByID fetches an invoice and its lines by ID.
func (s *InvoiceStore) GetByID(ctx context.Context, id string) (*core.Invoice, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("invoice sqlite store is required")
	}

	var invoice core.Invoice
	var invoiceNumber, status, currency string
	var customerID string
	var periodStartNano, periodEndNano, dueDateNano, issuedAtNano, discardedAtNano, createdAtNano, updatedAtNano sql.NullInt64

	err := s.db.QueryRowContext(ctx, `
SELECT id, invoice_number, customer_id, status, currency, period_start, period_end, due_date, notes, issued_at, discarded_at, created_at, updated_at
FROM invoices WHERE id = ?`, id).Scan(
		&invoice.ID,
		&invoiceNumber,
		&customerID,
		&status,
		&currency,
		&periodStartNano,
		&periodEndNano,
		&dueDateNano,
		&invoice.Notes,
		&issuedAtNano,
		&discardedAtNano,
		&createdAtNano,
		&updatedAtNano,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, app.ErrInvoiceNotFound
		}
		return nil, fmt.Errorf("get invoice by id: %w", err)
	}

	invoice.InvoiceNumber = invoiceNumber
	invoice.CustomerID = customerID
	invoice.Status = core.InvoiceStatus(status)
	invoice.Currency = currency
	invoice.PeriodStart = nanoToTime(periodStartNano)
	invoice.PeriodEnd = nanoToTime(periodEndNano)
	invoice.DueDate = nanoToTime(dueDateNano)
	invoice.IssuedAt = nanoToTime(issuedAtNano)
	invoice.DiscardedAt = nanoToTime(discardedAtNano)
	invoice.CreatedAt = nanoToTime(createdAtNano)
	invoice.UpdatedAt = nanoToTime(updatedAtNano)

	// Fetch lines.
	rows, err := s.db.QueryContext(ctx, `
SELECT id, invoice_id, service_agreement_id, time_entry_id, description, quantity_min, unit_rate_amount, unit_rate_currency
FROM invoice_lines WHERE invoice_id = ?`, id)
	if err != nil {
		return nil, fmt.Errorf("get invoice lines: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var line core.InvoiceLine
		var rateAmount int64
		var rateCurrency string
		var timeEntryID sql.NullString
		if err := rows.Scan(&line.ID, &line.InvoiceID, &line.ServiceAgreementID, &timeEntryID, &line.Description, &line.QuantityMin, &rateAmount, &rateCurrency); err != nil {
			return nil, fmt.Errorf("scan invoice line: %w", err)
		}
		if timeEntryID.Valid {
			line.TimeEntryID = timeEntryID.String
		}
		line.UnitRate = core.Money{Amount: rateAmount, Currency: rateCurrency}
		invoice.Lines = append(invoice.Lines, line)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate invoice lines: %w", err)
	}

	return &invoice, nil
}

func (s *InvoiceStore) AddLine(ctx context.Context, invoiceID string, line core.InvoiceLine) error {
	if s == nil || s.db == nil {
		return errors.New("invoice sqlite store is required")
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO invoice_lines (id, invoice_id, service_agreement_id, time_entry_id, description, quantity_min, unit_rate_amount, unit_rate_currency)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		line.ID,
		invoiceID,
		line.ServiceAgreementID,
		nullString(line.TimeEntryID),
		line.Description,
		line.QuantityMin,
		line.UnitRate.Amount,
		line.UnitRate.Currency,
	)
	if err != nil {
		return fmt.Errorf("insert invoice line: %w", err)
	}
	return nil
}

func (s *InvoiceStore) RemoveLine(ctx context.Context, invoiceID, lineID string) error {
	if s == nil || s.db == nil {
		return errors.New("invoice sqlite store is required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	var timeEntryID sql.NullString
	err = tx.QueryRowContext(ctx, `SELECT time_entry_id FROM invoice_lines WHERE invoice_id = ? AND id = ?`, invoiceID, lineID).Scan(&timeEntryID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return app.ErrInvoiceNotFound
		}
		return fmt.Errorf("select invoice line: %w", err)
	}
	if timeEntryID.Valid && timeEntryID.String != "" {
		_, err = tx.ExecContext(ctx, `UPDATE time_entries SET invoice_id = NULL, updated_at = ? WHERE id = ?`, time.Now().UTC().UnixNano(), timeEntryID.String)
		if err != nil {
			return fmt.Errorf("unlock time entry: %w", err)
		}
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM invoice_lines WHERE invoice_id = ? AND id = ?`, invoiceID, lineID); err != nil {
		return fmt.Errorf("delete invoice line: %w", err)
	}
	return tx.Commit()
}

// Update updates an invoice's status and timestamps (used for issuing and soft-discarding).
func (s *InvoiceStore) Update(ctx context.Context, invoice *core.Invoice) error {
	if s == nil || s.db == nil {
		return errors.New("invoice sqlite store is required")
	}
	if invoice == nil {
		return errors.New("invoice is required")
	}

	_, err := s.db.ExecContext(ctx, `
UPDATE invoices SET
	invoice_number = ?, status = ?, period_start = ?, period_end = ?, due_date = ?, notes = ?, issued_at = ?, discarded_at = ?, updated_at = ?
WHERE id = ?`,
		invoice.InvoiceNumber,
		string(invoice.Status),
		timeToNano(invoice.PeriodStart),
		timeToNano(invoice.PeriodEnd),
		timeToNano(invoice.DueDate),
		invoice.Notes,
		timeToNano(invoice.IssuedAt),
		timeToNano(invoice.DiscardedAt),
		invoice.UpdatedAt.UTC().UnixNano(),
		invoice.ID,
	)
	if err != nil {
		return fmt.Errorf("update invoice: %w", err)
	}
	return nil
}

// Delete hard-deletes an invoice, its lines, and unlocks associated time entries
// in a single transaction. Either all operations succeed or none do.
func (s *InvoiceStore) Delete(ctx context.Context, id string) error {
	if s == nil || s.db == nil {
		return errors.New("invoice sqlite store is required")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Unlock time entries (set invoice_id to NULL).
	_, err = tx.ExecContext(ctx, `UPDATE time_entries SET invoice_id = NULL, updated_at = ? WHERE invoice_id = ?`,
		time.Now().UTC().UnixNano(), id)
	if err != nil {
		return fmt.Errorf("unlock time entries: %w", err)
	}

	// Delete invoice (CASCADE removes invoice_lines).
	_, err = tx.ExecContext(ctx, `DELETE FROM invoices WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete invoice: %w", err)
	}

	return tx.Commit()
}

// ListByCustomer returns summary-only invoices for a customer, computing grand_total
// via a correlated aggregate subquery. An optional status filter can be applied.
func (s *InvoiceStore) ListByCustomer(ctx context.Context, customerID string, status ...core.InvoiceStatus) ([]core.InvoiceSummary, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("invoice sqlite store is required")
	}

	query := `
SELECT i.id, i.invoice_number, i.customer_id, i.status, i.currency, i.period_start, i.period_end, i.due_date, i.created_at,
       COALESCE((SELECT SUM(il.unit_rate_amount * il.quantity_min / 60)
                 FROM invoice_lines il
                 WHERE il.invoice_id = i.id), 0) AS grand_total
FROM invoices i
WHERE i.customer_id = ?`

	args := []interface{}{customerID}
	if len(status) > 0 {
		query += " AND i.status = ?"
		args = append(args, string(status[0]))
	}
	query += " ORDER BY i.created_at DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list invoices by customer: %w", err)
	}
	defer rows.Close()

	var summaries []core.InvoiceSummary
	for rows.Next() {
		var sum core.InvoiceSummary
		var periodStartNano, periodEndNano, dueDateNano, createdAtNano sql.NullInt64
		var status string
		if err := rows.Scan(&sum.ID, &sum.InvoiceNumber, &sum.CustomerID, &status, &sum.Currency, &periodStartNano, &periodEndNano, &dueDateNano, &createdAtNano, &sum.GrandTotal); err != nil {
			return nil, fmt.Errorf("scan invoice summary: %w", err)
		}
		sum.Status = core.InvoiceStatus(status)
		sum.PeriodStart = nanoToTime(periodStartNano)
		sum.PeriodEnd = nanoToTime(periodEndNano)
		sum.DueDate = nanoToTime(dueDateNano)
		sum.CreatedAt = nanoToTime(createdAtNano)
		summaries = append(summaries, sum)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate invoice summaries: %w", err)
	}

	return summaries, nil
}

func nullString(value string) interface{} {
	if value == "" {
		return nil
	}
	return value
}

func timeToNano(t time.Time) interface{} {
	if t.IsZero() {
		return nil
	}
	return t.UTC().UnixNano()
}

func nanoToTime(ns sql.NullInt64) time.Time {
	if !ns.Valid {
		return time.Time{}
	}
	return time.Unix(0, ns.Int64).UTC()
}
