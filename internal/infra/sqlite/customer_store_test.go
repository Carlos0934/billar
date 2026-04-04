package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
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

func TestCustomerStoreSaveInsertsNewCustomer(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	customerStore := NewCustomerStore(store)

	customer := core.Customer{
		Type:            core.CustomerTypeCompany,
		LegalName:       "Acme Corporation",
		TradeName:       "Acme",
		Email:           "contact@acme.com",
		Status:          core.CustomerStatusActive,
		DefaultCurrency: "USD",
	}

	if err := customerStore.Save(context.Background(), &customer); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify via List
	result, err := customerStore.List(context.Background(), app.ListQuery{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("List() total = %d, want 1", result.Total)
	}
	if len(result.Items) != 1 {
		t.Fatalf("List() items = %d, want 1", len(result.Items))
	}
	if result.Items[0].LegalName != customer.LegalName {
		t.Fatalf("List()[0].LegalName = %q, want %q", result.Items[0].LegalName, customer.LegalName)
	}
	if result.Items[0].ID != customer.ID {
		t.Fatalf("List()[0].ID = %q, want %q", result.Items[0].ID, customer.ID)
	}
}

func TestCustomerStoreSaveUpdatesExistingCustomer(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	customerStore := NewCustomerStore(store)

	customer := core.Customer{
		Type:            core.CustomerTypeCompany,
		LegalName:       "Acme Corporation",
		Email:           "old@acme.com",
		Status:          core.CustomerStatusActive,
		DefaultCurrency: "USD",
	}

	if err := customerStore.Save(context.Background(), &customer); err != nil {
		t.Fatalf("Save() first insert error = %v", err)
	}

	originalID := customer.ID
	customer.Email = "new@acme.com"
	customer.TradeName = "Acme Updated"

	if err := customerStore.Save(context.Background(), &customer); err != nil {
		t.Fatalf("Save() update error = %v", err)
	}

	// Verify the customer was updated, not duplicated
	result, err := customerStore.List(context.Background(), app.ListQuery{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("List() total = %d, want 1", result.Total)
	}
	if result.Items[0].ID != originalID {
		t.Fatalf("List()[0].ID = %q, want %q", result.Items[0].ID, originalID)
	}
	if result.Items[0].Email != "new@acme.com" {
		t.Fatalf("List()[0].Email = %q, want %q", result.Items[0].Email, "new@acme.com")
	}
	if result.Items[0].TradeName != "Acme Updated" {
		t.Fatalf("List()[0].TradeName = %q, want %q", result.Items[0].TradeName, "Acme Updated")
	}
}

func TestCustomerStoreGetByIDReturnsCorrectCustomer(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	customerStore := NewCustomerStore(store)

	inserted := core.Customer{
		ID:              "cus_test123",
		Type:            core.CustomerTypeCompany,
		LegalName:       "Test Company",
		Email:           "test@company.com",
		Status:          core.CustomerStatusActive,
		DefaultCurrency: "EUR",
	}
	insertCustomer(t, store.DB(), inserted)

	got, err := customerStore.GetByID(context.Background(), "cus_test123")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got == nil {
		t.Fatalf("GetByID() returned nil customer")
	}
	if got.ID != inserted.ID {
		t.Errorf("GetByID().ID = %q, want %q", got.ID, inserted.ID)
	}
	if got.LegalName != inserted.LegalName {
		t.Errorf("GetByID().LegalName = %q, want %q", got.LegalName, inserted.LegalName)
	}
	if got.Type != inserted.Type {
		t.Errorf("GetByID().Type = %q, want %q", got.Type, inserted.Type)
	}
	if got.Email != inserted.Email {
		t.Errorf("GetByID().Email = %q, want %q", got.Email, inserted.Email)
	}
	if got.Status != inserted.Status {
		t.Errorf("GetByID().Status = %q, want %q", got.Status, inserted.Status)
	}
}

func TestCustomerStoreGetByIDReturnsNotFoundForUnknownID(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	customerStore := NewCustomerStore(store)

	_, err := customerStore.GetByID(context.Background(), "cus_nonexistent")
	if err == nil {
		t.Fatalf("GetByID() expected error, got nil")
	}
	if !errors.Is(err, app.ErrCustomerNotFound) {
		t.Fatalf("GetByID() error = %v, want %v", err, app.ErrCustomerNotFound)
	}
}

func TestCustomerStoreDeleteRemovesCustomer(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	customerStore := NewCustomerStore(store)

	inserted := core.Customer{
		ID:              "cus_todelete",
		Type:            core.CustomerTypeIndividual,
		LegalName:       "To Delete",
		Status:          core.CustomerStatusActive,
		DefaultCurrency: "USD",
	}
	insertCustomer(t, store.DB(), inserted)

	if err := customerStore.Delete(context.Background(), "cus_todelete"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify via GetByID returns not found
	_, err := customerStore.GetByID(context.Background(), "cus_todelete")
	if err == nil {
		t.Fatalf("GetByID() expected error after delete, got nil")
	}
	if !errors.Is(err, app.ErrCustomerNotFound) {
		t.Fatalf("GetByID() error = %v, want %v", err, app.ErrCustomerNotFound)
	}

	// Verify via List returns empty
	result, err := customerStore.List(context.Background(), app.ListQuery{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if result.Total != 0 {
		t.Fatalf("List() total after delete = %d, want 0", result.Total)
	}
}

func TestCustomerStoreDeleteReturnsNotFoundForUnknownID(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	customerStore := NewCustomerStore(store)

	err := customerStore.Delete(context.Background(), "cus_nonexistent")
	if err == nil {
		t.Fatalf("Delete() expected error, got nil")
	}
	if !errors.Is(err, app.ErrCustomerNotFound) {
		t.Fatalf("Delete() error = %v, want %v", err, app.ErrCustomerNotFound)
	}
}

func TestCustomerStorePatchLeavesOtherFieldsUntouched(t *testing.T) {
	t.Parallel()

	// Setup: Create a customer with populated fields
	store := newTestStore(t)
	customerStore := NewCustomerStore(store)

	original := core.Customer{
		ID:              "cus_test_untouched",
		Type:            core.CustomerTypeCompany,
		LegalName:       "Original Legal Name",
		TradeName:       "Original Trade Name",
		Email:           "original@example.com",
		Phone:           "+1 555 0100",
		Website:         "https://original.example",
		BillingAddress:  core.Address{Street: "123 Main St", City: "Original City"},
		Status:          core.CustomerStatusActive,
		DefaultCurrency: "USD",
		Notes:           "Original notes that should remain untouched",
		CreatedAt:       time.Date(2026, 4, 4, 10, 0, 0, 0, time.UTC),
		UpdatedAt:       time.Date(2026, 4, 4, 10, 0, 0, 0, time.UTC),
	}
	insertCustomer(t, store.DB(), original)

	// Simulate a PATCH that updates only the email field
	email := "updated@example.com"
	patch := core.CustomerPatch{
		Email: &email,
	}

	// Retrieve, apply patch, and save
	customer, err := customerStore.GetByID(context.Background(), original.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	customer.ApplyPatch(patch)
	// Explicitly verify that ApplyPatch does NOT modify fields with nil pointers
	if customer.TradeName != original.TradeName {
		t.Fatalf("ApplyPatch modified TradeName (should be untouched), got = %q, want %q", customer.TradeName, original.TradeName)
	}
	if customer.Phone != original.Phone {
		t.Fatalf("ApplyPatch modified Phone (should be untouched), got = %q, want %q", customer.Phone, original.Phone)
	}
	if customer.Website != original.Website {
		t.Fatalf("ApplyPatch modified Website (should be untouched), got = %q, want %q", customer.Website, original.Website)
	}
	if customer.Notes != original.Notes {
		t.Fatalf("ApplyPatch modified Notes (should be untouched), got = %q, want %q", customer.Notes, original.Notes)
	}
	if customer.DefaultCurrency != original.DefaultCurrency {
		t.Fatalf("ApplyPatch modified DefaultCurrency (should be untouched), got = %q, want %q", customer.DefaultCurrency, original.DefaultCurrency)
	}

	// Validate after patch
	if err := customer.Validate(); err != nil {
		t.Fatalf("Validate() after patch error = %v", err)
	}

	if err := customerStore.Save(context.Background(), customer); err != nil {
		t.Fatalf("Save() after patch error = %v", err)
	}

	// Retrieve again and verify ALL untouched fields remain unchanged
	reloaded, err := customerStore.GetByID(context.Background(), original.ID)
	if err != nil {
		t.Fatalf("GetByID() after reload error = %v", err)
	}

	// Verify the patched field was updated
	if reloaded.Email != email {
		t.Fatalf("Email = %q, want %q (should be updated)", reloaded.Email, email)
	}

	// Verify ALL untouched fields remain unchanged through the full store round-trip
	if reloaded.LegalName != original.LegalName {
		t.Fatalf("LegalName = %q, want %q (should be untouched)", reloaded.LegalName, original.LegalName)
	}
	if reloaded.TradeName != original.TradeName {
		t.Fatalf("TradeName = %q, want %q (should be untouched)", reloaded.TradeName, original.TradeName)
	}
	if reloaded.Phone != original.Phone {
		t.Fatalf("Phone = %q, want %q (should be untouched)", reloaded.Phone, original.Phone)
	}
	if reloaded.Website != original.Website {
		t.Fatalf("Website = %q, want %q (should be untouched)", reloaded.Website, original.Website)
	}
	if reloaded.Notes != original.Notes {
		t.Fatalf("Notes = %q, want %q (should be untouched)", reloaded.Notes, original.Notes)
	}
	if reloaded.DefaultCurrency != original.DefaultCurrency {
		t.Fatalf("DefaultCurrency = %q, want %q (should be untouched)", reloaded.DefaultCurrency, original.DefaultCurrency)
	}
	if reloaded.BillingAddress.Street != original.BillingAddress.Street {
		t.Fatalf("BillingAddress.Street = %q, want %q (should be untouched)", reloaded.BillingAddress.Street, original.BillingAddress.Street)
	}
	if reloaded.BillingAddress.City != original.BillingAddress.City {
		t.Fatalf("BillingAddress.City = %q, want %q (should be untouched)", reloaded.BillingAddress.City, original.BillingAddress.City)
	}
}
