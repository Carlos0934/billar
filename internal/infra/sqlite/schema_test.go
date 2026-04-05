package sqlite

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/Carlos0934/billar/internal/core"
)

// TestSchemaHasLegalEntitiesTable verifies that the legal_entities table exists with correct columns
func TestSchemaHasLegalEntitiesTable(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	db := store.DB()

	// Verify the table exists and has the expected columns
	var count int
	err := db.QueryRowContext(context.Background(),
		"SELECT COUNT(*) FROM pragma_table_info('legal_entities')").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query legal_entities table info: %v", err)
	}
	if count < 10 {
		t.Fatalf("legal_entities table has %d columns, expected at least 10", count)
	}

	// Verify we can insert a legal entity
	_, err = db.ExecContext(context.Background(), `
		INSERT INTO legal_entities (id, type, legal_name, trade_name, tax_id, email, phone, website, billing_address, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"le_test123", "company", "Test Company", "Test", "", "test@example.com", "", "", "{}", time.Now().UTC().UnixNano(), time.Now().UTC().UnixNano())
	if err != nil {
		t.Fatalf("Failed to insert legal entity: %v", err)
	}
}

// TestSchemaHasIssuerProfilesTable verifies that the issuer_profiles table exists with FK to legal_entities
func TestSchemaHasIssuerProfilesTable(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	db := store.DB()

	// First create a legal entity
	_, err := db.ExecContext(context.Background(), `
		INSERT INTO legal_entities (id, type, legal_name, trade_name, tax_id, email, phone, website, billing_address, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"le_issuer123", "company", "Issuer Company", "Issuer", "", "issuer@example.com", "", "", "{}", time.Now().UTC().UnixNano(), time.Now().UTC().UnixNano())
	if err != nil {
		t.Fatalf("Failed to insert legal entity: %v", err)
	}

	// Verify we can insert an issuer profile with valid legal_entity_id
	_, err = db.ExecContext(context.Background(), `
		INSERT INTO issuer_profiles (id, legal_entity_id, default_currency, default_notes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		"iss_test123", "le_issuer123", "USD", "Test notes", time.Now().UTC().UnixNano(), time.Now().UTC().UnixNano())
	if err != nil {
		t.Fatalf("Failed to insert issuer profile: %v", err)
	}
}

// TestSchemaIssuerProfileFKConstraint tests that FK constraint prevents orphaned issuer profiles
func TestSchemaIssuerProfileFKConstraint(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	db := store.DB()

	// Try to insert an issuer profile with non-existent legal_entity_id
	_, err := db.ExecContext(context.Background(), `
		INSERT INTO issuer_profiles (id, legal_entity_id, default_currency, default_notes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		"iss_orphan", "le_nonexistent", "USD", "Notes", time.Now().UTC().UnixNano(), time.Now().UTC().UnixNano())
	if err == nil {
		t.Fatal("Expected FK constraint error for non-existent legal_entity_id, got nil")
	}
}

// TestSchemaHasCustomerProfilesTable verifies that the customer_profiles table exists with FK to legal_entities
func TestSchemaHasCustomerProfilesTable(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	db := store.DB()

	// First create a legal entity
	_, err := db.ExecContext(context.Background(), `
		INSERT INTO legal_entities (id, type, legal_name, trade_name, tax_id, email, phone, website, billing_address, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"le_customer123", "company", "Customer Company", "Customer", "", "customer@example.com", "", "", "{}", time.Now().UTC().UnixNano(), time.Now().UTC().UnixNano())
	if err != nil {
		t.Fatalf("Failed to insert legal entity: %v", err)
	}

	// Verify we can insert a customer profile with valid legal_entity_id
	_, err = db.ExecContext(context.Background(), `
		INSERT INTO customer_profiles (id, legal_entity_id, status, default_currency, notes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"cus_test123", "le_customer123", "active", "USD", "Test notes", time.Now().UTC().UnixNano(), time.Now().UTC().UnixNano())
	if err != nil {
		t.Fatalf("Failed to insert customer profile: %v", err)
	}
}

// TestSchemaCustomerProfileFKConstraint tests that FK constraint prevents orphaned customer profiles
func TestSchemaCustomerProfileFKConstraint(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	db := store.DB()

	// Try to insert a customer profile with non-existent legal_entity_id
	_, err := db.ExecContext(context.Background(), `
		INSERT INTO customer_profiles (id, legal_entity_id, status, default_currency, notes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"cus_orphan", "le_nonexistent2", "active", "USD", "Notes", time.Now().UTC().UnixNano(), time.Now().UTC().UnixNano())
	if err == nil {
		t.Fatal("Expected FK constraint error for non-existent legal_entity_id, got nil")
	}
}

// TestSchemaLegacyCustomersTableDropped verifies the old customers table no longer exists
func TestSchemaLegacyCustomersTableDropped(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	db := store.DB()

	// The customers table should not exist
	var name string
	err := db.QueryRowContext(context.Background(),
		"SELECT name FROM sqlite_master WHERE type='table' AND name='customers'").Scan(&name)
	if err != sql.ErrNoRows {
		t.Fatalf("Expected customers table to be dropped, but it exists or error: %v", err)
	}
}

// TestSchemaLegalEntityIndexes verifies indexes exist for legal_entities
func TestSchemaLegalEntityIndexes(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	db := store.DB()

	// Verify legal_name index exists
	var count int
	err := db.QueryRowContext(context.Background(),
		"SELECT COUNT(*) FROM pragma_index_list('legal_entities') WHERE name LIKE '%legal_name%'").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query legal_entities indexes: %v", err)
	}
	if count == 0 {
		t.Fatal("Expected legal_name index on legal_entities table")
	}
}

// Helper for tests that need to insert a legal entity directly
func insertLegalEntity(t *testing.T, db *sql.DB, entity core.LegalEntity) {
	t.Helper()

	billing := "{}"
	if entity.BillingAddress.Street != "" || entity.BillingAddress.City != "" {
		// Simple JSON for address
		billing = "{\"street\":\"" + entity.BillingAddress.Street + "\",\"city\":\"" + entity.BillingAddress.City + "\"}"
	}

	_, err := db.ExecContext(context.Background(), `
		INSERT INTO legal_entities (id, type, legal_name, trade_name, tax_id, email, phone, website, billing_address, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entity.ID, string(entity.Type), entity.LegalName, entity.TradeName, entity.TaxID,
		entity.Email, entity.Phone, entity.Website, billing,
		entity.CreatedAt.UTC().UnixNano(), entity.UpdatedAt.UTC().UnixNano())
	if err != nil {
		t.Fatalf("insertLegalEntity: %v", err)
	}
}
