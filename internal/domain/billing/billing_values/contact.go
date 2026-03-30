package billingvalues

import (
	"strings"
)

type EmailAddress struct {
	value string
}

func NewEmailAddress(value string) (EmailAddress, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || strings.Count(trimmed, "@") != 1 {
		return EmailAddress{}, ErrEmailAddressInvalid
	}

	parts := strings.Split(trimmed, "@")
	if parts[0] == "" || parts[1] == "" || !strings.Contains(parts[1], ".") {
		return EmailAddress{}, ErrEmailAddressInvalid
	}

	return EmailAddress{value: trimmed}, nil
}

func (email EmailAddress) String() string {
	return email.value
}

func (email EmailAddress) IsZero() bool {
	return email.value == ""
}

type PhoneNumber struct {
	value string
}

func NewPhoneNumber(value string) (PhoneNumber, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return PhoneNumber{}, ErrPhoneNumberRequired
	}

	return PhoneNumber{value: trimmed}, nil
}

func (phone PhoneNumber) String() string {
	return phone.value
}

func (phone PhoneNumber) IsZero() bool {
	return phone.value == ""
}

type TaxIdentifier struct {
	value string
}

func NewTaxIdentifier(value string) (TaxIdentifier, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return TaxIdentifier{}, ErrTaxIdentifierRequired
	}

	return TaxIdentifier{value: trimmed}, nil
}

func (taxID TaxIdentifier) String() string {
	return taxID.value
}

func (taxID TaxIdentifier) IsZero() bool {
	return taxID.value == ""
}
