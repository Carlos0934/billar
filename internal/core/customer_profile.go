package core

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	CustomerProfileStatusActive   CustomerProfileStatus = "active"
	CustomerProfileStatusInactive CustomerProfileStatus = "inactive"

	customerProfileIDPrefix = "cus_"
	customerProfileIDBytes  = 16
)

type CustomerProfileStatus string

func (s CustomerProfileStatus) IsValid() bool {
	switch s {
	case CustomerProfileStatusActive, CustomerProfileStatusInactive:
		return true
	default:
		return false
	}
}

type CustomerProfile struct {
	ID              string
	LegalEntityID   string
	Status          CustomerProfileStatus
	DefaultCurrency string
	Notes           string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type CustomerProfileParams struct {
	LegalEntityID   string
	DefaultCurrency string
	Notes           string
}

type CustomerProfilePatch struct {
	Status          *CustomerProfileStatus
	DefaultCurrency *string
	Notes           *string
}

func NewCustomerProfile(params CustomerProfileParams) (CustomerProfile, error) {
	if strings.TrimSpace(params.LegalEntityID) == "" {
		return CustomerProfile{}, errors.New("customer profile legal entity id is required")
	}
	if strings.TrimSpace(params.DefaultCurrency) == "" {
		return CustomerProfile{}, errors.New("customer profile default currency is required")
	}

	now := time.Now().UTC()
	profile := CustomerProfile{
		ID:              generateCustomerProfileID(),
		LegalEntityID:   strings.TrimSpace(params.LegalEntityID),
		Status:          CustomerProfileStatusActive,
		DefaultCurrency: strings.TrimSpace(params.DefaultCurrency),
		Notes:           params.Notes,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if profile.ID == "" {
		return CustomerProfile{}, errors.New("failed to generate customer profile id")
	}

	return profile, nil
}

func (p *CustomerProfile) ApplyPatch(patch CustomerProfilePatch) {
	changed := false

	if patch.Status != nil {
		p.Status = *patch.Status
		changed = true
	}
	if patch.DefaultCurrency != nil {
		p.DefaultCurrency = *patch.DefaultCurrency
		changed = true
	}
	if patch.Notes != nil {
		p.Notes = *patch.Notes
		changed = true
	}

	if changed {
		p.UpdatedAt = time.Now().UTC()
	}
}

func (p CustomerProfile) Validate() error {
	if strings.TrimSpace(p.LegalEntityID) == "" {
		return errors.New("customer profile legal entity id is required")
	}
	if strings.TrimSpace(p.DefaultCurrency) == "" {
		return errors.New("customer profile default currency is required")
	}
	if !p.Status.IsValid() {
		return fmt.Errorf("invalid customer profile status: %s", p.Status)
	}
	return nil
}

func (p CustomerProfile) ValidateDelete() error {
	return nil
}

func (p *CustomerProfile) CanReceiveInvoices() bool {
	return p != nil && p.Status == CustomerProfileStatusActive
}

func generateCustomerProfileID() string {
	return generatePrefixedID(customerProfileIDPrefix, customerProfileIDBytes)
}
