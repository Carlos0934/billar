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
