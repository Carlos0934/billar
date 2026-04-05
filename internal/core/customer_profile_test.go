package core

import (
	"strings"
	"testing"
	"time"
)

func TestCustomerProfileStatus_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		got  CustomerProfileStatus
		want bool
	}{
		{name: "active", got: CustomerProfileStatusActive, want: true},
		{name: "inactive", got: CustomerProfileStatusInactive, want: true},
		{name: "archived", got: CustomerProfileStatus("archived"), want: false},
		{name: "empty", got: CustomerProfileStatus(""), want: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.got.IsValid(); got != tt.want {
				t.Fatalf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCustomerProfileIDPrefix(t *testing.T) {
	t.Parallel()

	id := generateCustomerProfileID()
	if !strings.HasPrefix(id, "cus_") {
		t.Fatalf("ID = %q, want cus_ prefix", id)
	}
	if got, want := len(id), len("cus_")+32; got != want {
		t.Fatalf("ID length = %d, want %d", got, want)
	}
}

func TestCustomerProfileIDUniqueness(t *testing.T) {
	t.Parallel()

	id1 := generateCustomerProfileID()
	id2 := generateCustomerProfileID()

	if id1 == id2 {
		t.Fatal("expected distinct customer profile IDs")
	}
}

func TestNewCustomerProfile_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		params CustomerProfileParams
	}{
		{
			name: "minimal required fields",
			params: CustomerProfileParams{
				LegalEntityID:   "le_test123",
				DefaultCurrency: "USD",
			},
		},
		{
			name: "with notes",
			params: CustomerProfileParams{
				LegalEntityID:   "le_test456",
				DefaultCurrency: "EUR",
				Notes:           "VIP customer",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			profile, err := NewCustomerProfile(tt.params)
			if err != nil {
				t.Fatalf("NewCustomerProfile() error = %v", err)
			}

			if !strings.HasPrefix(profile.ID, "cus_") {
				t.Fatalf("ID = %q, want cus_ prefix", profile.ID)
			}
			if profile.LegalEntityID != tt.params.LegalEntityID {
				t.Fatalf("LegalEntityID = %q, want %q", profile.LegalEntityID, tt.params.LegalEntityID)
			}
			if profile.DefaultCurrency != tt.params.DefaultCurrency {
				t.Fatalf("DefaultCurrency = %q, want %q", profile.DefaultCurrency, tt.params.DefaultCurrency)
			}
			if profile.Notes != tt.params.Notes {
				t.Fatalf("Notes = %q, want %q", profile.Notes, tt.params.Notes)
			}
			if profile.Status != CustomerProfileStatusActive {
				t.Fatalf("Status = %q, want %q", profile.Status, CustomerProfileStatusActive)
			}
			if profile.CreatedAt.IsZero() {
				t.Fatal("CreatedAt is zero")
			}
			if profile.UpdatedAt.IsZero() {
				t.Fatal("UpdatedAt is zero")
			}
			if !profile.CreatedAt.Equal(profile.UpdatedAt) {
				t.Fatalf("CreatedAt = %v, UpdatedAt = %v, want equal", profile.CreatedAt, profile.UpdatedAt)
			}
		})
	}
}

func TestNewCustomerProfile_Errors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		params  CustomerProfileParams
		wantErr string
	}{
		{
			name: "blank legal entity id",
			params: CustomerProfileParams{
				LegalEntityID:   "",
				DefaultCurrency: "USD",
			},
			wantErr: "legal entity id",
		},
		{
			name: "blank default currency",
			params: CustomerProfileParams{
				LegalEntityID:   "le_test123",
				DefaultCurrency: "",
			},
			wantErr: "default currency",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := NewCustomerProfile(tt.params)
			if err == nil {
				t.Fatal("NewCustomerProfile() error = nil, want non-nil")
			}
			if tt.wantErr != "" && !strings.Contains(strings.ToLower(err.Error()), tt.wantErr) {
				t.Fatalf("error = %q, want contains %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestCustomerProfile_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		profile CustomerProfile
		wantErr string
	}{
		{
			name: "valid profile",
			profile: CustomerProfile{
				ID:              "cus_test123",
				LegalEntityID:   "le_test123",
				DefaultCurrency: "USD",
				Status:          CustomerProfileStatusActive,
			},
			wantErr: "",
		},
		{
			name: "blank legal entity id",
			profile: CustomerProfile{
				ID:              "cus_test123",
				LegalEntityID:   "",
				DefaultCurrency: "USD",
				Status:          CustomerProfileStatusActive,
			},
			wantErr: "legal entity id",
		},
		{
			name: "blank default currency",
			profile: CustomerProfile{
				ID:              "cus_test123",
				LegalEntityID:   "le_test123",
				DefaultCurrency: "",
				Status:          CustomerProfileStatusActive,
			},
			wantErr: "default currency",
		},
		{
			name: "invalid status",
			profile: CustomerProfile{
				ID:              "cus_test123",
				LegalEntityID:   "le_test123",
				DefaultCurrency: "USD",
				Status:          CustomerProfileStatus("invalid"),
			},
			wantErr: "customer profile status",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.profile.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() = %v, want nil", err)
				}
			} else {
				if err == nil {
					t.Fatal("Validate() = nil, want non-nil")
				}
				if !strings.Contains(strings.ToLower(err.Error()), tt.wantErr) {
					t.Fatalf("Validate() error = %q, want contains %q", err.Error(), tt.wantErr)
				}
			}
		})
	}
}

