package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Carlos0934/billar/internal/app"
	"github.com/Carlos0934/billar/internal/core"
)

type LegalEntityStore struct {
	db *sql.DB
}

func NewLegalEntityStore(store *Store) *LegalEntityStore {
	if store == nil {
		return nil
	}
	return &LegalEntityStore{db: store.DB()}
}

func (s *LegalEntityStore) List(ctx context.Context, query app.ListQuery) (app.ListResult[core.LegalEntity], error) {
	if s == nil || s.db == nil {
		return app.ListResult[core.LegalEntity]{}, errors.New("legal entity sqlite store is required")
	}

	query = query.Normalize()

	whereClause, args := legalEntitySearchClause(query.Search)
	countQuery := "SELECT COUNT(*) FROM legal_entities" + whereClause
	var total int
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return app.ListResult[core.LegalEntity]{}, fmt.Errorf("count legal entities: %w", err)
	}

	selectQuery := strings.Builder{}
	selectQuery.WriteString("SELECT id, type, legal_name, trade_name, tax_id, email, phone, website, billing_address, created_at, updated_at FROM legal_entities")
	selectQuery.WriteString(whereClause)
	selectQuery.WriteString(" ORDER BY ")
	selectQuery.WriteString(legalEntitySortColumn(query.SortField))
	selectQuery.WriteByte(' ')
	selectQuery.WriteString(legalEntitySortDirection(query.SortDir))
	selectQuery.WriteString(" LIMIT ? OFFSET ?")

	rows, err := s.db.QueryContext(ctx, selectQuery.String(), append(args, query.PageSize, legalEntityOffset(query.Page, query.PageSize))...)
	if err != nil {
		return app.ListResult[core.LegalEntity]{}, fmt.Errorf("list legal entities: %w", err)
	}
	defer rows.Close()

	items := make([]core.LegalEntity, 0)
	for rows.Next() {
		entity, err := scanLegalEntity(rows)
		if err != nil {
			return app.ListResult[core.LegalEntity]{}, err
		}
		items = append(items, entity)
	}
	if err := rows.Err(); err != nil {
		return app.ListResult[core.LegalEntity]{}, fmt.Errorf("iterate legal entities: %w", err)
	}

	return app.NewListResult(query, items, total), nil
}

func legalEntitySearchClause(search string) (string, []any) {
	search = strings.TrimSpace(search)
	if search == "" {
		return "", nil
	}

	return " WHERE legal_name LIKE ? COLLATE NOCASE", []any{"%" + search + "%"}
}

func legalEntitySortColumn(field string) string {
	switch strings.TrimSpace(strings.ToLower(field)) {
	case "created_at":
		return "created_at"
	case "legal_name":
		return "legal_name"
	case "trade_name":
		return "trade_name"
	case "email":
		return "email"
	case "type":
		return "type"
	default:
		return "created_at"
	}
}

func legalEntitySortDirection(dir string) string {
	if strings.EqualFold(strings.TrimSpace(dir), "desc") {
		return "DESC"
	}
	return "ASC"
}

func legalEntityOffset(page, pageSize int) int {
	if page <= 1 {
		return 0
	}
	return (page - 1) * pageSize
}

type legalEntityRowScanner interface {
	Scan(dest ...any) error
}

func scanLegalEntity(row legalEntityRowScanner) (core.LegalEntity, error) {
	var entity core.LegalEntity
	var billing string
	var createdAt int64
	var updatedAt int64

	if err := row.Scan(
		&entity.ID,
		&entity.Type,
		&entity.LegalName,
		&entity.TradeName,
		&entity.TaxID,
		&entity.Email,
		&entity.Phone,
		&entity.Website,
		&billing,
		&createdAt,
		&updatedAt,
	); err != nil {
		return core.LegalEntity{}, fmt.Errorf("scan legal entity: %w", err)
	}

	if strings.TrimSpace(billing) != "" {
		if err := json.Unmarshal([]byte(billing), &entity.BillingAddress); err != nil {
			return core.LegalEntity{}, fmt.Errorf("decode legal entity billing address: %w", err)
		}
	}

	entity.CreatedAt = time.Unix(0, createdAt).UTC()
	entity.UpdatedAt = time.Unix(0, updatedAt).UTC()

	return entity, nil
}

func (s *LegalEntityStore) Save(ctx context.Context, entity *core.LegalEntity) error {
	if s == nil || s.db == nil {
		return errors.New("legal entity sqlite store is required")
	}
	if entity == nil {
		return errors.New("legal entity is required")
	}

	billing, err := json.Marshal(entity.BillingAddress)
	if err != nil {
		return fmt.Errorf("encode billing address: %w", err)
	}

	// Check if entity already exists to avoid INSERT OR REPLACE which triggers cascade deletes
	var exists bool
	err = s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM legal_entities WHERE id = ?)", entity.ID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check existence: %w", err)
	}

	if exists {
		// Use UPDATE for existing entities to preserve linked profiles
		_, err = s.db.ExecContext(ctx, `
UPDATE legal_entities SET
	type = ?, legal_name = ?, trade_name = ?, tax_id = ?, email = ?, phone = ?, website = ?, billing_address = ?, updated_at = ?
WHERE id = ?`,
			string(entity.Type),
			entity.LegalName,
			entity.TradeName,
			entity.TaxID,
			entity.Email,
			entity.Phone,
			entity.Website,
			string(billing),
			entity.UpdatedAt.UTC().UnixNano(),
			entity.ID,
		)
		if err != nil {
			return fmt.Errorf("update legal entity: %w", err)
		}
	} else {
		// Use INSERT for new entities
		_, err = s.db.ExecContext(ctx, `
INSERT INTO legal_entities (
	id, type, legal_name, trade_name, tax_id, email, phone, website, billing_address, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			entity.ID,
			string(entity.Type),
			entity.LegalName,
			entity.TradeName,
			entity.TaxID,
			entity.Email,
			entity.Phone,
			entity.Website,
			string(billing),
			entity.CreatedAt.UTC().UnixNano(),
			entity.UpdatedAt.UTC().UnixNano(),
		)
		if err != nil {
			return fmt.Errorf("insert legal entity: %w", err)
		}
	}

	return nil
}

func (s *LegalEntityStore) GetByID(ctx context.Context, id string) (*core.LegalEntity, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("legal entity sqlite store is required")
	}

	query := `SELECT id, type, legal_name, trade_name, tax_id, email, phone, website, billing_address, created_at, updated_at FROM legal_entities WHERE id = ?`
	row := s.db.QueryRowContext(ctx, query, id)

	entity, err := scanLegalEntity(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, app.ErrLegalEntityNotFound
		}
		return nil, fmt.Errorf("get legal entity by id: %w", err)
	}

	return &entity, nil
}

func (s *LegalEntityStore) Delete(ctx context.Context, id string) error {
	if s == nil || s.db == nil {
		return errors.New("legal entity sqlite store is required")
	}

	result, err := s.db.ExecContext(ctx, "DELETE FROM legal_entities WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete legal entity: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return app.ErrLegalEntityNotFound
	}

	return nil
}
