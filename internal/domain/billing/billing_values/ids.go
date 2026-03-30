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

type IssuerProfileID struct {
	value string
}

func NewIssuerProfileID(value string) (IssuerProfileID, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return IssuerProfileID{}, ErrIssuerProfileIDRequired
	}

	return IssuerProfileID{value: trimmed}, nil
}

func (id IssuerProfileID) String() string {
	return id.value
}

func (id IssuerProfileID) IsZero() bool {
	return id.value == ""
}

type TimeEntryID struct {
	value string
}

func NewTimeEntryID(value string) (TimeEntryID, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return TimeEntryID{}, ErrTimeEntryIDRequired
	}

	return TimeEntryID{value: trimmed}, nil
}

func (id TimeEntryID) String() string {
	return id.value
}

func (id TimeEntryID) IsZero() bool {
	return id.value == ""
}

type InvoiceID struct {
	value string
}

func NewInvoiceID(value string) (InvoiceID, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return InvoiceID{}, ErrInvoiceIDRequired
	}

	return InvoiceID{value: trimmed}, nil
}

func (id InvoiceID) String() string {
	return id.value
}

func (id InvoiceID) IsZero() bool {
	return id.value == ""
}
