package billingvalues_test

import (
	"errors"
	"testing"

	"github.com/Carlos0934/billar/internal/domain/billing/billing_values"
)

func TestNewEmailAddress(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "valid", input: "billing@example.com", want: "billing@example.com"},
		{name: "rejects malformed", input: "billing.example.com", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			email, err := billingvalues.NewEmailAddress(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				if !errors.Is(err, billingvalues.ErrEmailAddressInvalid) {
					t.Fatalf("err = %v, want %v", err, billingvalues.ErrEmailAddressInvalid)
				}
				return
			}

			if got := email.String(); got != tt.want {
				t.Fatalf("email = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEmailAddressIsZero(t *testing.T) {
	if !((billingvalues.EmailAddress{}).IsZero()) {
		t.Fatal("expected zero email address to report IsZero")
	}

	email, err := billingvalues.NewEmailAddress("billing@example.com")
	if err != nil {
		t.Fatalf("NewEmailAddress() error = %v", err)
	}

	if email.IsZero() {
		t.Fatal("expected valid email address to be non-zero")
	}
}

func TestNewPhoneNumber(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "valid", input: "+1 809 555 1234", want: "+1 809 555 1234"},
		{name: "rejects blank", input: "   ", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			phone, err := billingvalues.NewPhoneNumber(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				if !errors.Is(err, billingvalues.ErrPhoneNumberRequired) {
					t.Fatalf("err = %v, want %v", err, billingvalues.ErrPhoneNumberRequired)
				}
				return
			}

			if got := phone.String(); got != tt.want {
				t.Fatalf("phone = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPhoneNumberIsZero(t *testing.T) {
	if !((billingvalues.PhoneNumber{}).IsZero()) {
		t.Fatal("expected zero phone number to report IsZero")
	}

	phone, err := billingvalues.NewPhoneNumber("+1 809 555 1234")
	if err != nil {
		t.Fatalf("NewPhoneNumber() error = %v", err)
	}

	if phone.IsZero() {
		t.Fatal("expected valid phone number to be non-zero")
	}
}

func TestNewTaxIdentifier(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "valid", input: "RNC-123456789", want: "RNC-123456789"},
		{name: "rejects blank", input: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			taxID, err := billingvalues.NewTaxIdentifier(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				if !errors.Is(err, billingvalues.ErrTaxIdentifierRequired) {
					t.Fatalf("err = %v, want %v", err, billingvalues.ErrTaxIdentifierRequired)
				}
				return
			}

			if got := taxID.String(); got != tt.want {
				t.Fatalf("tax id = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTaxIdentifierIsZero(t *testing.T) {
	if !((billingvalues.TaxIdentifier{}).IsZero()) {
		t.Fatal("expected zero tax identifier to report IsZero")
	}

	taxID, err := billingvalues.NewTaxIdentifier("RNC-123456789")
	if err != nil {
		t.Fatalf("NewTaxIdentifier() error = %v", err)
	}

	if taxID.IsZero() {
		t.Fatal("expected valid tax identifier to be non-zero")
	}
}
