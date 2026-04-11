package core

import (
	"errors"
	"strings"
	"time"
)

const (
	BillingModeHourly BillingMode = "hourly"

	serviceAgreementIDPrefix = "sa_"
	serviceAgreementIDBytes  = 16
)

// BillingMode represents the billing strategy for a service agreement.
type BillingMode string

// IsValid reports whether the billing mode is a supported value.
func (m BillingMode) IsValid() bool {
	switch m {
	case BillingModeHourly:
		return true
	default:
		return false
	}
}

// ServiceAgreement defines the billing rules for a customer profile.
// HourlyRate is stored as the smallest currency unit (e.g., cents) — no floats.
type ServiceAgreement struct {
	ID                string
	CustomerProfileID string
	Name              string
	Description       string
	BillingMode       BillingMode
	HourlyRate        int64
	Currency          string
	Active            bool
	ValidFrom         *time.Time
	ValidUntil        *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// ServiceAgreementParams holds the inputs for creating a new ServiceAgreement.
type ServiceAgreementParams struct {
	CustomerProfileID string
	Name              string
	Description       string
	BillingMode       BillingMode
	HourlyRate        int64
	Currency          string
	ValidFrom         *time.Time
	ValidUntil        *time.Time
}

// NewServiceAgreement constructs a validated ServiceAgreement from params.
// It enforces all domain invariants at construction time.
func NewServiceAgreement(params ServiceAgreementParams) (ServiceAgreement, error) {
	if strings.TrimSpace(params.CustomerProfileID) == "" {
		return ServiceAgreement{}, errors.New("service agreement customer profile id is required")
	}
	if strings.TrimSpace(params.Name) == "" {
		return ServiceAgreement{}, errors.New("service agreement name is required")
	}
	if strings.TrimSpace(params.Currency) == "" {
		return ServiceAgreement{}, errors.New("service agreement currency is required")
	}
	if params.HourlyRate <= 0 {
		return ServiceAgreement{}, errors.New("service agreement hourly rate must be positive")
	}
	if !params.BillingMode.IsValid() {
		return ServiceAgreement{}, errors.New("service agreement billing mode is unsupported")
	}

	now := time.Now().UTC()
	sa := ServiceAgreement{
		ID:                generateServiceAgreementID(),
		CustomerProfileID: strings.TrimSpace(params.CustomerProfileID),
		Name:              strings.TrimSpace(params.Name),
		Description:       params.Description,
		BillingMode:       params.BillingMode,
		HourlyRate:        params.HourlyRate,
		Currency:          strings.TrimSpace(params.Currency),
		Active:            true,
		ValidFrom:         params.ValidFrom,
		ValidUntil:        params.ValidUntil,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if sa.ID == "" {
		return ServiceAgreement{}, errors.New("failed to generate service agreement id")
	}

	return sa, nil
}

// UpdateRate sets a new hourly rate. The rate must be strictly positive.
// UpdatedAt is refreshed on success.
func (sa *ServiceAgreement) UpdateRate(rate int64) error {
	if rate <= 0 {
		return errors.New("service agreement hourly rate must be positive")
	}
	sa.HourlyRate = rate
	sa.UpdatedAt = time.Now().UTC()
	return nil
}

// Activate sets the agreement as active and refreshes UpdatedAt.
func (sa *ServiceAgreement) Activate() {
	sa.Active = true
	sa.UpdatedAt = time.Now().UTC()
}

// Deactivate sets the agreement as inactive and refreshes UpdatedAt.
func (sa *ServiceAgreement) Deactivate() {
	sa.Active = false
	sa.UpdatedAt = time.Now().UTC()
}

func generateServiceAgreementID() string {
	return generatePrefixedID(serviceAgreementIDPrefix, serviceAgreementIDBytes)
}
