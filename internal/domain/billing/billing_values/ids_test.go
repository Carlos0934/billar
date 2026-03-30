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

func TestNewIssuerProfileID(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "valid", input: "issuer-123", want: "issuer-123"},
		{name: "rejects blank", input: "   ", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := billingvalues.NewIssuerProfileID(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				if !errors.Is(err, billingvalues.ErrIssuerProfileIDRequired) {
					t.Fatalf("err = %v, want %v", err, billingvalues.ErrIssuerProfileIDRequired)
				}
				return
			}

			if got := id.String(); got != tt.want {
				t.Fatalf("id = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewTimeEntryID(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "valid", input: "time-123", want: "time-123"},
		{name: "rejects blank", input: "   ", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := billingvalues.NewTimeEntryID(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				if !errors.Is(err, billingvalues.ErrTimeEntryIDRequired) {
					t.Fatalf("err = %v, want %v", err, billingvalues.ErrTimeEntryIDRequired)
				}
				return
			}

			if got := id.String(); got != tt.want {
				t.Fatalf("id = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewInvoiceID(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "valid", input: "inv-123", want: "inv-123"},
		{name: "rejects blank", input: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := billingvalues.NewInvoiceID(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				if !errors.Is(err, billingvalues.ErrInvoiceIDRequired) {
					t.Fatalf("err = %v, want %v", err, billingvalues.ErrInvoiceIDRequired)
				}
				return
			}

			if got := id.String(); got != tt.want {
				t.Fatalf("id = %q, want %q", got, tt.want)
			}
		})
	}
}
