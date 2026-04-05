package core

import (
	"strings"
	"testing"
	"time"
)

func TestIssuerProfileIDPrefix(t *testing.T) {
	t.Parallel()

	id := generateIssuerProfileID()
	if !strings.HasPrefix(id, "iss_") {
		t.Fatalf("ID = %q, want iss_ prefix", id)
	}
	if got, want := len(id), len("iss_")+32; got != want {
		t.Fatalf("ID length = %d, want %d", got, want)
	}
}

func TestIssuerProfileIDUniqueness(t *testing.T) {
	t.Parallel()

	id1 := generateIssuerProfileID()
	id2 := generateIssuerProfileID()

	if id1 == id2 {
		t.Fatal("expected distinct issuer profile IDs")
	}
}

func TestNewIssuerProfile_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		params IssuerProfileParams
	}{
		{
			name: "minimal required fields",
			params: IssuerProfileParams{
				LegalEntityID:   "le_test123",
				DefaultCurrency: "USD",
			},
		},
		{
			name: "with default notes",
			params: IssuerProfileParams{
				LegalEntityID:   "le_test456",
				DefaultCurrency: "EUR",
				DefaultNotes:    "Payment terms: Net 30",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			profile, err := NewIssuerProfile(tt.params)
			if err != nil {
				t.Fatalf("NewIssuerProfile() error = %v", err)
			}

			if !strings.HasPrefix(profile.ID, "iss_") {
				t.Fatalf("ID = %q, want iss_ prefix", profile.ID)
			}
			if profile.LegalEntityID != tt.params.LegalEntityID {
				t.Fatalf("LegalEntityID = %q, want %q", profile.LegalEntityID, tt.params.LegalEntityID)
			}
			if profile.DefaultCurrency != tt.params.DefaultCurrency {
				t.Fatalf("DefaultCurrency = %q, want %q", profile.DefaultCurrency, tt.params.DefaultCurrency)
			}
			if profile.DefaultNotes != tt.params.DefaultNotes {
				t.Fatalf("DefaultNotes = %q, want %q", profile.DefaultNotes, tt.params.DefaultNotes)
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

func TestNewIssuerProfile_Errors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		params  IssuerProfileParams
		wantErr string
	}{
		{
			name: "blank legal entity id",
			params: IssuerProfileParams{
				LegalEntityID:   "",
				DefaultCurrency: "USD",
			},
			wantErr: "legal entity id",
		},
		{
			name: "blank default currency",
			params: IssuerProfileParams{
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

			_, err := NewIssuerProfile(tt.params)
			if err == nil {
				t.Fatal("NewIssuerProfile() error = nil, want non-nil")
			}
			if tt.wantErr != "" && !strings.Contains(strings.ToLower(err.Error()), tt.wantErr) {
				t.Fatalf("error = %q, want contains %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestIssuerProfile_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		profile IssuerProfile
		wantErr string
	}{
		{
			name: "valid profile",
			profile: IssuerProfile{
				ID:              "iss_test123",
				LegalEntityID:   "le_test123",
				DefaultCurrency: "USD",
			},
			wantErr: "",
		},
		{
			name: "blank legal entity id",
			profile: IssuerProfile{
				ID:              "iss_test123",
				LegalEntityID:   "",
				DefaultCurrency: "USD",
			},
			wantErr: "legal entity id",
		},
		{
			name: "blank default currency",
			profile: IssuerProfile{
				ID:              "iss_test123",
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

func TestIssuerProfilePatch_ApplyPatch_NilPointer_SkipsField(t *testing.T) {
	t.Parallel()

	original := IssuerProfile{
		ID:              "iss_test123",
		LegalEntityID:   "le_test123",
		DefaultCurrency: "USD",
		DefaultNotes:    "Original notes",
		UpdatedAt:       time.Now().Add(-24 * time.Hour),
	}

	patch := IssuerProfilePatch{}
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
	if profile.DefaultNotes != original.DefaultNotes {
		t.Fatalf("DefaultNotes = %q, want %q (nil pointer should skip)", profile.DefaultNotes, original.DefaultNotes)
	}
	// UpdatedAt should not change when nothing is patched
	if !profile.UpdatedAt.Equal(beforeApply) {
		t.Fatal("UpdatedAt should not change when no fields are patched")
	}
}

func TestIssuerProfilePatch_ApplyPatch_NonNilValue_ReplacesField(t *testing.T) {
	t.Parallel()

	original := IssuerProfile{
		ID:              "iss_test123",
		LegalEntityID:   "le_test123",
		DefaultCurrency: "USD",
		DefaultNotes:    "Original notes",
	}

	newCurrency := "EUR"
	newNotes := "New notes"

	patch := IssuerProfilePatch{
		DefaultCurrency: &newCurrency,
		DefaultNotes:    &newNotes,
	}

	profile := original
	beforeApply := profile.UpdatedAt
	profile.ApplyPatch(patch)

	// All patched fields should be replaced
	if profile.DefaultCurrency != newCurrency {
		t.Fatalf("DefaultCurrency = %q, want %q", profile.DefaultCurrency, newCurrency)
	}
	if profile.DefaultNotes != newNotes {
		t.Fatalf("DefaultNotes = %q, want %q", profile.DefaultNotes, newNotes)
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

func TestIssuerProfile_ValidateDelete(t *testing.T) {
	t.Parallel()

	// ValidateDelete is a seam for future relationship-protection logic.
	profile := IssuerProfile{
		ID:              "iss_test123",
		LegalEntityID:   "le_test123",
		DefaultCurrency: "USD",
	}

	err := profile.ValidateDelete()
	if err != nil {
		t.Fatalf("ValidateDelete() = %v, want nil", err)
	}
}
