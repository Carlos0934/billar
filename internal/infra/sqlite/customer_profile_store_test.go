package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Carlos0934/billar/internal/app"
	"github.com/Carlos0934/billar/internal/core"
)

func TestCustomerProfileStoreListReturnsEmptyResult(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	profileStore := NewCustomerProfileStore(store)

	got, err := profileStore.List(context.Background(), app.ListQuery{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	want := app.ListResult[core.CustomerProfile]{Items: []core.CustomerProfile{}, Total: 0, Page: 1, PageSize: 20}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("List() = %+v, want %+v", got, want)
	}
}

func TestCustomerProfileStoreListSearchesPaginatesAndSorts(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	profileStore := NewCustomerProfileStore(store)

	// Create legal entities
	insertLegalEntity(t, store.DB(), core.LegalEntity{ID: "le_c1", Type: core.EntityTypeCompany, LegalName: "Alpha Company"})
	insertLegalEntity(t, store.DB(), core.LegalEntity{ID: "le_c2", Type: core.EntityTypeCompany, LegalName: "Beta Company"})
	insertLegalEntity(t, store.DB(), core.LegalEntity{ID: "le_c3", Type: core.EntityTypeIndividual, LegalName: "Gamma Person"})

	// Create customer profiles
	insertCustomerProfile(t, store.DB(), core.CustomerProfile{
		ID:              "cus_old",
		LegalEntityID:   "le_c1",
		Status:          core.CustomerProfileStatusActive,
		DefaultCurrency: "USD",
		CreatedAt:       time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC),
		UpdatedAt:       time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC),
	})
	insertCustomerProfile(t, store.DB(), core.CustomerProfile{
		ID:              "cus_new",
		LegalEntityID:   "le_c2",
		Status:          core.CustomerProfileStatusActive,
		DefaultCurrency: "EUR",
		CreatedAt:       time.Date(2026, 4, 3, 10, 5, 0, 0, time.UTC),
		UpdatedAt:       time.Date(2026, 4, 3, 10, 5, 0, 0, time.UTC),
	})
	insertCustomerProfile(t, store.DB(), core.CustomerProfile{
		ID:              "cus_other",
		LegalEntityID:   "le_c3",
		Status:          core.CustomerProfileStatusInactive,
		DefaultCurrency: "USD",
		CreatedAt:       time.Date(2026, 4, 3, 10, 10, 0, 0, time.UTC),
		UpdatedAt:       time.Date(2026, 4, 3, 10, 10, 0, 0, time.UTC),
	})

	// Test list without search (all profiles)
	all, err := profileStore.List(context.Background(), app.ListQuery{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("List() all error = %v", err)
	}
	if all.Total != 3 {
		t.Fatalf("List() total = %d, want 3", all.Total)
	}

	// Test sorting by created_at desc
	first, err := profileStore.List(context.Background(), app.ListQuery{Page: 1, PageSize: 1, SortField: "created_at", SortDir: "desc"})
	if err != nil {
		t.Fatalf("List() first page error = %v", err)
	}
	wantFirst := app.ListResult[core.CustomerProfile]{
		Items: []core.CustomerProfile{{
			ID:              "cus_other",
			LegalEntityID:   "le_c3",
			Status:          core.CustomerProfileStatusInactive,
			DefaultCurrency: "USD",
			CreatedAt:       time.Date(2026, 4, 3, 10, 10, 0, 0, time.UTC),
			UpdatedAt:       time.Date(2026, 4, 3, 10, 10, 0, 0, time.UTC),
		}},
		Total:    3,
		Page:     1,
		PageSize: 1,
	}
	if !reflect.DeepEqual(first, wantFirst) {
		t.Fatalf("List() first = %+v, want %+v", first, wantFirst)
	}
}

func TestCustomerProfileStoreSaveInsertsNewProfile(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	profileStore := NewCustomerProfileStore(store)

	// Create legal entity
	entity := core.LegalEntity{
		ID:        "le_customer1",
		Type:      core.EntityTypeCompany,
		LegalName: "Customer Company",
	}
	insertLegalEntity(t, store.DB(), entity)

	profile := core.CustomerProfile{
		ID:              "cus_test1",
		LegalEntityID:   "le_customer1",
		Status:          core.CustomerProfileStatusActive,
		DefaultCurrency: "USD",
		Notes:           "Important customer",
	}

	if err := profileStore.Save(context.Background(), &profile); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify via GetByID
	got, err := profileStore.GetByID(context.Background(), "cus_test1")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got.ID != profile.ID {
		t.Errorf("ID = %q, want %q", got.ID, profile.ID)
	}
	if got.LegalEntityID != profile.LegalEntityID {
		t.Errorf("LegalEntityID = %q, want %q", got.LegalEntityID, profile.LegalEntityID)
	}
	if got.Status != profile.Status {
		t.Errorf("Status = %q, want %q", got.Status, profile.Status)
	}
	if got.DefaultCurrency != profile.DefaultCurrency {
		t.Errorf("DefaultCurrency = %q, want %q", got.DefaultCurrency, profile.DefaultCurrency)
	}
	if got.Notes != profile.Notes {
		t.Errorf("Notes = %q, want %q", got.Notes, profile.Notes)
	}
}

