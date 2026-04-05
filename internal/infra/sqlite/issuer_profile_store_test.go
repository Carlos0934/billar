package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Carlos0934/billar/internal/app"
	"github.com/Carlos0934/billar/internal/core"
)

func TestIssuerProfileStoreSaveInsertsNewProfile(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	issuerStore := NewIssuerProfileStore(store)

	// First create a legal entity
	entity := core.LegalEntity{
		ID:        "le_issuer1",
		Type:      core.EntityTypeCompany,
		LegalName: "Issuer Company",
	}
	insertLegalEntity(t, store.DB(), entity)

	profile := core.IssuerProfile{
		ID:              "iss_test1",
		LegalEntityID:   "le_issuer1",
		DefaultCurrency: "USD",
		DefaultNotes:    "Default invoice notes",
	}

	if err := issuerStore.Save(context.Background(), &profile); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify via GetByID
	got, err := issuerStore.GetByID(context.Background(), "iss_test1")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got.ID != profile.ID {
		t.Errorf("GetByID().ID = %q, want %q", got.ID, profile.ID)
	}
	if got.LegalEntityID != profile.LegalEntityID {
		t.Errorf("GetByID().LegalEntityID = %q, want %q", got.LegalEntityID, profile.LegalEntityID)
	}
	if got.DefaultCurrency != profile.DefaultCurrency {
		t.Errorf("GetByID().DefaultCurrency = %q, want %q", got.DefaultCurrency, profile.DefaultCurrency)
	}
	if got.DefaultNotes != profile.DefaultNotes {
		t.Errorf("GetByID().DefaultNotes = %q, want %q", got.DefaultNotes, profile.DefaultNotes)
	}
}

func TestIssuerProfileStoreSaveUpdatesExistingProfile(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	issuerStore := NewIssuerProfileStore(store)

	// First create a legal entity
	entity := core.LegalEntity{
		ID:        "le_issuer2",
		Type:      core.EntityTypeCompany,
		LegalName: "Issuer Company",
	}
	insertLegalEntity(t, store.DB(), entity)

	profile := core.IssuerProfile{
		ID:              "iss_test2",
		LegalEntityID:   "le_issuer2",
		DefaultCurrency: "USD",
		DefaultNotes:    "Original notes",
	}

	if err := issuerStore.Save(context.Background(), &profile); err != nil {
		t.Fatalf("Save() first insert error = %v", err)
	}

	// Update the profile
	profile.DefaultCurrency = "EUR"
	profile.DefaultNotes = "Updated notes"

	if err := issuerStore.Save(context.Background(), &profile); err != nil {
		t.Fatalf("Save() update error = %v", err)
	}

	// Verify the update
	got, err := issuerStore.GetByID(context.Background(), "iss_test2")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got.DefaultCurrency != "EUR" {
		t.Errorf("DefaultCurrency = %q, want %q", got.DefaultCurrency, "EUR")
	}
	if got.DefaultNotes != "Updated notes" {
		t.Errorf("DefaultNotes = %q, want %q", got.DefaultNotes, "Updated notes")
	}
}

func TestIssuerProfileStoreGetByIDReturnsCorrectProfile(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	issuerStore := NewIssuerProfileStore(store)

	// First create a legal entity
	entity := core.LegalEntity{
		ID:        "le_issuer3",
		Type:      core.EntityTypeCompany,
		LegalName: "Test Issuer",
	}
	insertLegalEntity(t, store.DB(), entity)

	profile := core.IssuerProfile{
		ID:              "iss_test3",
		LegalEntityID:   "le_issuer3",
		DefaultCurrency: "USD",
		DefaultNotes:    "Test notes",
		CreatedAt:       time.Date(2026, 4, 4, 12, 0, 0, 0, time.UTC),
		UpdatedAt:       time.Date(2026, 4, 4, 12, 0, 0, 0, time.UTC),
	}
	insertIssuerProfile(t, store.DB(), profile)

	got, err := issuerStore.GetByID(context.Background(), "iss_test3")
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
	if got.DefaultCurrency != profile.DefaultCurrency {
		t.Errorf("DefaultCurrency = %q, want %q", got.DefaultCurrency, profile.DefaultCurrency)
	}
	if got.DefaultNotes != profile.DefaultNotes {
		t.Errorf("DefaultNotes = %q, want %q", got.DefaultNotes, profile.DefaultNotes)
	}
}

func TestIssuerProfileStoreGetByIDReturnsNotFoundForUnknownID(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	issuerStore := NewIssuerProfileStore(store)

	_, err := issuerStore.GetByID(context.Background(), "iss_nonexistent")
	if err == nil {
		t.Fatal("GetByID() expected error, got nil")
	}
	if !errors.Is(err, app.ErrIssuerProfileNotFound) {
		t.Errorf("GetByID() error = %v, want %v", err, app.ErrIssuerProfileNotFound)
	}
}

