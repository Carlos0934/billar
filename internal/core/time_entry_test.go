package core

import (
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// NewTimeEntry — constructor invariants
// ---------------------------------------------------------------------------

func TestNewTimeEntry_ValidBillableEntry(t *testing.T) {
	t.Parallel()

	hours, err := NewHours(15000) // 1.5000 hours
	if err != nil {
		t.Fatalf("NewHours: %v", err)
	}

	params := TimeEntryParams{
		CustomerProfileID:  "cus_abc123",
		ServiceAgreementID: "sa_xyz789",
		Description:        "Initial implementation",
		Hours:              hours,
		Billable:           true,
		Date:               time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC),
	}

	entry, err := NewTimeEntry(params)
	if err != nil {
		t.Fatalf("NewTimeEntry() error = %v, want nil", err)
	}

	if !strings.HasPrefix(entry.ID, "te_") {
		t.Fatalf("ID = %q, want te_ prefix", entry.ID)
	}
	if entry.CustomerProfileID != params.CustomerProfileID {
		t.Fatalf("CustomerProfileID = %q, want %q", entry.CustomerProfileID, params.CustomerProfileID)
	}
	if entry.ServiceAgreementID != params.ServiceAgreementID {
		t.Fatalf("ServiceAgreementID = %q, want %q", entry.ServiceAgreementID, params.ServiceAgreementID)
	}
	if entry.Hours != hours {
		t.Fatalf("Hours = %d, want %d", entry.Hours, hours)
	}
	if !entry.Billable {
		t.Fatal("Billable = false, want true")
	}
	if entry.InvoiceID != "" {
		t.Fatalf("InvoiceID = %q, want empty on creation", entry.InvoiceID)
	}
	if entry.CreatedAt.IsZero() {
		t.Fatal("CreatedAt is zero")
	}
	if !entry.CreatedAt.Equal(entry.UpdatedAt) {
		t.Fatalf("CreatedAt != UpdatedAt on construction: %v vs %v", entry.CreatedAt, entry.UpdatedAt)
	}
}

func TestNewTimeEntry_NonBillableRequiresNoAgreement(t *testing.T) {
	t.Parallel()

	hours, _ := NewHours(10000)
	params := TimeEntryParams{
		CustomerProfileID: "cus_abc123",
		Description:       "Internal meeting",
		Hours:             hours,
		Billable:          false,
		Date:              time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC),
	}

	entry, err := NewTimeEntry(params)
	if err != nil {
		t.Fatalf("NewTimeEntry() for non-billable error = %v, want nil", err)
	}
	if entry.ServiceAgreementID != "" {
		t.Fatalf("ServiceAgreementID = %q, want empty for non-billable", entry.ServiceAgreementID)
	}
}

