package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/Carlos0934/billar/internal/app"
	"github.com/Carlos0934/billar/internal/core"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// newAgreementFixture inserts a customer profile (with legal entity) ready for
// service agreement tests, returning the customer profile ID.
func newAgreementFixture(t *testing.T, db *sql.DB, leID, cusID string) {
	t.Helper()
	insertLegalEntity(t, db, core.LegalEntity{
		ID:        leID,
		Type:      core.EntityTypeCompany,
		LegalName: "Agreement Test Co",
	})
	insertCustomerProfile(t, db, core.CustomerProfile{
		ID:              cusID,
		LegalEntityID:   leID,
		Status:          core.CustomerProfileStatusActive,
		DefaultCurrency: "USD",
	})
}

func insertServiceAgreement(t *testing.T, db *sql.DB, sa core.ServiceAgreement) {
	t.Helper()

	var validFrom, validUntil interface{}
	if sa.ValidFrom != nil {
		validFrom = sa.ValidFrom.UTC().UnixNano()
	}
	if sa.ValidUntil != nil {
		validUntil = sa.ValidUntil.UTC().UnixNano()
	}

	active := 0
	if sa.Active {
		active = 1
	}

	_, err := db.ExecContext(context.Background(), `
INSERT INTO service_agreements
  (id, customer_profile_id, name, description, billing_mode, hourly_rate, currency, active, valid_from, valid_until, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sa.ID, sa.CustomerProfileID, sa.Name, sa.Description,
		string(sa.BillingMode), sa.HourlyRate, sa.Currency, active,
		validFrom, validUntil,
		sa.CreatedAt.UTC().UnixNano(), sa.UpdatedAt.UTC().UnixNano(),
	)
	if err != nil {
		t.Fatalf("insertServiceAgreement: %v", err)
	}
}

// ---------------------------------------------------------------------------
// NewServiceAgreementStore
// ---------------------------------------------------------------------------

func TestNewServiceAgreementStoreRejectsNil(t *testing.T) {
	t.Parallel()

	s := NewServiceAgreementStore(nil)
	if s != nil {
		t.Fatal("NewServiceAgreementStore(nil) = non-nil, want nil")
	}
}

// ---------------------------------------------------------------------------
// Save — insert
// ---------------------------------------------------------------------------

func TestServiceAgreementStoreSaveInsertsNewAgreement(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	newAgreementFixture(t, store.DB(), "le_sa_ins", "cus_sa_ins")
	saStore := NewServiceAgreementStore(store)

	now := time.Date(2026, 4, 10, 10, 0, 0, 0, time.UTC)
	sa := core.ServiceAgreement{
		ID:                "sa_insert_test",
		CustomerProfileID: "cus_sa_ins",
		Name:              "Basic Support",
		Description:       "Monthly retainer",
		BillingMode:       core.BillingModeHourly,
		HourlyRate:        5000,
		Currency:          "USD",
		Active:            true,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if err := saStore.Save(context.Background(), &sa); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify round-trip via GetByID
	got, err := saStore.GetByID(context.Background(), "sa_insert_test")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got.ID != sa.ID {
		t.Errorf("ID = %q, want %q", got.ID, sa.ID)
	}
	if got.CustomerProfileID != sa.CustomerProfileID {
		t.Errorf("CustomerProfileID = %q, want %q", got.CustomerProfileID, sa.CustomerProfileID)
	}
	if got.Name != sa.Name {
		t.Errorf("Name = %q, want %q", got.Name, sa.Name)
	}
	if got.HourlyRate != sa.HourlyRate {
		t.Errorf("HourlyRate = %d, want %d", got.HourlyRate, sa.HourlyRate)
	}
	if got.Currency != sa.Currency {
		t.Errorf("Currency = %q, want %q", got.Currency, sa.Currency)
	}
	if !got.Active {
		t.Errorf("Active = false, want true")
	}
	if got.BillingMode != sa.BillingMode {
		t.Errorf("BillingMode = %q, want %q", got.BillingMode, sa.BillingMode)
	}
}

// ---------------------------------------------------------------------------
// Save — update
// ---------------------------------------------------------------------------

func TestServiceAgreementStoreSaveUpdatesExistingAgreement(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	newAgreementFixture(t, store.DB(), "le_sa_upd", "cus_sa_upd")
	saStore := NewServiceAgreementStore(store)

	now := time.Date(2026, 4, 10, 10, 0, 0, 0, time.UTC)
	sa := core.ServiceAgreement{
		ID:                "sa_update_test",
		CustomerProfileID: "cus_sa_upd",
		Name:              "Original Name",
		BillingMode:       core.BillingModeHourly,
		HourlyRate:        1000,
		Currency:          "USD",
		Active:            true,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if err := saStore.Save(context.Background(), &sa); err != nil {
		t.Fatalf("initial Save() error = %v", err)
	}

	// Mutate and save again (update rate + deactivate)
	sa.HourlyRate = 2000
	sa.Active = false
	sa.UpdatedAt = now.Add(time.Hour)

	if err := saStore.Save(context.Background(), &sa); err != nil {
		t.Fatalf("update Save() error = %v", err)
	}

	got, err := saStore.GetByID(context.Background(), "sa_update_test")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got.HourlyRate != 2000 {
		t.Errorf("HourlyRate = %d, want 2000", got.HourlyRate)
	}
	if got.Active {
		t.Errorf("Active = true after deactivation, want false")
	}
}

// ---------------------------------------------------------------------------
// GetByID — not found
// ---------------------------------------------------------------------------

func TestServiceAgreementStoreGetByIDReturnsNotFoundForUnknownID(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	saStore := NewServiceAgreementStore(store)

	_, err := saStore.GetByID(context.Background(), "sa_nonexistent")
	if err == nil {
		t.Fatal("GetByID() expected error, got nil")
	}
	if !errors.Is(err, app.ErrServiceAgreementNotFound) {
		t.Errorf("GetByID() error = %v, want %v", err, app.ErrServiceAgreementNotFound)
	}
}

// ---------------------------------------------------------------------------
// ListByCustomerProfileID
// ---------------------------------------------------------------------------

func TestServiceAgreementStoreListByCustomerProfileIDReturnsAgreements(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	newAgreementFixture(t, store.DB(), "le_sa_list", "cus_sa_list")
	saStore := NewServiceAgreementStore(store)

	now := time.Date(2026, 4, 10, 10, 0, 0, 0, time.UTC)
	insertServiceAgreement(t, store.DB(), core.ServiceAgreement{
		ID:                "sa_list_1",
		CustomerProfileID: "cus_sa_list",
		Name:              "Plan A",
		BillingMode:       core.BillingModeHourly,
		HourlyRate:        1000,
		Currency:          "USD",
		Active:            true,
		CreatedAt:         now,
		UpdatedAt:         now,
	})
	insertServiceAgreement(t, store.DB(), core.ServiceAgreement{
		ID:                "sa_list_2",
		CustomerProfileID: "cus_sa_list",
		Name:              "Plan B",
		BillingMode:       core.BillingModeHourly,
		HourlyRate:        2000,
		Currency:          "EUR",
		Active:            false,
		CreatedAt:         now.Add(time.Minute),
		UpdatedAt:         now.Add(time.Minute),
	})

	results, err := saStore.ListByCustomerProfileID(context.Background(), "cus_sa_list")
	if err != nil {
		t.Fatalf("ListByCustomerProfileID() error = %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}
	// Results ordered by created_at asc: sa_list_1 first
	if results[0].ID != "sa_list_1" {
		t.Errorf("results[0].ID = %q, want sa_list_1", results[0].ID)
	}
	if results[1].ID != "sa_list_2" {
		t.Errorf("results[1].ID = %q, want sa_list_2", results[1].ID)
	}
}

func TestServiceAgreementStoreListByCustomerProfileIDReturnsEmptyForUnknownProfile(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	saStore := NewServiceAgreementStore(store)

	results, err := saStore.ListByCustomerProfileID(context.Background(), "cus_unknown")
	if err != nil {
		t.Fatalf("ListByCustomerProfileID() error = %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

// ---------------------------------------------------------------------------
// Save — ValidFrom / ValidUntil round-trip
// ---------------------------------------------------------------------------

func TestServiceAgreementStoreSavePreservesOptionalDates(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	newAgreementFixture(t, store.DB(), "le_sa_dates", "cus_sa_dates")
	saStore := NewServiceAgreementStore(store)

	now := time.Date(2026, 4, 10, 10, 0, 0, 0, time.UTC)
	future := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	sa := core.ServiceAgreement{
		ID:                "sa_dates_test",
		CustomerProfileID: "cus_sa_dates",
		Name:              "Dated Agreement",
		BillingMode:       core.BillingModeHourly,
		HourlyRate:        3000,
		Currency:          "DOP",
		Active:            true,
		ValidFrom:         &now,
		ValidUntil:        &future,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if err := saStore.Save(context.Background(), &sa); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := saStore.GetByID(context.Background(), "sa_dates_test")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got.ValidFrom == nil {
		t.Fatal("ValidFrom = nil, want non-nil")
	}
	if !got.ValidFrom.Equal(now) {
		t.Errorf("ValidFrom = %v, want %v", got.ValidFrom, now)
	}
	if got.ValidUntil == nil {
		t.Fatal("ValidUntil = nil, want non-nil")
	}
	if !got.ValidUntil.Equal(future) {
		t.Errorf("ValidUntil = %v, want %v", got.ValidUntil, future)
	}
}

// ---------------------------------------------------------------------------
// Save — FK violation (no customer_profile_id)
// ---------------------------------------------------------------------------

func TestServiceAgreementStoreSaveFKViolationOnMissingCustomerProfile(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	saStore := NewServiceAgreementStore(store)

	now := time.Now().UTC()
	sa := core.ServiceAgreement{
		ID:                "sa_orphan",
		CustomerProfileID: "cus_nonexistent",
		Name:              "Orphan Agreement",
		BillingMode:       core.BillingModeHourly,
		HourlyRate:        1000,
		Currency:          "USD",
		Active:            true,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	err := saStore.Save(context.Background(), &sa)
	if err == nil {
		t.Fatal("Save() expected FK constraint error for non-existent customer_profile_id, got nil")
	}
}