func TestIssuerProfileStoreFKViolationOnMissingLegalEntity(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	issuerStore := NewIssuerProfileStore(store)

	// Try to save a profile with non-existent legal_entity_id
	profile := core.IssuerProfile{
		ID:              "iss_orphan",
		LegalEntityID:   "le_nonexistent",
		DefaultCurrency: "USD",
	}

	err := issuerStore.Save(context.Background(), &profile)
	if err == nil {
		t.Fatal("Save() expected FK constraint error for non-existent legal_entity_id, got nil")
	}
}

func TestIssuerProfileStoreUpdatePreservesLegalEntityID(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	issuerStore := NewIssuerProfileStore(store)

	// Create a legal entity
	entity := core.LegalEntity{
		ID:        "le_issuer_preserve",
		Type:      core.EntityTypeCompany,
		LegalName: "Issuer Preserve Test",
	}
	insertLegalEntity(t, store.DB(), entity)

	profile := core.IssuerProfile{
		ID:              "iss_preserve",
		LegalEntityID:   "le_issuer_preserve",
		DefaultCurrency: "USD",
		DefaultNotes:    "Original",
	}
	insertIssuerProfile(t, store.DB(), profile)

	// Update only default notes
	profile.DefaultNotes = "Updated"
	if err := issuerStore.Save(context.Background(), &profile); err != nil {
		t.Fatalf("Save() update error = %v", err)
	}

	// Verify LegalEntityID is preserved
	got, err := issuerStore.GetByID(context.Background(), "iss_preserve")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got.LegalEntityID != "le_issuer_preserve" {
		t.Errorf("LegalEntityID = %q, want %q", got.LegalEntityID, "le_issuer_preserve")
	}
	if got.DefaultNotes != "Updated" {
		t.Errorf("DefaultNotes = %q, want %q", got.DefaultNotes, "Updated")
	}
}

func TestIssuerProfileStoreRejectsDuplicateLegalEntityID(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	issuerStore := NewIssuerProfileStore(store)

	// Create a legal entity
	entity := core.LegalEntity{
		ID:        "le_issuer_unique",
		Type:      core.EntityTypeCompany,
		LegalName: "Issuer Unique Test",
	}
	insertLegalEntity(t, store.DB(), entity)

	// Create first issuer profile for this legal entity
	profile1 := core.IssuerProfile{
		ID:              "iss_unique1",
		LegalEntityID:   "le_issuer_unique",
		DefaultCurrency: "USD",
	}
	if err := issuerStore.Save(context.Background(), &profile1); err != nil {
		t.Fatalf("Save() first profile error = %v", err)
	}

	// CRITICAL: Attempt to create a second issuer profile for the same legal entity
	// This MUST fail due to UNIQUE constraint on legal_entity_id
	profile2 := core.IssuerProfile{
		ID:              "iss_unique2",
		LegalEntityID:   "le_issuer_unique", // Same legal_entity_id as profile1
		DefaultCurrency: "EUR",
	}

	err := issuerStore.Save(context.Background(), &profile2)
	if err == nil {
		t.Fatal("Save() second profile with same legal_entity_id should have failed UNIQUE constraint, got nil")
	}
	// The error should be a SQLite constraint violation
	// SQLite returns "UNIQUE constraint failed" for UNIQUE violations
	if !strings.Contains(err.Error(), "UNIQUE constraint failed") && !strings.Contains(err.Error(), "unique") {
		t.Errorf("Save() error = %v, want UNIQUE constraint error", err)
	}

	// Verify only one profile exists for this legal entity
	got, err := issuerStore.GetByID(context.Background(), "iss_unique1")
	if err != nil {
		t.Fatalf("GetByID() first profile should still exist, got error = %v", err)
	}
	if got.LegalEntityID != "le_issuer_unique" {
		t.Errorf("First profile LegalEntityID = %q, want %q", got.LegalEntityID, "le_issuer_unique")
	}
}

// Helper for tests
func insertIssuerProfile(t *testing.T, db *sql.DB, profile core.IssuerProfile) {
	t.Helper()

	_, err := db.ExecContext(context.Background(), `
INSERT INTO issuer_profiles (id, legal_entity_id, default_currency, default_notes, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?)`,
		profile.ID,
		profile.LegalEntityID,
		profile.DefaultCurrency,
		profile.DefaultNotes,
		profile.CreatedAt.UTC().UnixNano(),
		profile.UpdatedAt.UTC().UnixNano(),
	)
	if err != nil {
		t.Fatalf("insertIssuerProfile: %v", err)
	}
}