func TestNewTimeEntry_Errors(t *testing.T) {
	t.Parallel()

	validHours, _ := NewHours(15000)
	validDate := time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		params  TimeEntryParams
		wantErr string
	}{
		{
			name: "billable entry missing service agreement",
			params: TimeEntryParams{
				CustomerProfileID: "cus_abc123",
				Description:       "Some work",
				Hours:             validHours,
				Billable:          true,
				// ServiceAgreementID intentionally omitted
				Date: validDate,
			},
			wantErr: "service agreement",
		},
		{
			name: "missing customer profile id",
			params: TimeEntryParams{
				CustomerProfileID:  "",
				ServiceAgreementID: "sa_xyz789",
				Description:        "Some work",
				Hours:              validHours,
				Billable:           true,
				Date:               validDate,
			},
			wantErr: "customer profile",
		},
		{
			name: "non-positive hours",
			params: TimeEntryParams{
				CustomerProfileID:  "cus_abc123",
				ServiceAgreementID: "sa_xyz789",
				Description:        "Some work",
				Hours:              Hours(0), // zero — invalid
				Billable:           true,
				Date:               validDate,
			},
			wantErr: "hours",
		},
		{
			name: "missing description",
			params: TimeEntryParams{
				CustomerProfileID:  "cus_abc123",
				ServiceAgreementID: "sa_xyz789",
				Description:        "",
				Hours:              validHours,
				Billable:           true,
				Date:               validDate,
			},
			wantErr: "description",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := NewTimeEntry(tt.params)
			if err == nil {
				t.Fatalf("NewTimeEntry() error = nil, want error containing %q", tt.wantErr)
			}
			if !strings.Contains(strings.ToLower(err.Error()), tt.wantErr) {
				t.Fatalf("error = %q, want contains %q", err.Error(), tt.wantErr)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TimeEntry.Update — pre-invoice mutation and lock behaviour
// ---------------------------------------------------------------------------

func TestTimeEntry_UpdateSucceedsBeforeInvoicing(t *testing.T) {
	t.Parallel()

	hours, _ := NewHours(15000)
	entry, err := NewTimeEntry(TimeEntryParams{
		CustomerProfileID:  "cus_abc123",
		ServiceAgreementID: "sa_xyz789",
		Description:        "Original description",
		Hours:              hours,
		Billable:           true,
		Date:               time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("NewTimeEntry: %v", err)
	}

	newHours, _ := NewHours(20000)
	if err := entry.Update("Updated description", newHours); err != nil {
		t.Fatalf("Update() error = %v, want nil", err)
	}
	if entry.Description != "Updated description" {
		t.Fatalf("Description = %q, want Updated description", entry.Description)
	}
	if entry.Hours != newHours {
		t.Fatalf("Hours = %d, want %d", entry.Hours, newHours)
	}
}

func TestTimeEntry_UpdateFailsAfterInvoiceAssignment(t *testing.T) {
	t.Parallel()

	hours, _ := NewHours(15000)
	entry, err := NewTimeEntry(TimeEntryParams{
		CustomerProfileID:  "cus_abc123",
		ServiceAgreementID: "sa_xyz789",
		Description:        "Work done",
		Hours:              hours,
		Billable:           true,
		Date:               time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("NewTimeEntry: %v", err)
	}

	// Assign invoice first
	if err := entry.AssignToInvoice("inv_001"); err != nil {
		t.Fatalf("AssignToInvoice: %v", err)
	}

	// Now update must be rejected
	newHours, _ := NewHours(20000)
	err = entry.Update("New description", newHours)
	if err == nil {
		t.Fatal("Update() error = nil, want locked error for invoiced entry")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "lock") {
		t.Fatalf("error = %q, want contains 'lock'", err.Error())
	}
}

// ---------------------------------------------------------------------------
// TimeEntry.UnassignFromInvoice
// ---------------------------------------------------------------------------

func TestTimeEntry_UnassignFromInvoiceClearsIDAndRestoresMutability(t *testing.T) {
	t.Parallel()

	hours, _ := NewHours(15000)
	entry, err := NewTimeEntry(TimeEntryParams{
		CustomerProfileID:  "cus_abc123",
		ServiceAgreementID: "sa_xyz789",
		Description:        "Work",
		Hours:              hours,
		Billable:           true,
		Date:               time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("NewTimeEntry: %v", err)
	}
	if err := entry.AssignToInvoice("inv_001"); err != nil {
		t.Fatalf("AssignToInvoice: %v", err)
	}

	// Unassign
	entry.UnassignFromInvoice()

	if entry.InvoiceID != "" {
		t.Fatalf("InvoiceID = %q, want empty after UnassignFromInvoice", entry.InvoiceID)
	}

	// Must be mutable again
	newHours, _ := NewHours(30000)
	if err := entry.Update("After unassign", newHours); err != nil {
		t.Fatalf("Update() after UnassignFromInvoice error = %v, want nil", err)
	}
}

// ---------------------------------------------------------------------------
// TimeEntry.Update — invariant re-validation
// ---------------------------------------------------------------------------

func TestTimeEntry_Update_RejectsBlankDescription(t *testing.T) {
	t.Parallel()

	hours, _ := NewHours(15000)
	entry, err := NewTimeEntry(TimeEntryParams{
		CustomerProfileID:  "cus_abc123",
		ServiceAgreementID: "sa_xyz789",
		Description:        "Original description",
		Hours:              hours,
		Billable:           true,
		Date:               time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("NewTimeEntry: %v", err)
	}

	err = entry.Update("", hours)
	if err == nil {
		t.Fatal("Update() error = nil, want error for blank description")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "description") {
		t.Fatalf("error = %q, want contains 'description'", err.Error())
	}
	// The original description must remain unchanged.
	if entry.Description != "Original description" {
		t.Fatalf("Description = %q after failed update, want unchanged Original description", entry.Description)
	}
}

func TestTimeEntry_Update_RejectsWhitespaceOnlyDescription(t *testing.T) {
	t.Parallel()

	hours, _ := NewHours(15000)
	entry, err := NewTimeEntry(TimeEntryParams{
		CustomerProfileID:  "cus_abc123",
		ServiceAgreementID: "sa_xyz789",
		Description:        "Original",
		Hours:              hours,
		Billable:           true,
		Date:               time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("NewTimeEntry: %v", err)
	}

	err = entry.Update("   ", hours)
	if err == nil {
		t.Fatal("Update() error = nil, want error for whitespace-only description")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "description") {
		t.Fatalf("error = %q, want contains 'description'", err.Error())
	}
}

func TestTimeEntry_Update_RejectsNonPositiveHours(t *testing.T) {
	t.Parallel()

	hours, _ := NewHours(15000)
	entry, err := NewTimeEntry(TimeEntryParams{
		CustomerProfileID:  "cus_abc123",
		ServiceAgreementID: "sa_xyz789",
		Description:        "Original",
		Hours:              hours,
		Billable:           true,
		Date:               time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("NewTimeEntry: %v", err)
	}

	err = entry.Update("Updated", Hours(0))
	if err == nil {
		t.Fatal("Update() error = nil, want error for zero hours")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "hours") {
		t.Fatalf("error = %q, want contains 'hours'", err.Error())
	}
	// Hours must remain unchanged.
	if entry.Hours != hours {
		t.Fatalf("Hours changed after failed update: got %d, want %d", entry.Hours, hours)
	}
}

// ---------------------------------------------------------------------------
// ID uniqueness
// ---------------------------------------------------------------------------

func TestNewTimeEntry_IDUniqueness(t *testing.T) {
	t.Parallel()

	hours, _ := NewHours(10000)
	params := TimeEntryParams{
		CustomerProfileID:  "cus_abc123",
		ServiceAgreementID: "sa_xyz789",
		Description:        "Work",
		Hours:              hours,
		Billable:           true,
		Date:               time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC),
	}

	e1, _ := NewTimeEntry(params)
	e2, _ := NewTimeEntry(params)
	if e1.ID == e2.ID {
		t.Fatal("expected distinct time entry IDs, got the same")
	}
}
