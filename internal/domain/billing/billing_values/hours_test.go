package billingvalues_test

import (
	"errors"
	"testing"

	"github.com/Carlos0934/billar/internal/domain/billing/billing_values"
)

func TestNewHours(t *testing.T) {
	tests := []struct {
		name    string
		input   int64
		want    string
		wantErr bool
	}{
		{name: "zero", input: 0, want: "0.0000"},
		{name: "scaled precision", input: 156250, want: "15.6250"},
		{name: "rejects negative", input: -1, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hours, err := billingvalues.NewHours(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				if !errors.Is(err, billingvalues.ErrHoursNegative) {
					t.Fatalf("err = %v, want %v", err, billingvalues.ErrHoursNegative)
				}
				return
			}

			if got := hours.Value(); got != tt.input {
				t.Fatalf("value = %d, want %d", got, tt.input)
			}

			if got := hours.String(); got != tt.want {
				t.Fatalf("string = %q, want %q", got, tt.want)
			}
		})
	}
}
