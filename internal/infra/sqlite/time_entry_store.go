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

// TimeEntryStore persists and retrieves TimeEntry entities via SQLite.
// The time_entries table does NOT store customer_profile_id; it is always
// derived at read time via JOIN on service_agreements.
type TimeEntryStore struct {
	db *sql.DB
}

// NewTimeEntryStore constructs a TimeEntryStore from an open Store.
// Returns nil when store is nil, consistent with other sqlite store constructors.
func NewTimeEntryStore(store *Store) *TimeEntryStore {
	if store == nil {
		return nil
	}
	return &TimeEntryStore{db: store.DB()}
}

// Save upserts a TimeEntry: inserts on first call, updates on subsequent calls.
// CustomerProfileID is NOT persisted; it is carried on the entity but stored
// only in service_agreements.
func (s *TimeEntryStore) Save(ctx context.Context, entry *core.TimeEntry) error {
	if s == nil || s.db == nil {
		return errors.New("time entry sqlite store is required")
	}
	if entry == nil {
		return errors.New("time entry is required")
	}

	var exists bool
	err := s.db.QueryRowContext(ctx,
		"SELECT EXISTS(SELECT 1 FROM time_entries WHERE id = ?)", entry.ID,
	).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check time entry existence: %w", err)
	}

	billable := 0
	if entry.Billable {
		billable = 1
	}

	var invoiceID interface{}
	if entry.InvoiceID != "" {
		invoiceID = entry.InvoiceID
	}

	if exists {
		_, err = s.db.ExecContext(ctx, `
UPDATE time_entries SET
	description = ?, hours = ?, billable = ?, invoice_id = ?,
	date = ?, updated_at = ?
WHERE id = ?`,
			entry.Description,
			int64(entry.Hours),
			billable,
			invoiceID,
			entry.Date.UTC().UnixNano(),
			entry.UpdatedAt.UTC().UnixNano(),
			entry.ID,
		)
		if err != nil {
			return fmt.Errorf("update time entry: %w", err)
		}
	} else {
		_, err = s.db.ExecContext(ctx, `
INSERT INTO time_entries
  (id, service_agreement_id, description, hours, billable, invoice_id, date, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			entry.ID,
			entry.ServiceAgreementID,
			entry.Description,
			int64(entry.Hours),
			billable,
			invoiceID,
			entry.Date.UTC().UnixNano(),
			entry.CreatedAt.UTC().UnixNano(),
			entry.UpdatedAt.UTC().UnixNano(),
		)
		if err != nil {
			return fmt.Errorf("insert time entry: %w", err)
		}
	}

	return nil
}

// GetByID fetches a single TimeEntry by its ID.
// CustomerProfileID is populated via JOIN on service_agreements.
// Returns app.ErrTimeEntryNotFound when no row matches.
func (s *TimeEntryStore) GetByID(ctx context.Context, id string) (*core.TimeEntry, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("time entry sqlite store is required")
	}

	row := s.db.QueryRowContext(ctx, `
SELECT t.id, t.service_agreement_id, sa.customer_profile_id,
       t.description, t.hours, t.billable, t.invoice_id,
       t.date, t.created_at, t.updated_at
FROM time_entries t
JOIN service_agreements sa ON t.service_agreement_id = sa.id
WHERE t.id = ?`, id)

	entry, err := scanTimeEntry(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, app.ErrTimeEntryNotFound
		}
		return nil, fmt.Errorf("get time entry by id: %w", err)
	}

	return &entry, nil
}

// Delete removes a TimeEntry by ID.
func (s *TimeEntryStore) Delete(ctx context.Context, id string) error {
	if s == nil || s.db == nil {
		return errors.New("time entry sqlite store is required")
	}

	_, err := s.db.ExecContext(ctx, "DELETE FROM time_entries WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete time entry: %w", err)
	}
	return nil
}

// ListByCustomerProfile returns all time entries for the given customer profile,
// ordered by date ascending. CustomerProfileID is JOIN-derived.
func (s *TimeEntryStore) ListByCustomerProfile(ctx context.Context, customerID string) ([]core.TimeEntry, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("time entry sqlite store is required")
	}

	rows, err := s.db.QueryContext(ctx, `
SELECT t.id, t.service_agreement_id, sa.customer_profile_id,
       t.description, t.hours, t.billable, t.invoice_id,
       t.date, t.created_at, t.updated_at
FROM time_entries t
JOIN service_agreements sa ON t.service_agreement_id = sa.id
WHERE sa.customer_profile_id = ?
ORDER BY t.date ASC`, customerID)
	if err != nil {
		return nil, fmt.Errorf("list time entries by customer profile: %w", err)
	}
	defer rows.Close()

	items := make([]core.TimeEntry, 0)
	for rows.Next() {
		entry, err := scanTimeEntry(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate time entries by customer profile: %w", err)
	}

	return items, nil
}

// ListUnbilled returns all time entries with no assigned invoice for the given customer profile,
// ordered by date ascending. CustomerProfileID is JOIN-derived.
func (s *TimeEntryStore) ListUnbilled(ctx context.Context, customerID string) ([]core.TimeEntry, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("time entry sqlite store is required")
	}

	rows, err := s.db.QueryContext(ctx, `
SELECT t.id, t.service_agreement_id, sa.customer_profile_id,
       t.description, t.hours, t.billable, t.invoice_id,
       t.date, t.created_at, t.updated_at
FROM time_entries t
JOIN service_agreements sa ON t.service_agreement_id = sa.id
WHERE sa.customer_profile_id = ? AND t.invoice_id IS NULL
ORDER BY t.date ASC`, customerID)
	if err != nil {
		return nil, fmt.Errorf("list unbilled time entries: %w", err)
	}
	defer rows.Close()

	items := make([]core.TimeEntry, 0)
	for rows.Next() {
		entry, err := scanTimeEntry(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate unbilled time entries: %w", err)
	}

	return items, nil
}

// timeEntryRowScanner abstracts *sql.Row and *sql.Rows for scanTimeEntry.
type timeEntryRowScanner interface {
	Scan(dest ...any) error
}

func scanTimeEntry(row timeEntryRowScanner) (core.TimeEntry, error) {
	var entry core.TimeEntry
	var hoursMinutes int64
	var billable int
	var invoiceID sql.NullString
	var dateNano, createdAtNano, updatedAtNano int64

	if err := row.Scan(
		&entry.ID,
		&entry.ServiceAgreementID,
		&entry.CustomerProfileID,
		&entry.Description,
		&hoursMinutes,
		&billable,
		&invoiceID,
		&dateNano,
		&createdAtNano,
		&updatedAtNano,
	); err != nil {
		return core.TimeEntry{}, err
	}

	hours, err := core.NewHours(hoursMinutes)
	if err != nil {
		return core.TimeEntry{}, fmt.Errorf("scan time entry hours: %w", err)
	}

	entry.Hours = hours
	entry.Billable = billable != 0
	entry.Date = time.Unix(0, dateNano).UTC()
	entry.CreatedAt = time.Unix(0, createdAtNano).UTC()
	entry.UpdatedAt = time.Unix(0, updatedAtNano).UTC()

	if invoiceID.Valid {
		entry.InvoiceID = invoiceID.String
	}

	return entry, nil
}
