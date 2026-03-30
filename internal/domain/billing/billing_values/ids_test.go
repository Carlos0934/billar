package billingvalues_test

import (
	"errors"
	"testing"

	"github.com/Carlos0934/billar/internal/domain/billing/billing_values"
)

func TestNewCustomerID(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "valid", input: "cust-123", want: "cust-123"},
		{name: "rejects blank", input: "   ", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := billingvalues.NewCustomerID(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				if !errors.Is(err, billingvalues.ErrCustomerIDRequired) {
					t.Fatalf("err = %v, want %v", err, billingvalues.ErrCustomerIDRequired)
				}
				return
			}

			if got := id.String(); got != tt.want {
				t.Fatalf("id = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewServiceAgreementID(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "valid", input: "agr-123", want: "agr-123"},
		{name: "rejects blank", input: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := billingvalues.NewServiceAgreementID(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				if !errors.Is(err, billingvalues.ErrServiceAgreementIDRequired) {
					t.Fatalf("err = %v, want %v", err, billingvalues.ErrServiceAgreementIDRequired)
				}
				return
			}

			if got := id.String(); got != tt.want {
				t.Fatalf("id = %q, want %q", got, tt.want)
			}
		})
	}
}
