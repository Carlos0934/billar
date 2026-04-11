package core

import (
	"errors"
	"testing"
)

func TestNewMoney(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		amount   int64
		currency string
		wantErr  bool
	}{
		{name: "valid", amount: 15000, currency: "USD"},
		{name: "zero amount rejected", amount: 0, currency: "USD", wantErr: true},
		{name: "blank currency rejected", amount: 15000, currency: "", wantErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := NewMoney(tt.amount, tt.currency)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("NewMoney() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("NewMoney() error = %v, want nil", err)
			}
			if got.Amount != tt.amount || got.Currency != tt.currency {
				t.Fatalf("NewMoney() = %#v, want amount %d currency %q", got, tt.amount, tt.currency)
			}
		})
	}
}

func TestMoneyAdd(t *testing.T) {
	t.Parallel()

	base, err := NewMoney(10000, "USD")
	if err != nil {
		t.Fatalf("NewMoney(base): %v", err)
	}

	t.Run("same currency", func(t *testing.T) {
		t.Parallel()

		other, err := NewMoney(5000, "USD")
		if err != nil {
			t.Fatalf("NewMoney(other): %v", err)
		}

		got, err := base.Add(other)
		if err != nil {
			t.Fatalf("Add() error = %v", err)
		}
		if got.Amount != 15000 {
			t.Fatalf("Add() amount = %d, want 15000", got.Amount)
		}
	})

	t.Run("currency mismatch", func(t *testing.T) {
		t.Parallel()

		other, err := NewMoney(5000, "EUR")
		if err != nil {
			t.Fatalf("NewMoney(other): %v", err)
		}

		_, err = base.Add(other)
		if err == nil {
			t.Fatal("Add() error = nil, want currency mismatch")
		}
		if !errors.Is(err, ErrMoneyCurrencyMismatch) {
			t.Fatalf("Add() error = %v, want ErrMoneyCurrencyMismatch", err)
		}
	})
}

func TestMoneyMultiply(t *testing.T) {
	t.Parallel()

	base, err := NewMoney(1250, "USD")
	if err != nil {
		t.Fatalf("NewMoney(base): %v", err)
	}

	got, err := base.Multiply(3)
	if err != nil {
		t.Fatalf("Multiply() error = %v", err)
	}
	if got.Amount != 3750 {
		t.Fatalf("Multiply() amount = %d, want 3750", got.Amount)
	}
	if !got.Equal(Money{Amount: 3750, Currency: "USD"}) {
		t.Fatalf("Equal() = false, want true")
	}
}

func TestMoneyEqualAndIsPositive(t *testing.T) {
	t.Parallel()

	positive, err := NewMoney(1, "USD")
	if err != nil {
		t.Fatalf("NewMoney(positive): %v", err)
	}
	if !positive.IsPositive() {
		t.Fatal("IsPositive() = false, want true")
	}

	if !positive.Equal(Money{Amount: 1, Currency: "USD"}) {
		t.Fatal("Equal() = false, want true")
	}
	if positive.Equal(Money{Amount: 1, Currency: "EUR"}) {
		t.Fatal("Equal() = true, want false")
	}
	if (Money{}).IsPositive() {
		t.Fatal("zero-value Money IsPositive() = true, want false")
	}
}

func TestMoneyCompare(t *testing.T) {
	t.Parallel()

	low, err := NewMoney(1000, "USD")
	if err != nil {
		t.Fatalf("NewMoney(low): %v", err)
	}
	high, err := NewMoney(2500, "USD")
	if err != nil {
		t.Fatalf("NewMoney(high): %v", err)
	}
	equal, err := NewMoney(1000, "USD")
	if err != nil {
		t.Fatalf("NewMoney(equal): %v", err)
	}
	different, err := NewMoney(1000, "EUR")
	if err != nil {
		t.Fatalf("NewMoney(different): %v", err)
	}

	tests := []struct {
		name    string
		left    Money
		right   Money
		want    int
		wantErr bool
	}{
		{name: "lower amount", left: low, right: high, want: -1},
		{name: "equal amount", left: low, right: equal, want: 0},
		{name: "higher amount", left: high, right: low, want: 1},
		{name: "currency mismatch", left: low, right: different, wantErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tt.left.Compare(tt.right)
			if tt.wantErr {
				if err == nil {
					t.Fatal("Compare() error = nil, want currency mismatch")
				}
				if !errors.Is(err, ErrMoneyCurrencyMismatch) {
					t.Fatalf("Compare() error = %v, want ErrMoneyCurrencyMismatch", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Compare() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("Compare() = %d, want %d", got, tt.want)
			}
		})
	}
}