func TestCustomerProfilePatch_ApplyPatch_NilPointer_SkipsField(t *testing.T) {
	t.Parallel()

	original := CustomerProfile{
		ID:              "cus_test123",
		LegalEntityID:   "le_test123",
		DefaultCurrency: "USD",
		Status:          CustomerProfileStatusActive,
		Notes:           "Original notes",
		UpdatedAt:       time.Now().Add(-24 * time.Hour),
	}

	patch := CustomerProfilePatch{}
	profile := original
	beforeApply := profile.UpdatedAt
	profile.ApplyPatch(patch)

	// All fields should remain unchanged
	if profile.LegalEntityID != original.LegalEntityID {
		t.Fatalf("LegalEntityID = %q, want %q (nil pointer should skip)", profile.LegalEntityID, original.LegalEntityID)
	}
	if profile.DefaultCurrency != original.DefaultCurrency {
		t.Fatalf("DefaultCurrency = %q, want %q (nil pointer should skip)", profile.DefaultCurrency, original.DefaultCurrency)
	}
	if profile.Status != original.Status {
		t.Fatalf("Status = %q, want %q (nil pointer should skip)", profile.Status, original.Status)
	}
	if profile.Notes != original.Notes {
		t.Fatalf("Notes = %q, want %q (nil pointer should skip)", profile.Notes, original.Notes)
	}
	// UpdatedAt should not change when nothing is patched
	if !profile.UpdatedAt.Equal(beforeApply) {
		t.Fatal("UpdatedAt should not change when no fields are patched")
	}
}

func TestCustomerProfilePatch_ApplyPatch_NonNilValue_ReplacesField(t *testing.T) {
	t.Parallel()

	original := CustomerProfile{
		ID:              "cus_test123",
		LegalEntityID:   "le_test123",
		DefaultCurrency: "USD",
		Status:          CustomerProfileStatusActive,
		Notes:           "Original notes",
	}

	newStatus := CustomerProfileStatusInactive
	newCurrency := "EUR"
	newNotes := "New notes"

	patch := CustomerProfilePatch{
		Status:          &newStatus,
		DefaultCurrency: &newCurrency,
		Notes:           &newNotes,
	}

	profile := original
	beforeApply := profile.UpdatedAt
	profile.ApplyPatch(patch)

	// All patched fields should be replaced
	if profile.Status != newStatus {
		t.Fatalf("Status = %q, want %q", profile.Status, newStatus)
	}
	if profile.DefaultCurrency != newCurrency {
		t.Fatalf("DefaultCurrency = %q, want %q", profile.DefaultCurrency, newCurrency)
	}
	if profile.Notes != newNotes {
		t.Fatalf("Notes = %q, want %q", profile.Notes, newNotes)
	}

	// LegalEntityID should remain unchanged
	if profile.LegalEntityID != original.LegalEntityID {
		t.Fatalf("LegalEntityID = %q, want %q (should not change)", profile.LegalEntityID, original.LegalEntityID)
	}

	// UpdatedAt should be updated
	if profile.UpdatedAt.Before(beforeApply) || profile.UpdatedAt.Equal(beforeApply) {
		t.Fatal("UpdatedAt should be more recent after patching")
	}
}

func TestCustomerProfile_ValidateDelete(t *testing.T) {
	t.Parallel()

	// ValidateDelete is a seam for future relationship-protection logic.
	profile := CustomerProfile{
		ID:              "cus_test123",
		LegalEntityID:   "le_test123",
		DefaultCurrency: "USD",
		Status:          CustomerProfileStatusActive,
	}

	err := profile.ValidateDelete()
	if err != nil {
		t.Fatalf("ValidateDelete() = %v, want nil", err)
	}
}

func TestCustomerProfile_CanReceiveInvoices(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		profile CustomerProfile
		want    bool
	}{
		{name: "active", profile: CustomerProfile{Status: CustomerProfileStatusActive}, want: true},
		{name: "inactive", profile: CustomerProfile{Status: CustomerProfileStatusInactive}, want: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.profile.CanReceiveInvoices(); got != tt.want {
				t.Fatalf("CanReceiveInvoices() = %v, want %v", got, tt.want)
			}
		})
	}
}
