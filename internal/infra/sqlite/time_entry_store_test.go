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
// Fixtures
// ---------------------------------------------------------------------------

// newTimeEntryFixture creates a customer profile + service agreement ready
// for time entry tests.  Returns the customer profile ID and service agreement ID.
func newTimeEntryFixture(t *testing.T, db *sql.DB, leID, cusID, saID string) {
	t.Helper()
	insertLegalEntity(t, db, core.LegalEntity{
		ID:        leID,
		Type:      core.EntityTypeCompany,
		LegalName: "Time Entry Test Co",
	})
	insertCustomerProfile(t, db, core.CustomerProfile{
		ID:              cusID,
		LegalEntityID:   leID,
		Status:          core.CustomerProfileStatusActive,
		DefaultCurrency: "USD",
	})
	insertServiceAgreement(t, db, core.ServiceAgreement{
		ID:                saID,
		CustomerProfileID: cusID,
		Name:              "Test Agreement",
		BillingMode:       core.BillingModeHourly,
		HourlyRate:        5000,
		Currency:          "USD",
		Active:            true,
		CreatedAt:         time.Date(2026, 4, 10, 10, 0, 0, 0, time.UTC),
		UpdatedAt:         time.Date(2026, 4, 10, 10, 0, 0, 0, time.UTC),
	})
}

