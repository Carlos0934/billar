package core

import (
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// BillingMode.IsValid
// ---------------------------------------------------------------------------

func TestBillingMode_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		got  BillingMode
		want bool
	}{
		{name: "hourly", got: BillingModeHourly, want: true},
		{name: "retainer", got: BillingMode("retainer"), want: false},
		{name: "empty", got: BillingMode(""), want: false},
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

// ---------------------------------------------------------------------------
// NewServiceAgreement — success
// ---------------------------------------------------------------------------

func TestNewServiceAgreement_Success(t *testing.T) {
	t.Parallel()

	params := ServiceAgreementParams{
		CustomerProfileID: "cus_abc123",
		Name:              "Monthly Support",
		Description:       "Basic support package",
		BillingMode:       BillingModeHourly,
		HourlyRate:        1000,
		Currency:          "USD",
	}

	sa, err := NewServiceAgreement(params)
	if err != nil {
		t.Fatalf("NewServiceAgreement() error = %v", err)
	}

	if !strings.HasPrefix(sa.ID, "sa_") {
		t.Fatalf("ID = %q, want sa_ prefix", sa.ID)
	}
	if sa.CustomerProfileID != params.CustomerProfileID {
		t.Fatalf("CustomerProfileID = %q, want %q", sa.CustomerProfileID, params.CustomerProfileID)
	}
	if sa.Name != params.Name {
		t.Fatalf("Name = %q, want %q", sa.Name, params.Name)
	}
	if sa.BillingMode != params.BillingMode {
		t.Fatalf("BillingMode = %q, want %q", sa.BillingMode, params.BillingMode)
	}
	if sa.HourlyRate != params.HourlyRate {
		t.Fatalf("HourlyRate = %d, want %d", sa.HourlyRate, params.HourlyRate)
	}
	if sa.Currency != params.Currency {
		t.Fatalf("Currency = %q, want %q", sa.Currency, params.Currency)
	}
	if !sa.Active {
		t.Fatal("Active = false, want true")
	}
	if sa.CreatedAt.IsZero() {
		t.Fatal("CreatedAt is zero")
	}
	if sa.UpdatedAt.IsZero() {
		t.Fatal("UpdatedAt is zero")
	}
	if !sa.CreatedAt.Equal(sa.UpdatedAt) {
		t.Fatalf("CreatedAt = %v, UpdatedAt = %v, want equal on construction", sa.CreatedAt, sa.UpdatedAt)
	}
}

// ---------------------------------------------------------------------------
// NewServiceAgreement — validation errors
// ---------------------------------------------------------------------------

func TestNewServiceAgreement_Errors(t *testing.T) {
	t.Parallel()

	validBase := ServiceAgreementParams{
		CustomerProfileID: "cus_abc123",
		Name:              "Monthly Support",
		BillingMode:       BillingModeHourly,
		HourlyRate:        1000,
		Currency:          "USD",
	}

	tests := []struct {
		name    string
		params  ServiceAgreementParams
		wantErr string
	}{
		{
			name:    "zero hourly rate",
			params:  func() ServiceAgreementParams { p := validBase; p.HourlyRate = 0; return p }(),
			wantErr: "hourly rate",
		},
		{
			name:    "negative hourly rate",
			params:  func() ServiceAgreementParams { p := validBase; p.HourlyRate = -500; return p }(),
			wantErr: "hourly rate",
		},
		{
			name:    "unsupported billing mode",
			params:  func() ServiceAgreementParams { p := validBase; p.BillingMode = BillingMode("retainer"); return p }(),
			wantErr: "billing mode",
		},
		{
			name:    "missing customer profile id",
			params:  func() ServiceAgreementParams { p := validBase; p.CustomerProfileID = ""; return p }(),
			wantErr: "customer profile",
		},
		{
			name:    "missing name",
			params:  func() ServiceAgreementParams { p := validBase; p.Name = ""; return p }(),
			wantErr: "name",
		},
		{
			name:    "missing currency",
			params:  func() ServiceAgreementParams { p := validBase; p.Currency = ""; return p }(),
			wantErr: "currency",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := NewServiceAgreement(tt.params)
			if err == nil {
				t.Fatal("NewServiceAgreement() error = nil, want non-nil")
			}
			if !strings.Contains(strings.ToLower(err.Error()), tt.wantErr) {
				t.Fatalf("error = %q, want contains %q", err.Error(), tt.wantErr)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ServiceAgreement.UpdateRate
// ---------------------------------------------------------------------------

func TestServiceAgreement_UpdateRate(t *testing.T) {
	t.Parallel()

	makeAgreement := func() ServiceAgreement {
		sa, err := NewServiceAgreement(ServiceAgreementParams{
			CustomerProfileID: "cus_abc123",
			Name:              "Monthly Support",
			BillingMode:       BillingModeHourly,
			HourlyRate:        1000,
			Currency:          "USD",
		})
		if err != nil {
			panic(err)
		}
		return sa
	}

	t.Run("positive rate succeeds and refreshes UpdatedAt", func(t *testing.T) {
		t.Parallel()

		sa := makeAgreement()
		before := sa.UpdatedAt
		// Small sleep to ensure time advances
		time.Sleep(time.Millisecond)

		if err := sa.UpdateRate(1500); err != nil {
			t.Fatalf("UpdateRate(1500) error = %v", err)
		}
		if sa.HourlyRate != 1500 {
			t.Fatalf("HourlyRate = %d, want 1500", sa.HourlyRate)
		}
		if !sa.UpdatedAt.After(before) {
			t.Fatalf("UpdatedAt = %v, want after %v", sa.UpdatedAt, before)
		}
	})

	t.Run("zero rate returns error", func(t *testing.T) {
		t.Parallel()

		sa := makeAgreement()
		if err := sa.UpdateRate(0); err == nil {
			t.Fatal("UpdateRate(0) error = nil, want non-nil")
		}
		// Rate must remain unchanged
		if sa.HourlyRate != 1000 {
			t.Fatalf("HourlyRate = %d, want 1000 (unchanged)", sa.HourlyRate)
		}
	})

	t.Run("negative rate returns error", func(t *testing.T) {
		t.Parallel()

		sa := makeAgreement()
		if err := sa.UpdateRate(-100); err == nil {
			t.Fatal("UpdateRate(-100) error = nil, want non-nil")
		}
		if sa.HourlyRate != 1000 {
			t.Fatalf("HourlyRate = %d, want 1000 (unchanged)", sa.HourlyRate)
		}
	})
}

// ---------------------------------------------------------------------------
// ServiceAgreement.Activate / Deactivate
// ---------------------------------------------------------------------------

func TestServiceAgreement_ActivateDeactivate(t *testing.T) {
	t.Parallel()

	makeAgreement := func() ServiceAgreement {
		sa, err := NewServiceAgreement(ServiceAgreementParams{
			CustomerProfileID: "cus_abc123",
			Name:              "Monthly Support",
			BillingMode:       BillingModeHourly,
			HourlyRate:        1000,
			Currency:          "USD",
		})
		if err != nil {
			panic(err)
		}
		return sa
	}

	t.Run("Deactivate sets Active=false and refreshes UpdatedAt", func(t *testing.T) {
		t.Parallel()

		sa := makeAgreement()
		if !sa.Active {
			t.Fatal("pre-condition: expected Active=true")
		}
		before := sa.UpdatedAt
		time.Sleep(time.Millisecond)

		sa.Deactivate()

		if sa.Active {
			t.Fatal("Active = true after Deactivate(), want false")
		}
		if !sa.UpdatedAt.After(before) {
			t.Fatalf("UpdatedAt = %v, want after %v", sa.UpdatedAt, before)
		}
	})

	t.Run("Activate sets Active=true and refreshes UpdatedAt", func(t *testing.T) {
		t.Parallel()

		sa := makeAgreement()
		sa.Deactivate()

		before := sa.UpdatedAt
		time.Sleep(time.Millisecond)

		sa.Activate()

		if !sa.Active {
			t.Fatal("Active = false after Activate(), want true")
		}
		if !sa.UpdatedAt.After(before) {
			t.Fatalf("UpdatedAt = %v, want after %v", sa.UpdatedAt, before)
		}
	})
}

// ---------------------------------------------------------------------------
// ID uniqueness
// ---------------------------------------------------------------------------

func TestServiceAgreementIDUniqueness(t *testing.T) {
	t.Parallel()

	params := ServiceAgreementParams{
		CustomerProfileID: "cus_abc123",
		Name:              "Monthly Support",
		BillingMode:       BillingModeHourly,
		HourlyRate:        1000,
		Currency:          "USD",
	}

	sa1, _ := NewServiceAgreement(params)
	sa2, _ := NewServiceAgreement(params)
	if sa1.ID == sa2.ID {
		t.Fatal("expected distinct service agreement IDs")
	}
}
