package billingvalues

import (
	"strings"
)

type CountryCode struct {
	value string
}

func NewCountryCode(value string) (CountryCode, error) {
	trimmed := strings.TrimSpace(strings.ToUpper(value))
	if len(trimmed) != 2 {
		return CountryCode{}, ErrCountryCodeLength
	}

	for _, r := range trimmed {
		if r < 'A' || r > 'Z' {
			return CountryCode{}, ErrCountryCodeLetters
		}
	}

	return CountryCode{value: trimmed}, nil
}

func (code CountryCode) String() string {
	return code.value
}

func (code CountryCode) IsZero() bool {
	return code.value == ""
}
