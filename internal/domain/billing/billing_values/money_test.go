package billingvalues_test

import (
	"errors"
	"testing"

	"github.com/Carlos0934/billar/internal/domain/billing/billing_values"
)

func TestNewCurrencyCode(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "uppercase", input: "USD", want: "USD"},
		{name: "normalizes lowercase", input: "usd", want: "USD"},
		{name: "rejects short", input: "US", wantErr: true},
		{name: "rejects non letters", input: "U1D", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, err := billingvalues.NewCurrencyCode(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				switch tt.name {
				case "rejects short":
					if !errors.Is(err, billingvalues.ErrCurrencyCodeLength) {
						t.Fatalf("err = %v, want %v", err, billingvalues.ErrCurrencyCodeLength)
					}
				case "rejects non letters":
					if !errors.Is(err, billingvalues.ErrCurrencyCodeLetters) {
						t.Fatalf("err = %v, want %v", err, billingvalues.ErrCurrencyCodeLetters)
					}
				}
				return
			}

			if got := code.String(); got != tt.want {
				t.Fatalf("currency = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewMoneyPreservesScaledPrecision(t *testing.T) {
	currency, err := billingvalues.NewCurrencyCode("USD")
	if err != nil {
		t.Fatalf("NewCurrencyCode() error = %v", err)
	}

	money, err := billingvalues.NewMoney(156250, currency)
	if err != nil {
		t.Fatalf("NewMoney() error = %v", err)
	}

	if got := money.Amount(); got != 156250 {
		t.Fatalf("amount = %d, want %d", got, 156250)
	}

	if got := money.Currency().String(); got != "USD" {
		t.Fatalf("currency = %q, want %q", got, "USD")
	}

	if got := money.String(); got != "15.6250 USD" {
		t.Fatalf("string = %q, want %q", got, "15.6250 USD")
	}
}

func TestNewMoneyRejectsMissingCurrency(t *testing.T) {
	if _, err := billingvalues.NewMoney(1, billingvalues.CurrencyCode{}); err == nil {
		t.Fatal("expected error for missing currency")
	} else if !errors.Is(err, billingvalues.ErrCurrencyRequired) {
		t.Fatalf("err = %v, want %v", err, billingvalues.ErrCurrencyRequired)
	}
}
