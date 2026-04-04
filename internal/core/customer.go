package core

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	CustomerTypeIndividual CustomerType = "individual"
	CustomerTypeCompany    CustomerType = "company"

	CustomerStatusActive   CustomerStatus = "active"
	CustomerStatusInactive CustomerStatus = "inactive"

	defaultCustomerCurrency = "USD"
	customerIDPrefix        = "cus_"
	customerIDBytes         = 16
	customerIDHexChars      = 32
)

type CustomerType string

func (t CustomerType) IsValid() bool {
	switch t {
	case CustomerTypeIndividual, CustomerTypeCompany:
		return true
	default:
		return false
	}
}

type CustomerStatus string

func (s CustomerStatus) IsValid() bool {
	switch s {
	case CustomerStatusActive, CustomerStatusInactive:
		return true
	default:
		return false
	}
}

type Address struct {
	Street     string
	City       string
	State      string
	PostalCode string
	Country    string
}

type Customer struct {
	ID              string
	Type            CustomerType
	LegalName       string
	TradeName       string
	TaxID           string
	Email           string
	Phone           string
	Website         string
	BillingAddress  Address
	Status          CustomerStatus
	DefaultCurrency string
	Notes           string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type CustomerParams struct {
	Type           CustomerType
	LegalName      string
	TradeName      string
	TaxID          string
	Email          string
	Phone          string
	Website        string
	BillingAddress Address
	Notes          string
}

// CustomerPatch defines optional updates for a customer.
// Nil pointers mean the field should be skipped (not updated).
// Non-nil pointers (even if empty string) mean the field should be applied.
type CustomerPatch struct {
	Type            *CustomerType
	LegalName       *string
	TradeName       *string
	TaxID           *string
	Email           *string
	Phone           *string
	Website         *string
	BillingAddress  *Address
	Notes           *string
	DefaultCurrency *string
}

func NewCustomer(params CustomerParams) (Customer, error) {
	if strings.TrimSpace(params.LegalName) == "" {
		return Customer{}, errors.New("customer legal name is required")
	}
	if !params.Type.IsValid() {
		return Customer{}, fmt.Errorf("invalid customer type: %s", params.Type)
	}

	now := time.Now().UTC()
	customer := Customer{
		ID:              generateCustomerID(),
		Type:            params.Type,
		LegalName:       strings.TrimSpace(params.LegalName),
		TradeName:       params.TradeName,
		TaxID:           params.TaxID,
		Email:           params.Email,
		Phone:           params.Phone,
		Website:         params.Website,
		BillingAddress:  params.BillingAddress,
		Status:          CustomerStatusActive,
		DefaultCurrency: defaultCustomerCurrency,
		Notes:           params.Notes,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if customer.ID == "" {
		return Customer{}, errors.New("failed to generate customer id")
	}

	return customer, nil
}

func (c *Customer) CanReceiveInvoices() bool {
	return c != nil && c.Status == CustomerStatusActive
}

// ApplyPatch applies the given patch to the customer.
// Nil pointers in the patch mean the field should be skipped.
// Non-nil pointers (even if empty string) mean the field should be applied.
// The method updates the UpdatedAt timestamp if any field was changed.
func (c *Customer) ApplyPatch(patch CustomerPatch) {
	changed := false

	if patch.Type != nil {
		c.Type = *patch.Type
		changed = true
	}
	if patch.LegalName != nil {
		c.LegalName = *patch.LegalName
		changed = true
	}
	if patch.TradeName != nil {
		c.TradeName = *patch.TradeName
		changed = true
	}
	if patch.TaxID != nil {
		c.TaxID = *patch.TaxID
		changed = true
	}
	if patch.Email != nil {
		c.Email = *patch.Email
		changed = true
	}
	if patch.Phone != nil {
		c.Phone = *patch.Phone
		changed = true
	}
	if patch.Website != nil {
		c.Website = *patch.Website
		changed = true
	}
	if patch.BillingAddress != nil {
		c.BillingAddress = *patch.BillingAddress
		changed = true
	}
	if patch.Notes != nil {
		c.Notes = *patch.Notes
		changed = true
	}
	if patch.DefaultCurrency != nil {
		c.DefaultCurrency = *patch.DefaultCurrency
		changed = true
	}

	if changed {
		c.UpdatedAt = time.Now().UTC()
	}
}

// Validate checks whether the customer is in a valid state.
// This is used to re-validate after patches to ensure invariants are maintained.
func (c Customer) Validate() error {
	if strings.TrimSpace(c.LegalName) == "" {
		return errors.New("customer legal name is required")
	}
	if !c.Type.IsValid() {
		return fmt.Errorf("invalid customer type: %s", c.Type)
	}
	return nil
}

// ValidateDelete checks whether the customer can be deleted.
// This is a seam for future relationship-protection logic (e.g., blocking
// deletion if the customer has invoices). Currently always returns nil.
func (c Customer) ValidateDelete() error {
	return nil
}

func generateCustomerID() string {
	buf := make([]byte, customerIDBytes)
	if _, err := rand.Read(buf); err != nil {
		return ""
	}

	encoded := hex.EncodeToString(buf)
	if len(encoded) != customerIDHexChars {
		return ""
	}

	return customerIDPrefix + encoded
}
