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
