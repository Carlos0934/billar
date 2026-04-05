package sqlite

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/Carlos0934/billar/internal/app"
	"github.com/Carlos0934/billar/internal/core"
)

func TestLegalEntityStoreListReturnsEmptyResult(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	legalStore := NewLegalEntityStore(store)

	got, err := legalStore.List(context.Background(), app.ListQuery{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	want := app.ListResult[core.LegalEntity]{Items: []core.LegalEntity{}, Total: 0, Page: 1, PageSize: 20}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("List() = %+v, want %+v", got, want)
	}
}

func TestLegalEntityStoreListSearchesPaginatesAndSorts(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	legalStore := NewLegalEntityStore(store)

	// Insert test data
	insertLegalEntity(t, store.DB(), core.LegalEntity{
		ID:        "le_old",
		Type:      core.EntityTypeCompany,
		LegalName: "Acme Alpha",
		CreatedAt: time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC),
	})
	insertLegalEntity(t, store.DB(), core.LegalEntity{
		ID:        "le_new",
		Type:      core.EntityTypeCompany,
		LegalName: "Acme Zeta",
		CreatedAt: time.Date(2026, 4, 3, 10, 5, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 4, 3, 10, 5, 0, 0, time.UTC),
	})
	insertLegalEntity(t, store.DB(), core.LegalEntity{
		ID:        "le_other",
		Type:      core.EntityTypeIndividual,
		LegalName: "Beta Other",
		CreatedAt: time.Date(2026, 4, 3, 10, 10, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 4, 3, 10, 10, 0, 0, time.UTC),
	})

	// Test first page with search, sorted by created_at desc
	first, err := legalStore.List(context.Background(), app.ListQuery{Search: "  acme  ", Page: 1, PageSize: 1, SortField: "created_at", SortDir: "desc"})
	if err != nil {
		t.Fatalf("List() first page error = %v", err)
	}
	wantFirst := app.ListResult[core.LegalEntity]{
		Items: []core.LegalEntity{{
			ID:        "le_new",
			Type:      core.EntityTypeCompany,
			LegalName: "Acme Zeta",
			CreatedAt: time.Date(2026, 4, 3, 10, 5, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, 4, 3, 10, 5, 0, 0, time.UTC),
		}},
		Total:    2,
		Page:     1,
		PageSize: 1,
	}
	if !reflect.DeepEqual(first, wantFirst) {
		t.Fatalf("List() first page = %+v, want %+v", first, wantFirst)
	}

	// Test second page
	second, err := legalStore.List(context.Background(), app.ListQuery{Search: "acme", Page: 2, PageSize: 1, SortField: "created_at", SortDir: "desc"})
	if err != nil {
		t.Fatalf("List() second page error = %v", err)
	}
	wantSecond := app.ListResult[core.LegalEntity]{
		Items: []core.LegalEntity{{
			ID:        "le_old",
			Type:      core.EntityTypeCompany,
			LegalName: "Acme Alpha",
			CreatedAt: time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC),
		}},
		Total:    2,
		Page:     2,
		PageSize: 1,
	}
	if !reflect.DeepEqual(second, wantSecond) {
		t.Fatalf("List() second page = %+v, want %+v", second, wantSecond)
	}

	// Test sort by legal_name ascending
	byName, err := legalStore.List(context.Background(), app.ListQuery{Search: "acme", Page: 1, PageSize: 2, SortField: "name", SortDir: "asc"})
	if err != nil {
		t.Fatalf("List() by name error = %v", err)
	}
	wantByName := app.ListResult[core.LegalEntity]{
		Items: []core.LegalEntity{{
			ID:        "le_old",
			Type:      core.EntityTypeCompany,
			LegalName: "Acme Alpha",
			CreatedAt: time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC),
		}, {
			ID:        "le_new",
			Type:      core.EntityTypeCompany,
			LegalName: "Acme Zeta",
			CreatedAt: time.Date(2026, 4, 3, 10, 5, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, 4, 3, 10, 5, 0, 0, time.UTC),
		}},
		Total:    2,
		Page:     1,
		PageSize: 2,
	}
	if !reflect.DeepEqual(byName, wantByName) {
		t.Fatalf("List() by name = %+v, want %+v", byName, wantByName)
	}
}

