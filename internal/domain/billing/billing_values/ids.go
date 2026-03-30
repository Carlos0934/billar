package billingvalues

import (
	"strings"
)

type CustomerID struct {
	value string
}

func NewCustomerID(value string) (CustomerID, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return CustomerID{}, ErrCustomerIDRequired
	}

	return CustomerID{value: trimmed}, nil
}

func (id CustomerID) String() string {
	return id.value
}

func (id CustomerID) IsZero() bool {
	return id.value == ""
}

type ServiceAgreementID struct {
	value string
}

func NewServiceAgreementID(value string) (ServiceAgreementID, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ServiceAgreementID{}, ErrServiceAgreementIDRequired
	}

	return ServiceAgreementID{value: trimmed}, nil
}

func (id ServiceAgreementID) String() string {
	return id.value
}

func (id ServiceAgreementID) IsZero() bool {
	return id.value == ""
}
