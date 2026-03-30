package billingvalues_test

import (
	"testing"

	"github.com/Carlos0934/billar/internal/domain/billing/billing_values"
)

func TestNewAddress(t *testing.T) {
	do, err := billingvalues.NewCountryCode("do")
	if err != nil {
		t.Fatalf("NewCountryCode() error = %v", err)
	}

	tests := []struct {
		name    string
		input   billingvalues.AddressInput
		wantErr bool
	}{
		{
			name: "valid",
			input: billingvalues.AddressInput{
				Line1:      "123 Main Street",
				City:       "Santo Domingo",
				PostalCode: "10101",
				Country:    do,
			},
		},
		{
			name: "rejects missing line1",
			input: billingvalues.AddressInput{
				City:    "Santo Domingo",
				Country: do,
			},
			wantErr: true,
		},
		{
			name: "rejects missing country",
			input: billingvalues.AddressInput{
				Line1: "123 Main Street",
				City:  "Santo Domingo",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			address, err := billingvalues.NewAddress(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}

			if got := address.Line1(); got != tt.input.Line1 {
				t.Fatalf("line1 = %q, want %q", got, tt.input.Line1)
			}

			if got := address.City(); got != tt.input.City {
				t.Fatalf("city = %q, want %q", got, tt.input.City)
			}

			if got := address.Country(); got != tt.input.Country {
				t.Fatalf("country = %q, want %q", got.String(), tt.input.Country.String())
			}
		})
	}
}