func TestLegalEntityStoreSaveInsertsNewEntity(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	legalStore := NewLegalEntityStore(store)

	entity := core.LegalEntity{
		Type:           core.EntityTypeCompany,
		LegalName:      "Acme Corporation",
		TradeName:      "Acme",
		Email:          "contact@acme.com",
		BillingAddress: core.Address{Street: "123 Main St", City: "City"},
	}

	if err := legalStore.Save(context.Background(), &entity); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify via List
	result, err := legalStore.List(context.Background(), app.ListQuery{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("List() total = %d, want 1", result.Total)
	}
	if len(result.Items) != 1 {
		t.Fatalf("List() items = %d, want 1", len(result.Items))
	}
	if result.Items[0].LegalName != entity.LegalName {
		t.Fatalf("List()[0].LegalName = %q, want %q", result.Items[0].LegalName, entity.LegalName)
	}
	if result.Items[0].ID != entity.ID {
		t.Fatalf("List()[0].ID = %q, want %q", result.Items[0].ID, entity.ID)
	}
}

func TestLegalEntityStoreSaveUpdatesExistingEntity(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	legalStore := NewLegalEntityStore(store)

	entity := core.LegalEntity{
		Type:      core.EntityTypeCompany,
		LegalName: "Acme Corporation",
		Email:     "old@acme.com",
	}

	if err := legalStore.Save(context.Background(), &entity); err != nil {
		t.Fatalf("Save() first insert error = %v", err)
	}

	originalID := entity.ID
	entity.Email = "new@acme.com"
	entity.TradeName = "Acme Updated"

	if err := legalStore.Save(context.Background(), &entity); err != nil {
		t.Fatalf("Save() update error = %v", err)
	}

	// Verify the entity was updated, not duplicated
	result, err := legalStore.List(context.Background(), app.ListQuery{Page: 1, PageSize: 10})
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

func TestLegalEntityStoreGetByIDReturnsCorrectEntity(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	legalStore := NewLegalEntityStore(store)

	inserted := core.LegalEntity{
		ID:             "le_test123",
		Type:           core.EntityTypeCompany,
		LegalName:      "Test Company",
		Email:          "test@company.com",
		BillingAddress: core.Address{Street: "456 Oak Ave", City: "Test City"},
	}
	insertLegalEntity(t, store.DB(), inserted)

	got, err := legalStore.GetByID(context.Background(), "le_test123")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got == nil {
		t.Fatalf("GetByID() returned nil entity")
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
	if got.BillingAddress.Street != inserted.BillingAddress.Street {
		t.Errorf("GetByID().BillingAddress.Street = %q, want %q", got.BillingAddress.Street, inserted.BillingAddress.Street)
	}
	if got.BillingAddress.City != inserted.BillingAddress.City {
		t.Errorf("GetByID().BillingAddress.City = %q, want %q", got.BillingAddress.City, inserted.BillingAddress.City)
	}
}

func TestLegalEntityStoreGetByIDReturnsNotFoundForUnknownID(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	legalStore := NewLegalEntityStore(store)

	_, err := legalStore.GetByID(context.Background(), "le_nonexistent")
	if err == nil {
		t.Fatalf("GetByID() expected error, got nil")
	}
	if !errors.Is(err, app.ErrLegalEntityNotFound) {
		t.Fatalf("GetByID() error = %v, want %v", err, app.ErrLegalEntityNotFound)
	}
}

func TestLegalEntityStoreDeleteRemovesEntity(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	legalStore := NewLegalEntityStore(store)

	inserted := core.LegalEntity{
		ID:        "le_todelete",
		Type:      core.EntityTypeIndividual,
		LegalName: "To Delete",
	}
	insertLegalEntity(t, store.DB(), inserted)

	if err := legalStore.Delete(context.Background(), "le_todelete"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify via GetByID returns not found
	_, err := legalStore.GetByID(context.Background(), "le_todelete")
	if err == nil {
		t.Fatalf("GetByID() expected error after delete, got nil")
	}
	if !errors.Is(err, app.ErrLegalEntityNotFound) {
		t.Fatalf("GetByID() error = %v, want %v", err, app.ErrLegalEntityNotFound)
	}

	// Verify via List returns empty
	result, err := legalStore.List(context.Background(), app.ListQuery{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if result.Total != 0 {
		t.Fatalf("List() total after delete = %d, want 0", result.Total)
	}
}

func TestLegalEntityStoreDeleteReturnsNotFoundForUnknownID(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	legalStore := NewLegalEntityStore(store)

	err := legalStore.Delete(context.Background(), "le_nonexistent")
	if err == nil {
		t.Fatalf("Delete() expected error, got nil")
	}
	if !errors.Is(err, app.ErrLegalEntityNotFound) {
		t.Fatalf("Delete() error = %v, want %v", err, app.ErrLegalEntityNotFound)
	}
}

func TestLegalEntityStoreUpdatePreservesLinkedProfiles(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	legalStore := NewLegalEntityStore(store)
	issuerStore := NewIssuerProfileStore(store)
	customerStore := NewCustomerProfileStore(store)

	// Create a legal entity
	entity := core.LegalEntity{
		ID:        "le_preserve_test",
		Type:      core.EntityTypeCompany,
		LegalName: "Original Name",
		Email:     "original@example.com",
	}
	insertLegalEntity(t, store.DB(), entity)

	// Create linked issuer profile
	issuer := core.IssuerProfile{
		ID:              "iss_preserve_test",
		LegalEntityID:   "le_preserve_test",
		DefaultCurrency: "USD",
		DefaultNotes:    "Issuer notes",
	}
	insertIssuerProfile(t, store.DB(), issuer)

	// Create linked customer profile
	customer := core.CustomerProfile{
		ID:              "cus_preserve_test",
		LegalEntityID:   "le_preserve_test",
		Status:          core.CustomerProfileStatusActive,
		DefaultCurrency: "EUR",
	}
	insertCustomerProfile(t, store.DB(), customer)

	// Update the legal entity
	entity.Email = "updated@example.com"
	entity.LegalName = "Updated Name"

	if err := legalStore.Save(context.Background(), &entity); err != nil {
		t.Fatalf("Save() update error = %v", err)
	}

	// Verify the entity was updated
	gotEntity, err := legalStore.GetByID(context.Background(), "le_preserve_test")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if gotEntity.Email != "updated@example.com" {
		t.Errorf("Email = %q, want %q", gotEntity.Email, "updated@example.com")
	}
	if gotEntity.LegalName != "Updated Name" {
		t.Errorf("LegalName = %q, want %q", gotEntity.LegalName, "Updated Name")
	}

	// CRITICAL: Verify issuer profile is NOT deleted
	gotIssuer, err := issuerStore.GetByID(context.Background(), "iss_preserve_test")
	if err != nil {
		t.Fatalf("GetByID() issuer profile should still exist after entity update, got error = %v", err)
	}
	if gotIssuer.LegalEntityID != "le_preserve_test" {
		t.Errorf("Issuer LegalEntityID = %q, want %q", gotIssuer.LegalEntityID, "le_preserve_test")
	}

	// CRITICAL: Verify customer profile is NOT deleted
	gotCustomer, err := customerStore.GetByID(context.Background(), "cus_preserve_test")
	if err != nil {
		t.Fatalf("GetByID() customer profile should still exist after entity update, got error = %v", err)
	}
	if gotCustomer.LegalEntityID != "le_preserve_test" {
		t.Errorf("Customer LegalEntityID = %q, want %q", gotCustomer.LegalEntityID, "le_preserve_test")
	}
}

func TestLegalEntityStorePatchLeavesOtherFieldsUntouched(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	legalStore := NewLegalEntityStore(store)

	original := core.LegalEntity{
		ID:             "le_test_untouched",
		Type:           core.EntityTypeCompany,
		LegalName:      "Original Legal Name",
		TradeName:      "Original Trade Name",
		Email:          "original@example.com",
		Phone:          "+1 555 0100",
		Website:        "https://original.example",
		BillingAddress: core.Address{Street: "123 Main St", City: "Original City"},
		CreatedAt:      time.Date(2026, 4, 4, 10, 0, 0, 0, time.UTC),
		UpdatedAt:      time.Date(2026, 4, 4, 10, 0, 0, 0, time.UTC),
	}
	insertLegalEntity(t, store.DB(), original)

	// Simulate a PATCH that updates only the email field
	email := "updated@example.com"
	patch := core.LegalEntityPatch{
		Email: &email,
	}

	// Retrieve, apply patch, and save
	entity, err := legalStore.GetByID(context.Background(), original.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	entity.ApplyPatch(patch)

	// Verify that ApplyPatch does NOT modify fields with nil pointers
	if entity.TradeName != original.TradeName {
		t.Fatalf("ApplyPatch modified TradeName (should be untouched), got = %q, want %q", entity.TradeName, original.TradeName)
	}
	if entity.Phone != original.Phone {
		t.Fatalf("ApplyPatch modified Phone (should be untouched), got = %q, want %q", entity.Phone, original.Phone)
	}
	if entity.Website != original.Website {
		t.Fatalf("ApplyPatch modified Website (should be untouched), got = %q, want %q", entity.Website, original.Website)
	}

	// Validate after patch
	if err := entity.Validate(); err != nil {
		t.Fatalf("Validate() after patch error = %v", err)
	}

	if err := legalStore.Save(context.Background(), entity); err != nil {
		t.Fatalf("Save() after patch error = %v", err)
	}

	// Retrieve again and verify ALL untouched fields remain unchanged
	reloaded, err := legalStore.GetByID(context.Background(), original.ID)
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
	if reloaded.BillingAddress.Street != original.BillingAddress.Street {
		t.Fatalf("BillingAddress.Street = %q, want %q (should be untouched)", reloaded.BillingAddress.Street, original.BillingAddress.Street)
	}
	if reloaded.BillingAddress.City != original.BillingAddress.City {
		t.Fatalf("BillingAddress.City = %q, want %q (should be untouched)", reloaded.BillingAddress.City, original.BillingAddress.City)
	}
}
