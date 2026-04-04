package app

import (
	"time"

	"github.com/Carlos0934/billar/internal/core"
)

type AddressDTO struct {
	Street     string `json:"street" toon:"street"`
	City       string `json:"city" toon:"city"`
	State      string `json:"state" toon:"state"`
	PostalCode string `json:"postal_code" toon:"postal_code"`
	Country    string `json:"country" toon:"country"`
}

type CustomerDTO struct {
	ID              string     `json:"id" toon:"id"`
	Type            string     `json:"type" toon:"type"`
	LegalName       string     `json:"legal_name" toon:"legal_name"`
	TradeName       string     `json:"trade_name" toon:"trade_name"`
	TaxID           string     `json:"tax_id" toon:"tax_id"`
	Email           string     `json:"email" toon:"email"`
	Phone           string     `json:"phone" toon:"phone"`
	Website         string     `json:"website" toon:"website"`
	BillingAddress  AddressDTO `json:"billing_address" toon:"billing_address"`
	Status          string     `json:"status" toon:"status"`
	DefaultCurrency string     `json:"default_currency" toon:"default_currency"`
	Notes           string     `json:"notes" toon:"notes"`
	CreatedAt       string     `json:"created_at" toon:"created_at"`
	UpdatedAt       string     `json:"updated_at" toon:"updated_at"`
}

func customerToDTO(customer core.Customer) CustomerDTO {
	return CustomerDTO{
		ID:              customer.ID,
		Type:            string(customer.Type),
		LegalName:       customer.LegalName,
		TradeName:       customer.TradeName,
		TaxID:           customer.TaxID,
		Email:           customer.Email,
		Phone:           customer.Phone,
		Website:         customer.Website,
		BillingAddress:  addressToDTO(customer.BillingAddress),
		Status:          string(customer.Status),
		DefaultCurrency: customer.DefaultCurrency,
		Notes:           customer.Notes,
		CreatedAt:       formatCustomerTime(customer.CreatedAt),
		UpdatedAt:       formatCustomerTime(customer.UpdatedAt),
	}
}

func addressToDTO(address core.Address) AddressDTO {
	return AddressDTO{
		Street:     address.Street,
		City:       address.City,
		State:      address.State,
		PostalCode: address.PostalCode,
		Country:    address.Country,
	}
}

func formatCustomerTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

// CreateCustomerCommand represents the input for creating a customer.
type CreateCustomerCommand struct {
	Type            string     `json:"type"`
	LegalName       string     `json:"legal_name"`
	TradeName       string     `json:"trade_name"`
	TaxID           string     `json:"tax_id"`
	Email           string     `json:"email"`
	Phone           string     `json:"phone"`
	Website         string     `json:"website"`
	BillingAddress  AddressDTO `json:"billing_address"`
	Notes           string     `json:"notes"`
	DefaultCurrency string     `json:"default_currency"`
}

// PatchCustomerCommand represents a partial update to a customer.
// Pointer fields distinguish between "not sent" (nil) and "clear field" (empty string).
type PatchCustomerCommand struct {
	Type            *string     `json:"type,omitempty"`
	LegalName       *string     `json:"legal_name,omitempty"`
	TradeName       *string     `json:"trade_name,omitempty"`
	TaxID           *string     `json:"tax_id,omitempty"`
	Email           *string     `json:"email,omitempty"`
	Phone           *string     `json:"phone,omitempty"`
	Website         *string     `json:"website,omitempty"`
	BillingAddress  *AddressDTO `json:"billing_address,omitempty"`
	Notes           *string     `json:"notes,omitempty"`
	DefaultCurrency *string     `json:"default_currency,omitempty"`
}