func TestCustomerProfileStoreSaveUpdatesExistingProfile(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	profileStore := NewCustomerProfileStore(store)

	// Create legal entity
	entity := core.LegalEntity{
		ID:        "le_customer2",
		Type:      core.EntityTypeCompany,
		LegalName: "Customer Company",
	}
	insertLegalEntity(t, store.DB(), entity)

	profile := core.CustomerProfile{
		ID:              "cus_test2",
		LegalEntityID:   "le_customer2",
		Status:          core.CustomerProfileStatusActive,
		DefaultCurrency: "USD",
		Notes:           "Original notes",
	}

	if err := profileStore.Save(context.Background(), &profile); err != nil {
		t.Fatalf("Save() first insert error = %v", err)
	}

	// Update the profile
	profile.Status = core.CustomerProfileStatusInactive
	profile.DefaultCurrency = "EUR"
	profile.Notes = "Updated notes"

	if err := profileStore.Save(context.Background(), &profile); err != nil {
		t.Fatalf("Save() update error = %v", err)
	}

	// Verify the update
	got, err := profileStore.GetByID(context.Background(), "cus_test2")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got.Status != core.CustomerProfileStatusInactive {
		t.Errorf("Status = %q, want %q", got.Status, core.CustomerProfileStatusInactive)
	}
	if got.DefaultCurrency != "EUR" {
		t.Errorf("DefaultCurrency = %q, want %q", got.DefaultCurrency, "EUR")
	}
	if got.Notes != "Updated notes" {
		t.Errorf("Notes = %q, want %q", got.Notes, "Updated notes")
	}
}

func TestCustomerProfileStoreGetByIDReturnsCorrectProfile(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	profileStore := NewCustomerProfileStore(store)

	// Create legal entity
	entity := core.LegalEntity{
		ID:        "le_customer3",
		Type:      core.EntityTypeCompany,
		LegalName: "Test Customer",
	}
	insertLegalEntity(t, store.DB(), entity)

	profile := core.CustomerProfile{
		ID:              "cus_test3",
		LegalEntityID:   "le_customer3",
		Status:          core.CustomerProfileStatusActive,
		DefaultCurrency: "USD",
		Notes:           "Test notes",
		CreatedAt:       time.Date(2026, 4, 4, 12, 0, 0, 0, time.UTC),
		UpdatedAt:       time.Date(2026, 4, 4, 12, 0, 0, 0, time.UTC),
	}
	insertCustomerProfile(t, store.DB(), profile)

	got, err := profileStore.GetByID(context.Background(), "cus_test3")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got == nil {
		t.Fatal("GetByID() returned nil")
	}
	if got.ID != profile.ID {
		t.Errorf("ID = %q, want %q", got.ID, profile.ID)
	}
	if got.LegalEntityID != profile.LegalEntityID {
		t.Errorf("LegalEntityID = %q, want %q", got.LegalEntityID, profile.LegalEntityID)
	}
	if got.Status != profile.Status {
		t.Errorf("Status = %q, want %q", got.Status, profile.Status)
	}
}

func TestCustomerProfileStoreGetByIDReturnsNotFoundForUnknownID(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	profileStore := NewCustomerProfileStore(store)

	_, err := profileStore.GetByID(context.Background(), "cus_nonexistent")
	if err == nil {
		t.Fatal("GetByID() expected error, got nil")
	}
	if !errors.Is(err, app.ErrCustomerProfileNotFound) {
		t.Errorf("GetByID() error = %v, want %v", err, app.ErrCustomerProfileNotFound)
	}
}

