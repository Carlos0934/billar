package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/Carlos0934/billar/internal/app"
	"github.com/Carlos0934/billar/internal/core"
)

func TestCustomerStoreListReturnsEmptyResult(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	customerStore := NewCustomerStore(store)

	got, err := customerStore.List(context.Background(), app.ListQuery{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	want := app.ListResult[core.Customer]{Items: []core.Customer{}, Total: 0, Page: 1, PageSize: 20}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("List() = %+v, want %+v", got, want)
	}
}

func TestCustomerStoreListSearchesPaginatesAndSorts(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	customerStore := NewCustomerStore(store)

	insertCustomer(t, store.DB(), core.Customer{
		ID:              "cus_old",
		Type:            core.CustomerTypeCompany,
		LegalName:       "Acme Alpha",
		Status:          core.CustomerStatusActive,
		DefaultCurrency: "USD",
		CreatedAt:       time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC),
		UpdatedAt:       time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC),
	})
	insertCustomer(t, store.DB(), core.Customer{
		ID:              "cus_new",
		Type:            core.CustomerTypeCompany,
		LegalName:       "Acme Zeta",
		Status:          core.CustomerStatusActive,
		DefaultCurrency: "USD",
		CreatedAt:       time.Date(2026, 4, 3, 10, 5, 0, 0, time.UTC),
		UpdatedAt:       time.Date(2026, 4, 3, 10, 5, 0, 0, time.UTC),
	})
	insertCustomer(t, store.DB(), core.Customer{
		ID:              "cus_other",
		Type:            core.CustomerTypeIndividual,
		LegalName:       "Beta Other",
		Status:          core.CustomerStatusActive,
		DefaultCurrency: "USD",
		CreatedAt:       time.Date(2026, 4, 3, 10, 10, 0, 0, time.UTC),
		UpdatedAt:       time.Date(2026, 4, 3, 10, 10, 0, 0, time.UTC),
	})

	first, err := customerStore.List(context.Background(), app.ListQuery{Search: "  acme  ", Page: 1, PageSize: 1, SortField: "created_at", SortDir: "desc"})
	if err != nil {
		t.Fatalf("List() first page error = %v", err)
	}
	wantFirst := app.ListResult[core.Customer]{
		Items: []core.Customer{{
			ID:              "cus_new",
			Type:            core.CustomerTypeCompany,
			LegalName:       "Acme Zeta",
			Status:          core.CustomerStatusActive,
			DefaultCurrency: "USD",
			CreatedAt:       time.Date(2026, 4, 3, 10, 5, 0, 0, time.UTC),
			UpdatedAt:       time.Date(2026, 4, 3, 10, 5, 0, 0, time.UTC),
		}},
		Total:    2,
		Page:     1,
		PageSize: 1,
	}
	if !reflect.DeepEqual(first, wantFirst) {
		t.Fatalf("List() first page = %+v, want %+v", first, wantFirst)
	}

	second, err := customerStore.List(context.Background(), app.ListQuery{Search: "acme", Page: 2, PageSize: 1, SortField: "created_at", SortDir: "desc"})
	if err != nil {
		t.Fatalf("List() second page error = %v", err)
	}
	wantSecond := app.ListResult[core.Customer]{
		Items: []core.Customer{{
			ID:              "cus_old",
			Type:            core.CustomerTypeCompany,
			LegalName:       "Acme Alpha",
			Status:          core.CustomerStatusActive,
			DefaultCurrency: "USD",
			CreatedAt:       time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC),
			UpdatedAt:       time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC),
		}},
		Total:    2,
		Page:     2,
		PageSize: 1,
	}
	if !reflect.DeepEqual(second, wantSecond) {
		t.Fatalf("List() second page = %+v, want %+v", second, wantSecond)
	}

	byName, err := customerStore.List(context.Background(), app.ListQuery{Search: "acme", Page: 1, PageSize: 2, SortField: "name", SortDir: "asc"})
	if err != nil {
		t.Fatalf("List() by name error = %v", err)
	}
	wantByName := app.ListResult[core.Customer]{
		Items: []core.Customer{{
			ID:              "cus_old",
			Type:            core.CustomerTypeCompany,
			LegalName:       "Acme Alpha",
			Status:          core.CustomerStatusActive,
			DefaultCurrency: "USD",
			CreatedAt:       time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC),
			UpdatedAt:       time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC),
		}, {
			ID:              "cus_new",
			Type:            core.CustomerTypeCompany,
			LegalName:       "Acme Zeta",
			Status:          core.CustomerStatusActive,
			DefaultCurrency: "USD",
			CreatedAt:       time.Date(2026, 4, 3, 10, 5, 0, 0, time.UTC),
			UpdatedAt:       time.Date(2026, 4, 3, 10, 5, 0, 0, time.UTC),
		}},
		Total:    2,
		Page:     1,
		PageSize: 2,
	}
	if !reflect.DeepEqual(byName, wantByName) {
		t.Fatalf("List() by name = %+v, want %+v", byName, wantByName)
	}
}

func newTestStore(t *testing.T) *Store {
	t.Helper()

	store, err := Open(filepath.Join(t.TempDir(), "customers.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	})
	return store
}

func insertCustomer(t *testing.T, db *sql.DB, customer core.Customer) {
	t.Helper()

	billing, err := json.Marshal(customer.BillingAddress)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	_, err = db.ExecContext(context.Background(), `
INSERT INTO customers (
	id, type, legal_name, trade_name, tax_id, email, phone, website, billing_address, status, default_currency, notes, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		customer.ID,
		string(customer.Type),
		customer.LegalName,
		customer.TradeName,
		customer.TaxID,
		customer.Email,
		customer.Phone,
		customer.Website,
		string(billing),
		string(customer.Status),
		customer.DefaultCurrency,
		customer.Notes,
		customer.CreatedAt.UTC().UnixNano(),
		customer.UpdatedAt.UTC().UnixNano(),
	)
	if err != nil {
		t.Fatalf("ExecContext() error = %v", err)
	}
}
