package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Carlos0934/billar/internal/app"
	"github.com/Carlos0934/billar/internal/core"
)

type CustomerProfileStore struct {
	db *sql.DB
}

func NewCustomerProfileStore(store *Store) *CustomerProfileStore {
	if store == nil {
		return nil
	}
	return &CustomerProfileStore{db: store.DB()}
}

func (s *CustomerProfileStore) List(ctx context.Context, query app.ListQuery) (app.ListResult[core.CustomerProfile], error) {
	if s == nil || s.db == nil {
		return app.ListResult[core.CustomerProfile]{}, errors.New("customer profile sqlite store is required")
	}

	query = query.Normalize()

	whereClause, args := customerProfileSearchClause(query.Search)
	countQuery := "SELECT COUNT(*) FROM customer_profiles" + whereClause
	var total int
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return app.ListResult[core.CustomerProfile]{}, fmt.Errorf("count customer profiles: %w", err)
	}

	selectQuery := strings.Builder{}
	selectQuery.WriteString("SELECT id, legal_entity_id, status, default_currency, notes, created_at, updated_at FROM customer_profiles")
	selectQuery.WriteString(whereClause)
	selectQuery.WriteString(" ORDER BY ")
	selectQuery.WriteString(customerProfileSortColumn(query.SortField))
	selectQuery.WriteByte(' ')
	selectQuery.WriteString(customerProfileSortDirection(query.SortDir))
	selectQuery.WriteString(" LIMIT ? OFFSET ?")

	rows, err := s.db.QueryContext(ctx, selectQuery.String(), append(args, query.PageSize, customerProfileOffset(query.Page, query.PageSize))...)
	if err != nil {
		return app.ListResult[core.CustomerProfile]{}, fmt.Errorf("list customer profiles: %w", err)
	}
	defer rows.Close()

	items := make([]core.CustomerProfile, 0)
	for rows.Next() {
		profile, err := scanCustomerProfile(rows)
		if err != nil {
			return app.ListResult[core.CustomerProfile]{}, err
		}
		items = append(items, profile)
	}
	if err := rows.Err(); err != nil {
		return app.ListResult[core.CustomerProfile]{}, fmt.Errorf("iterate customer profiles: %w", err)
	}

	return app.NewListResult(query, items, total), nil
}

func customerProfileSearchClause(search string) (string, []any) {
	search = strings.TrimSpace(search)
	if search == "" {
		return "", nil
	}

	// Search by joining with legal_entities on legal_name
	return ` WHERE legal_entity_id IN (SELECT id FROM legal_entities WHERE legal_name LIKE ? COLLATE NOCASE)`, []any{"%" + search + "%"}
}

func customerProfileSortColumn(field string) string {
	switch strings.TrimSpace(strings.ToLower(field)) {
	case "created_at":
		return "created_at"
	case "status":
		return "status"
	case "default_currency":
		return "default_currency"
	default:
		return "created_at"
	}
}

func customerProfileSortDirection(dir string) string {
	if strings.EqualFold(strings.TrimSpace(dir), "desc") {
		return "DESC"
	}
	return "ASC"
}

func customerProfileOffset(page, pageSize int) int {
	if page <= 1 {
		return 0
	}
	return (page - 1) * pageSize
}

type customerProfileRowScanner interface {
	Scan(dest ...any) error
}

func scanCustomerProfile(row customerProfileRowScanner) (core.CustomerProfile, error) {
	var profile core.CustomerProfile
	var createdAt int64
	var updatedAt int64

	if err := row.Scan(
		&profile.ID,
		&profile.LegalEntityID,
		&profile.Status,
		&profile.DefaultCurrency,
		&profile.Notes,
		&createdAt,
		&updatedAt,
	); err != nil {
		return core.CustomerProfile{}, fmt.Errorf("scan customer profile: %w", err)
	}

	profile.CreatedAt = time.Unix(0, createdAt).UTC()
	profile.UpdatedAt = time.Unix(0, updatedAt).UTC()

	return profile, nil
}

func (s *CustomerProfileStore) Save(ctx context.Context, profile *core.CustomerProfile) error {
	if s == nil || s.db == nil {
		return errors.New("customer profile sqlite store is required")
	}
	if profile == nil {
		return errors.New("customer profile is required")
	}

	// Check if profile already exists to avoid INSERT OR REPLACE which bypasses UNIQUE constraint
	var exists bool
	err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM customer_profiles WHERE id = ?)", profile.ID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check existence: %w", err)
	}

	if exists {
		// Use UPDATE for existing profiles - preserves legal_entity_id and created_at
		_, err = s.db.ExecContext(ctx, `
UPDATE customer_profiles SET
	status = ?, default_currency = ?, notes = ?, updated_at = ?
WHERE id = ?`,
			string(profile.Status),
			profile.DefaultCurrency,
			profile.Notes,
			profile.UpdatedAt.UTC().UnixNano(),
			profile.ID,
		)
		if err != nil {
			return fmt.Errorf("update customer profile: %w", err)
		}
	} else {
		// Use INSERT for new profiles - will fail if legal_entity_id UNIQUE constraint is violated
		_, err = s.db.ExecContext(ctx, `
INSERT INTO customer_profiles (
	id, legal_entity_id, status, default_currency, notes, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			profile.ID,
			profile.LegalEntityID,
			string(profile.Status),
			profile.DefaultCurrency,
			profile.Notes,
			profile.CreatedAt.UTC().UnixNano(),
			profile.UpdatedAt.UTC().UnixNano(),
		)
		if err != nil {
			return fmt.Errorf("insert customer profile: %w", err)
		}
	}

	return nil
}

func (s *CustomerProfileStore) GetByID(ctx context.Context, id string) (*core.CustomerProfile, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("customer profile sqlite store is required")
	}

	query := `SELECT id, legal_entity_id, status, default_currency, notes, created_at, updated_at FROM customer_profiles WHERE id = ?`
	row := s.db.QueryRowContext(ctx, query, id)

	profile, err := scanCustomerProfile(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, app.ErrCustomerProfileNotFound
		}
		return nil, fmt.Errorf("get customer profile by id: %w", err)
	}

	return &profile, nil
}

func (s *CustomerProfileStore) Delete(ctx context.Context, id string) error {
	if s == nil || s.db == nil {
		return errors.New("customer profile sqlite store is required")
	}

	result, err := s.db.ExecContext(ctx, "DELETE FROM customer_profiles WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete customer profile: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return app.ErrCustomerProfileNotFound
	}

	return nil
}
