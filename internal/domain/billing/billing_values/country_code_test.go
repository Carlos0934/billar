package billingvalues_test

import (
	"errors"
	"testing"

	"github.com/Carlos0934/billar/internal/domain/billing/billing_values"
)

func TestNewCountryCode(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "valid uppercase", input: "DO", want: "DO"},
		{name: "normalizes lowercase", input: "us", want: "US"},
		{name: "rejects empty", input: "", wantErr: true},
		{name: "rejects wrong length", input: "DOM", wantErr: true},
		{name: "rejects non letters", input: "D1", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, err := billingvalues.NewCountryCode(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				switch tt.name {
				case "rejects empty", "rejects wrong length":
					if !errors.Is(err, billingvalues.ErrCountryCodeLength) {
						t.Fatalf("err = %v, want %v", err, billingvalues.ErrCountryCodeLength)
					}
				case "rejects non letters":
					if !errors.Is(err, billingvalues.ErrCountryCodeLetters) {
						t.Fatalf("err = %v, want %v", err, billingvalues.ErrCountryCodeLetters)
					}
				}
				return
			}

			if got := code.String(); got != tt.want {
				t.Fatalf("code = %q, want %q", got, tt.want)
			}
		})
	}
}