// insertTimeEntry inserts a time entry directly into the DB for testing.
// Does NOT include customer_profile_id (normalized schema).
func insertTimeEntry(t *testing.T, db *sql.DB, entry core.TimeEntry) {
	t.Helper()

	billable := 0
	if entry.Billable {
		billable = 1
	}

	var invoiceID interface{}
	if entry.InvoiceID != "" {
		invoiceID = entry.InvoiceID
	}

	_, err := db.ExecContext(context.Background(), `
INSERT INTO time_entries (id, service_agreement_id, description, hours, billable, invoice_id, date, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.ID,
		entry.ServiceAgreementID,
		entry.Description,
		int64(entry.Hours),
		billable,
		invoiceID,
		entry.Date.UTC().UnixNano(),
		entry.CreatedAt.UTC().UnixNano(),
		entry.UpdatedAt.UTC().UnixNano(),
	)
	if err != nil {
		t.Fatalf("insertTimeEntry: %v", err)
	}
}

// ---------------------------------------------------------------------------
// NewTimeEntryStore
// ---------------------------------------------------------------------------

func TestNewTimeEntryStoreRejectsNil(t *testing.T) {
	t.Parallel()

	s := NewTimeEntryStore(nil)
	if s != nil {
		t.Fatal("NewTimeEntryStore(nil) = non-nil, want nil")
	}
}

// ---------------------------------------------------------------------------
// Save — insert
// ---------------------------------------------------------------------------

func TestTimeEntryStoreSaveInsertsNewEntry(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	newTimeEntryFixture(t, store.DB(), "le_te_ins", "cus_te_ins", "sa_te_ins")
	teStore := NewTimeEntryStore(store)

	now := time.Date(2026, 4, 10, 10, 0, 0, 0, time.UTC)
	hours, _ := core.NewHours(60)
	entry := core.TimeEntry{
		ID:                 "te_ins_001",
		ServiceAgreementID: "sa_te_ins",
		CustomerProfileID:  "cus_te_ins",
		Description:        "Write tests",
		Hours:              hours,
		Billable:           true,
		Date:               now,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	if err := teStore.Save(context.Background(), &entry); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := teStore.GetByID(context.Background(), "te_ins_001")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got.ID != entry.ID {
		t.Errorf("ID = %q, want %q", got.ID, entry.ID)
	}
	if got.ServiceAgreementID != entry.ServiceAgreementID {
		t.Errorf("ServiceAgreementID = %q, want %q", got.ServiceAgreementID, entry.ServiceAgreementID)
	}
	// CustomerProfileID is derived via JOIN — must match fixture
	if got.CustomerProfileID != "cus_te_ins" {
		t.Errorf("CustomerProfileID = %q, want %q", got.CustomerProfileID, "cus_te_ins")
	}
	if got.Description != entry.Description {
		t.Errorf("Description = %q, want %q", got.Description, entry.Description)
	}
	if got.Hours != entry.Hours {
		t.Errorf("Hours = %v, want %v", got.Hours, entry.Hours)
	}
	if !got.Billable {
		t.Errorf("Billable = false, want true")
	}
}

// ---------------------------------------------------------------------------
// Save — update (upsert)
// ---------------------------------------------------------------------------

func TestTimeEntryStoreSaveUpdatesExistingEntry(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	newTimeEntryFixture(t, store.DB(), "le_te_upd", "cus_te_upd", "sa_te_upd")
	teStore := NewTimeEntryStore(store)

	now := time.Date(2026, 4, 10, 10, 0, 0, 0, time.UTC)
	hours60, _ := core.NewHours(60)
	entry := core.TimeEntry{
		ID:                 "te_upd_001",
		ServiceAgreementID: "sa_te_upd",
		CustomerProfileID:  "cus_te_upd",
		Description:        "Initial description",
		Hours:              hours60,
		Billable:           true,
		Date:               now,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	if err := teStore.Save(context.Background(), &entry); err != nil {
		t.Fatalf("initial Save() error = %v", err)
	}

	// Mutate and save again
	hours120, _ := core.NewHours(120)
	entry.Description = "Updated description"
	entry.Hours = hours120
	entry.UpdatedAt = now.Add(time.Hour)

	if err := teStore.Save(context.Background(), &entry); err != nil {
		t.Fatalf("update Save() error = %v", err)
	}

	got, err := teStore.GetByID(context.Background(), "te_upd_001")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got.Description != "Updated description" {
		t.Errorf("Description = %q, want %q", got.Description, "Updated description")
	}
	if got.Hours != hours120 {
		t.Errorf("Hours = %v, want %v", got.Hours, hours120)
	}
}

// ---------------------------------------------------------------------------
// GetByID — not found
// ---------------------------------------------------------------------------

func TestTimeEntryStoreGetByIDReturnsNotFoundForUnknownID(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	teStore := NewTimeEntryStore(store)

	_, err := teStore.GetByID(context.Background(), "te_nonexistent")
	if err == nil {
		t.Fatal("GetByID() expected error, got nil")
	}
	if !errors.Is(err, app.ErrTimeEntryNotFound) {
		t.Errorf("GetByID() error = %v, want %v", err, app.ErrTimeEntryNotFound)
	}
}

// ---------------------------------------------------------------------------
// GetByID — CustomerProfileID is JOIN-derived
// ---------------------------------------------------------------------------

func TestTimeEntryStoreGetByIDDerivesCustomerProfileIDFromJoin(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	newTimeEntryFixture(t, store.DB(), "le_te_join", "cus_te_join", "sa_te_join")
	teStore := NewTimeEntryStore(store)

	now := time.Date(2026, 4, 10, 10, 0, 0, 0, time.UTC)
	hours, _ := core.NewHours(30)
	insertTimeEntry(t, store.DB(), core.TimeEntry{
		ID:                 "te_join_001",
		ServiceAgreementID: "sa_te_join",
		Description:        "Join test",
		Hours:              hours,
		Billable:           true,
		Date:               now,
		CreatedAt:          now,
		UpdatedAt:          now,
	})

	got, err := teStore.GetByID(context.Background(), "te_join_001")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	// CustomerProfileID must come from service_agreements JOIN — not stored on time_entries
	if got.CustomerProfileID != "cus_te_join" {
		t.Errorf("CustomerProfileID = %q, want %q (JOIN-derived)", got.CustomerProfileID, "cus_te_join")
	}
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestTimeEntryStoreDeleteRemovesEntry(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	newTimeEntryFixture(t, store.DB(), "le_te_del", "cus_te_del", "sa_te_del")
	teStore := NewTimeEntryStore(store)

	now := time.Date(2026, 4, 10, 10, 0, 0, 0, time.UTC)
	hours, _ := core.NewHours(30)
	insertTimeEntry(t, store.DB(), core.TimeEntry{
		ID:                 "te_del_001",
		ServiceAgreementID: "sa_te_del",
		Description:        "To delete",
		Hours:              hours,
		Billable:           true,
		Date:               now,
		CreatedAt:          now,
		UpdatedAt:          now,
	})

	if err := teStore.Delete(context.Background(), "te_del_001"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err := teStore.GetByID(context.Background(), "te_del_001")
	if err == nil {
		t.Fatal("GetByID() expected error after delete, got nil")
	}
	if !errors.Is(err, app.ErrTimeEntryNotFound) {
		t.Errorf("GetByID() error = %v, want %v", err, app.ErrTimeEntryNotFound)
	}
}

// ---------------------------------------------------------------------------
// ListByCustomerProfile
// ---------------------------------------------------------------------------

func TestTimeEntryStoreListByCustomerProfileReturnsEntries(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	newTimeEntryFixture(t, store.DB(), "le_te_list", "cus_te_list", "sa_te_list")
	// Also create a second customer to verify filtering
	newTimeEntryFixture(t, store.DB(), "le_te_list2", "cus_te_list2", "sa_te_list2")
	teStore := NewTimeEntryStore(store)

	now := time.Date(2026, 4, 10, 10, 0, 0, 0, time.UTC)
	hours, _ := core.NewHours(60)
	insertTimeEntry(t, store.DB(), core.TimeEntry{
		ID:                 "te_list_001",
		ServiceAgreementID: "sa_te_list",
		Description:        "Entry one",
		Hours:              hours,
		Billable:           true,
		Date:               now,
		CreatedAt:          now,
		UpdatedAt:          now,
	})
	insertTimeEntry(t, store.DB(), core.TimeEntry{
		ID:                 "te_list_002",
		ServiceAgreementID: "sa_te_list",
		Description:        "Entry two",
		Hours:              hours,
		Billable:           false,
		Date:               now.Add(time.Minute),
		CreatedAt:          now.Add(time.Minute),
		UpdatedAt:          now.Add(time.Minute),
	})
	// This one belongs to a different customer
	insertTimeEntry(t, store.DB(), core.TimeEntry{
		ID:                 "te_list_other",
		ServiceAgreementID: "sa_te_list2",
		Description:        "Other customer entry",
		Hours:              hours,
		Billable:           true,
		Date:               now,
		CreatedAt:          now,
		UpdatedAt:          now,
	})

	results, err := teStore.ListByCustomerProfile(context.Background(), "cus_te_list")
	if err != nil {
		t.Fatalf("ListByCustomerProfile() error = %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("ListByCustomerProfile() len = %d, want 2", len(results))
	}

	// Verify customer profile IDs are populated via JOIN
	for _, r := range results {
		if r.CustomerProfileID != "cus_te_list" {
			t.Errorf("CustomerProfileID = %q, want %q", r.CustomerProfileID, "cus_te_list")
		}
	}
}

func TestTimeEntryStoreListByCustomerProfileReturnsEmptyForUnknownCustomer(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	teStore := NewTimeEntryStore(store)

	results, err := teStore.ListByCustomerProfile(context.Background(), "cus_unknown")
	if err != nil {
		t.Fatalf("ListByCustomerProfile() error = %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("ListByCustomerProfile() len = %d, want 0", len(results))
	}
}

// ---------------------------------------------------------------------------
// ListUnbilled
// ---------------------------------------------------------------------------

func TestTimeEntryStoreListUnbilledReturnsOnlyUnbilledEntries(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	newTimeEntryFixture(t, store.DB(), "le_te_ub", "cus_te_ub", "sa_te_ub")
	teStore := NewTimeEntryStore(store)

	now := time.Date(2026, 4, 10, 10, 0, 0, 0, time.UTC)
	hours, _ := core.NewHours(60)

	// Unbilled entry (no invoice_id)
	insertTimeEntry(t, store.DB(), core.TimeEntry{
		ID:                 "te_ub_open",
		ServiceAgreementID: "sa_te_ub",
		Description:        "Unbilled work",
		Hours:              hours,
		Billable:           true,
		Date:               now,
		CreatedAt:          now,
		UpdatedAt:          now,
	})
	// Billed entry (has invoice_id)
	insertTimeEntry(t, store.DB(), core.TimeEntry{
		ID:                 "te_ub_billed",
		ServiceAgreementID: "sa_te_ub",
		Description:        "Billed work",
		Hours:              hours,
		Billable:           true,
		InvoiceID:          "inv_123",
		Date:               now.Add(time.Minute),
		CreatedAt:          now.Add(time.Minute),
		UpdatedAt:          now.Add(time.Minute),
	})

	results, err := teStore.ListUnbilled(context.Background(), "cus_te_ub")
	if err != nil {
		t.Fatalf("ListUnbilled() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("ListUnbilled() len = %d, want 1", len(results))
	}
	if results[0].ID != "te_ub_open" {
		t.Errorf("ListUnbilled() entry ID = %q, want %q", results[0].ID, "te_ub_open")
	}
	if results[0].InvoiceID != "" {
		t.Errorf("ListUnbilled() entry InvoiceID = %q, want empty", results[0].InvoiceID)
	}
}

func TestTimeEntryStoreListUnbilledReturnsEmptyWhenAllBilled(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	newTimeEntryFixture(t, store.DB(), "le_te_allbilled", "cus_te_allbilled", "sa_te_allbilled")
	teStore := NewTimeEntryStore(store)

	now := time.Date(2026, 4, 10, 10, 0, 0, 0, time.UTC)
	hours, _ := core.NewHours(60)
	insertTimeEntry(t, store.DB(), core.TimeEntry{
		ID:                 "te_alb_001",
		ServiceAgreementID: "sa_te_allbilled",
		Description:        "Already billed",
		Hours:              hours,
		Billable:           true,
		InvoiceID:          "inv_456",
		Date:               now,
		CreatedAt:          now,
		UpdatedAt:          now,
	})

	results, err := teStore.ListUnbilled(context.Background(), "cus_te_allbilled")
	if err != nil {
		t.Fatalf("ListUnbilled() error = %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("ListUnbilled() len = %d, want 0 (all billed)", len(results))
	}
}