func TestCustomerProfileStoreDeleteRemovesProfile(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	profileStore := NewCustomerProfileStore(store)

	// Create legal entity
	entity := core.LegalEntity{
		ID:        "le_customer_delete",
		Type:      core.EntityTypeCompany,
		LegalName: "To Delete",
	}
	insertLegalEntity(t, store.DB(), entity)

	profile := core.CustomerProfile{
		ID:              "cus_todelete",
		LegalEntityID:   "le_customer_delete",
		Status:          core.CustomerProfileStatusActive,
		DefaultCurrency: "USD",
	}
	insertCustomerProfile(t, store.DB(), profile)

	if err := profileStore.Delete(context.Background(), "cus_todelete"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify via GetByID returns not found
	_, err := profileStore.GetByID(context.Background(), "cus_todelete")
	if err == nil {
		t.Fatal("GetByID() expected error after delete, got nil")
	}
	if !errors.Is(err, app.ErrCustomerProfileNotFound) {
		t.Errorf("GetByID() error = %v, want %v", err, app.ErrCustomerProfileNotFound)
	}

	// Verify via List returns empty
	result, err := profileStore.List(context.Background(), app.ListQuery{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if result.Total != 0 {
		t.Errorf("List() total after delete = %d, want 0", result.Total)
	}
}

func TestCustomerProfileStoreDeleteReturnsNotFoundForUnknownID(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	profileStore := NewCustomerProfileStore(store)

	err := profileStore.Delete(context.Background(), "cus_nonexistent")
	if err == nil {
		t.Fatal("Delete() expected error, got nil")
	}
	if !errors.Is(err, app.ErrCustomerProfileNotFound) {
		t.Errorf("Delete() error = %v, want %v", err, app.ErrCustomerProfileNotFound)
	}
}

func TestCustomerProfileStoreFKViolationOnMissingLegalEntity(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	profileStore := NewCustomerProfileStore(store)

	// Try to save a profile with non-existent legal_entity_id
	profile := core.CustomerProfile{
		ID:              "cus_orphan",
		LegalEntityID:   "le_nonexistent",
		Status:          core.CustomerProfileStatusActive,
		DefaultCurrency: "USD",
	}

	err := profileStore.Save(context.Background(), &profile)
	if err == nil {
		t.Fatal("Save() expected FK constraint error for non-existent legal_entity_id, got nil")
	}
}

func TestCustomerProfileStoreUpdatePreservesLegalEntityID(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	profileStore := NewCustomerProfileStore(store)

	// Create legal entity
	entity := core.LegalEntity{
		ID:        "le_preserve",
		Type:      core.EntityTypeCompany,
		LegalName: "Preserve Test",
	}
	insertLegalEntity(t, store.DB(), entity)

	profile := core.CustomerProfile{
		ID:              "cus_preserve",
		LegalEntityID:   "le_preserve",
		Status:          core.CustomerProfileStatusActive,
		DefaultCurrency: "USD",
		Notes:           "Original",
	}
	insertCustomerProfile(t, store.DB(), profile)

	// Update only notes
	profile.Notes = "Updated"
	if err := profileStore.Save(context.Background(), &profile); err != nil {
		t.Fatalf("Save() update error = %v", err)
	}

	// Verify LegalEntityID is preserved
	got, err := profileStore.GetByID(context.Background(), "cus_preserve")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got.LegalEntityID != "le_preserve" {
		t.Errorf("LegalEntityID = %q, want %q", got.LegalEntityID, "le_preserve")
	}
	if got.Notes != "Updated" {
		t.Errorf("Notes = %q, want %q", got.Notes, "Updated")
	}
}

func TestCustomerProfileStoreRejectsDuplicateLegalEntityID(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	profileStore := NewCustomerProfileStore(store)

	// Create legal entity
	entity := core.LegalEntity{
		ID:        "le_customer_unique",
		Type:      core.EntityTypeCompany,
		LegalName: "Customer Unique Test",
	}
	insertLegalEntity(t, store.DB(), entity)

	// Create first customer profile for this legal entity
	profile1 := core.CustomerProfile{
		ID:              "cus_unique1",
		LegalEntityID:   "le_customer_unique",
		Status:          core.CustomerProfileStatusActive,
		DefaultCurrency: "USD",
	}
	if err := profileStore.Save(context.Background(), &profile1); err != nil {
		t.Fatalf("Save() first profile error = %v", err)
	}

	// CRITICAL: Attempt to create a second customer profile for the same legal entity
	// This MUST fail due to UNIQUE constraint on legal_entity_id
	profile2 := core.CustomerProfile{
		ID:              "cus_unique2",
		LegalEntityID:   "le_customer_unique", // Same legal_entity_id as profile1
		Status:          core.CustomerProfileStatusActive,
		DefaultCurrency: "EUR",
	}

	err := profileStore.Save(context.Background(), &profile2)
	if err == nil {
		t.Fatal("Save() second profile with same legal_entity_id should have failed UNIQUE constraint, got nil")
	}
	// The error should be a SQLite constraint violation
	// SQLite returns "UNIQUE constraint failed" for UNIQUE violations
	if !strings.Contains(err.Error(), "UNIQUE constraint failed") && !strings.Contains(err.Error(), "unique") {
		t.Errorf("Save() error = %v, want UNIQUE constraint error", err)
	}

	// Verify only one profile exists for this legal entity
	got, err := profileStore.GetByID(context.Background(), "cus_unique1")
	if err != nil {
		t.Fatalf("GetByID() first profile should still exist, got error = %v", err)
	}
	if got.LegalEntityID != "le_customer_unique" {
		t.Errorf("First profile LegalEntityID = %q, want %q", got.LegalEntityID, "le_customer_unique")
	}
}

// Helper for tests
func insertCustomerProfile(t *testing.T, db *sql.DB, profile core.CustomerProfile) {
	t.Helper()

	_, err := db.ExecContext(context.Background(), `
INSERT INTO customer_profiles (id, legal_entity_id, status, default_currency, notes, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?)`,
		profile.ID,
		profile.LegalEntityID,
		string(profile.Status),
		profile.DefaultCurrency,
		profile.Notes,
		profile.CreatedAt.UTC().UnixNano(),
		profile.UpdatedAt.UTC().UnixNano(),
	)
	if err != nil {
		t.Fatalf("insertCustomerProfile: %v", err)
	}
}
