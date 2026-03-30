package billingvalues_test

import (
	"errors"
	"testing"

	"github.com/Carlos0934/billar/internal/domain/billing/billing_values"
)

func TestNewHours(t *testing.T) {
	tests := []struct {
		name     string
		input    int64
		want     string
		positive bool
		wantErr  bool
	}{
		{name: "scaled precision", input: 12500, want: "1.2500", positive: true},
		{name: "larger scaled precision", input: 156250, want: "15.6250", positive: true},
		{name: "rejects zero", input: 0, wantErr: true},
		{name: "rejects negative", input: -1, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hours, err := billingvalues.NewHours(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				if !errors.Is(err, billingvalues.ErrHoursMustBePositive) {
					t.Fatalf("err = %v, want %v", err, billingvalues.ErrHoursMustBePositive)
				}
				return
			}

			if got := hours.Value(); got != tt.input {
				t.Fatalf("value = %d, want %d", got, tt.input)
			}

			if got := hours.String(); got != tt.want {
				t.Fatalf("string = %q, want %q", got, tt.want)
			}

			if got := hours.IsPositive(); got != tt.positive {
				t.Fatalf("IsPositive() = %v, want %v", got, tt.positive)
			}
		})
	}
}
