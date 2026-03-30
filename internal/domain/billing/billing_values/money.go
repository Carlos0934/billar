package billingvalues

import (
	"fmt"
	"strings"
)

const moneyScale = int64(10_000)

type CurrencyCode struct {
	value string
}

func NewCurrencyCode(value string) (CurrencyCode, error) {
	trimmed := strings.ToUpper(strings.TrimSpace(value))
	if len(trimmed) != 3 {
		return CurrencyCode{}, ErrCurrencyCodeLength
	}

	for _, r := range trimmed {
		if r < 'A' || r > 'Z' {
			return CurrencyCode{}, ErrCurrencyCodeLetters
		}
	}

	return CurrencyCode{value: trimmed}, nil
}

func MustCurrencyCode(value string) CurrencyCode {
	code, err := NewCurrencyCode(value)
	if err != nil {
		panic(err)
	}

	return code
}

func DefaultCurrencyCode() CurrencyCode {
	return MustCurrencyCode("USD")
}

func (code CurrencyCode) String() string {
	return code.value
}

func (code CurrencyCode) IsZero() bool {
	return code.value == ""
}

type Money struct {
	amount   int64
	currency CurrencyCode
}

func NewMoney(amount int64, currency CurrencyCode) (Money, error) {
	if currency.IsZero() {
		return Money{}, ErrCurrencyRequired
	}

	return Money{amount: amount, currency: currency}, nil
}

func (money Money) Amount() int64 {
	return money.amount
}

func (money Money) Currency() CurrencyCode {
	return money.currency
}

func (money Money) String() string {
	absolute := money.amount
	if absolute < 0 {
		absolute = -absolute
	}

	whole := absolute / moneyScale
	frac := absolute % moneyScale
	prefix := ""
	if money.amount < 0 {
		prefix = "-"
	}

	return fmt.Sprintf("%s%d.%04d %s", prefix, whole, frac, money.currency.String())
}
