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

type CustomerStore struct {
	db *sql.DB
}

func NewCustomerStore(store *Store) *CustomerStore {
	if store == nil {
		return nil
	}
	return &CustomerStore{db: store.DB()}
}

func (s *CustomerStore) List(ctx context.Context, query app.ListQuery) (app.ListResult[core.Customer], error) {
	if s == nil || s.db == nil {
		return app.ListResult[core.Customer]{}, errors.New("customer sqlite store is required")
	}

	query = query.Normalize()

	whereClause, args := customerSearchClause(query.Search)
	countQuery := "SELECT COUNT(*) FROM customers" + whereClause
	var total int
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return app.ListResult[core.Customer]{}, fmt.Errorf("count customers: %w", err)
	}

	selectQuery := strings.Builder{}
	selectQuery.WriteString("SELECT id, type, legal_name, trade_name, tax_id, email, phone, website, billing_address, status, default_currency, notes, created_at, updated_at FROM customers")
	selectQuery.WriteString(whereClause)
	selectQuery.WriteString(" ORDER BY ")
	selectQuery.WriteString(customerSortColumn(query.SortField))
	selectQuery.WriteByte(' ')
	selectQuery.WriteString(customerSortDirection(query.SortDir))
	selectQuery.WriteString(" LIMIT ? OFFSET ?")

	rows, err := s.db.QueryContext(ctx, selectQuery.String(), append(args, query.PageSize, customerOffset(query.Page, query.PageSize))...)
	if err != nil {
		return app.ListResult[core.Customer]{}, fmt.Errorf("list customers: %w", err)
	}
	defer rows.Close()

	items := make([]core.Customer, 0)
	for rows.Next() {
		customer, err := scanCustomer(rows)
		if err != nil {
			return app.ListResult[core.Customer]{}, err
		}
		items = append(items, customer)
	}
	if err := rows.Err(); err != nil {
		return app.ListResult[core.Customer]{}, fmt.Errorf("iterate customers: %w", err)
	}

	return app.NewListResult(query, items, total), nil
}

func customerSearchClause(search string) (string, []any) {
	search = strings.TrimSpace(search)
	if search == "" {
		return "", nil
	}

	return " WHERE legal_name LIKE ? COLLATE NOCASE", []any{"%" + search + "%"}
}

func customerSortColumn(field string) string {
	switch strings.TrimSpace(strings.ToLower(field)) {
	case "created_at":
		return "created_at"
	case "legal_name":
		return "legal_name"
	case "trade_name":
		return "trade_name"
	case "status":
		return "status"
	case "email":
		return "email"
	case "type":
		return "type"
	default:
		return "created_at"
	}
}

func customerSortDirection(dir string) string {
	if strings.EqualFold(strings.TrimSpace(dir), "desc") {
		return "DESC"
	}
	return "ASC"
}

func customerOffset(page, pageSize int) int {
	if page <= 1 {
		return 0
	}
	return (page - 1) * pageSize
}

type customerRowScanner interface {
	Scan(dest ...any) error
}

func scanCustomer(row customerRowScanner) (core.Customer, error) {
	var customer core.Customer
	var billing string
	var createdAt int64
	var updatedAt int64

	if err := row.Scan(
		&customer.ID,
		&customer.Type,
		&customer.LegalName,
		&customer.TradeName,
		&customer.TaxID,
		&customer.Email,
		&customer.Phone,
		&customer.Website,
		&billing,
		&customer.Status,
		&customer.DefaultCurrency,
		&customer.Notes,
		&createdAt,
		&updatedAt,
	); err != nil {
		return core.Customer{}, fmt.Errorf("scan customer: %w", err)
	}

	if strings.TrimSpace(billing) != "" {
		if err := json.Unmarshal([]byte(billing), &customer.BillingAddress); err != nil {
			return core.Customer{}, fmt.Errorf("decode customer billing address: %w", err)
		}
	}

	customer.CreatedAt = time.Unix(0, createdAt).UTC()
	customer.UpdatedAt = time.Unix(0, updatedAt).UTC()

	return customer, nil
}
