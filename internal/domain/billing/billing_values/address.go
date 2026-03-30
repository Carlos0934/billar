package billingvalues

import (
	"strings"
)

type AddressInput struct {
	Line1      string
	Line2      string
	City       string
	State      string
	PostalCode string
	Country    CountryCode
}

type Address struct {
	line1      string
	line2      string
	city       string
	state      string
	postalCode string
	country    CountryCode
}

func NewAddress(input AddressInput) (Address, error) {
	if input.Country.IsZero() {
		return Address{}, ErrAddressCountryRequired
	}

	address := Address{
		line1:      strings.TrimSpace(input.Line1),
		line2:      strings.TrimSpace(input.Line2),
		city:       strings.TrimSpace(input.City),
		state:      strings.TrimSpace(input.State),
		postalCode: strings.TrimSpace(input.PostalCode),
		country:    input.Country,
	}

	if address.line1 == "" {
		return Address{}, ErrAddressLine1Required
	}
	if address.city == "" {
		return Address{}, ErrAddressCityRequired
	}
	return address, nil
}

func (address Address) Line1() string {
	return address.line1
}

func (address Address) Line2() string {
	return address.line2
}

func (address Address) City() string {
	return address.city
}

func (address Address) State() string {
	return address.state
}

func (address Address) PostalCode() string {
	return address.postalCode
}

func (address Address) Country() CountryCode {
	return address.country
}
