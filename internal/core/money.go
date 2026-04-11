package core

import (
	"errors"
	"fmt"
	"strings"
)

var ErrMoneyCurrencyMismatch = errors.New("money currency mismatch")

type Money struct {
	Amount   int64
	Currency string
}

func NewMoney(amount int64, currency string) (Money, error) {
	if amount <= 0 {
		return Money{}, errors.New("money amount must be positive")
	}
	if strings.TrimSpace(currency) == "" {
		return Money{}, errors.New("money currency is required")
	}
	return Money{Amount: amount, Currency: strings.TrimSpace(currency)}, nil
}

func (m Money) Add(other Money) (Money, error) {
	if m.Currency != other.Currency {
		return Money{}, fmt.Errorf("add money: %w", ErrMoneyCurrencyMismatch)
	}
	return Money{Amount: m.Amount + other.Amount, Currency: m.Currency}, nil
}

func (m Money) Compare(other Money) (int, error) {
	if m.Currency != other.Currency {
		return 0, fmt.Errorf("compare money: %w", ErrMoneyCurrencyMismatch)
	}
	switch {
	case m.Amount < other.Amount:
		return -1, nil
	case m.Amount > other.Amount:
		return 1, nil
	default:
		return 0, nil
	}
}

func (m Money) Multiply(factor int64) (Money, error) {
	if factor <= 0 {
		return Money{}, errors.New("money multiply factor must be positive")
	}
	if m.Currency == "" {
		return Money{}, errors.New("money currency is required")
	}
	return Money{Amount: m.Amount * factor, Currency: m.Currency}, nil
}

func (m Money) Equal(other Money) bool {
	return m.Amount == other.Amount && m.Currency == other.Currency
}

func (m Money) IsPositive() bool {
	return m.Amount > 0 && strings.TrimSpace(m.Currency) != ""
}
