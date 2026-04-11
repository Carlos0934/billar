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

// ServiceAgreementStore persists and retrieves ServiceAgreement entities via SQLite.
type ServiceAgreementStore struct {
	db *sql.DB
}

// NewServiceAgreementStore constructs a ServiceAgreementStore from an open Store.
// Returns nil when store is nil, consistent with other sqlite store constructors.
func NewServiceAgreementStore(store *Store) *ServiceAgreementStore {
	if store == nil {
		return nil
	}
	return &ServiceAgreementStore{db: store.DB()}
}

// Save upserts a ServiceAgreement: inserts on first call, updates on subsequent calls.
func (s *ServiceAgreementStore) Save(ctx context.Context, sa *core.ServiceAgreement) error {
	if s == nil || s.db == nil {
		return errors.New("service agreement sqlite store is required")
	}
	if sa == nil {
		return errors.New("service agreement is required")
	}

	var exists bool
	err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM service_agreements WHERE id = ?)", sa.ID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check existence: %w", err)
	}

	active := 0
	if sa.Active {
		active = 1
	}

	var validFrom, validUntil interface{}
	if sa.ValidFrom != nil {
		validFrom = sa.ValidFrom.UTC().UnixNano()
	}
	if sa.ValidUntil != nil {
		validUntil = sa.ValidUntil.UTC().UnixNano()
	}

	if exists {
		_, err = s.db.ExecContext(ctx, `
UPDATE service_agreements SET
	name = ?, description = ?, billing_mode = ?, hourly_rate = ?,
	currency = ?, active = ?, valid_from = ?, valid_until = ?, updated_at = ?
WHERE id = ?`,
			sa.Name, sa.Description, string(sa.BillingMode), sa.HourlyRate,
			sa.Currency, active, validFrom, validUntil,
			sa.UpdatedAt.UTC().UnixNano(),
			sa.ID,
		)
		if err != nil {
			return fmt.Errorf("update service agreement: %w", err)
		}
	} else {
		_, err = s.db.ExecContext(ctx, `
INSERT INTO service_agreements
  (id, customer_profile_id, name, description, billing_mode, hourly_rate, currency, active, valid_from, valid_until, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			sa.ID, sa.CustomerProfileID, sa.Name, sa.Description,
			string(sa.BillingMode), sa.HourlyRate, sa.Currency, active,
			validFrom, validUntil,
			sa.CreatedAt.UTC().UnixNano(), sa.UpdatedAt.UTC().UnixNano(),
		)
		if err != nil {
			return fmt.Errorf("insert service agreement: %w", err)
		}
	}

	return nil
}

// GetByID fetches a single ServiceAgreement by its ID.
// Returns app.ErrServiceAgreementNotFound when no row matches.
func (s *ServiceAgreementStore) GetByID(ctx context.Context, id string) (*core.ServiceAgreement, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("service agreement sqlite store is required")
	}

	row := s.db.QueryRowContext(ctx, `
SELECT id, customer_profile_id, name, description, billing_mode, hourly_rate,
       currency, active, valid_from, valid_until, created_at, updated_at
FROM service_agreements
WHERE id = ?`, id)

	sa, err := scanServiceAgreement(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, app.ErrServiceAgreementNotFound
		}
		return nil, fmt.Errorf("get service agreement by id: %w", err)
	}

	return &sa, nil
}

// ListByCustomerProfileID returns all agreements for the given customer profile,
// ordered by created_at ascending.
func (s *ServiceAgreementStore) ListByCustomerProfileID(ctx context.Context, customerProfileID string) ([]core.ServiceAgreement, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("service agreement sqlite store is required")
	}

	rows, err := s.db.QueryContext(ctx, `
SELECT id, customer_profile_id, name, description, billing_mode, hourly_rate,
       currency, active, valid_from, valid_until, created_at, updated_at
FROM service_agreements
WHERE customer_profile_id = ?
ORDER BY created_at ASC`, customerProfileID)
	if err != nil {
		return nil, fmt.Errorf("list service agreements: %w", err)
	}
	defer rows.Close()

	items := make([]core.ServiceAgreement, 0)
	for rows.Next() {
		sa, err := scanServiceAgreement(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, sa)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate service agreements: %w", err)
	}

	return items, nil
}

// serviceAgreementRowScanner abstracts *sql.Row and *sql.Rows for scanServiceAgreement.
type serviceAgreementRowScanner interface {
	Scan(dest ...any) error
}

func scanServiceAgreement(row serviceAgreementRowScanner) (core.ServiceAgreement, error) {
	var sa core.ServiceAgreement
	var active int
	var validFromNano, validUntilNano sql.NullInt64
	var createdAtNano, updatedAtNano int64
	var billingMode string

	if err := row.Scan(
		&sa.ID, &sa.CustomerProfileID, &sa.Name, &sa.Description,
		&billingMode, &sa.HourlyRate, &sa.Currency, &active,
		&validFromNano, &validUntilNano,
		&createdAtNano, &updatedAtNano,
	); err != nil {
		return core.ServiceAgreement{}, err
	}

	sa.BillingMode = core.BillingMode(billingMode)
	sa.Active = active != 0
	sa.CreatedAt = time.Unix(0, createdAtNano).UTC()
	sa.UpdatedAt = time.Unix(0, updatedAtNano).UTC()

	if validFromNano.Valid {
		t := time.Unix(0, validFromNano.Int64).UTC()
		sa.ValidFrom = &t
	}
	if validUntilNano.Valid {
		t := time.Unix(0, validUntilNano.Int64).UTC()
		sa.ValidUntil = &t
	}

	return sa, nil
}
