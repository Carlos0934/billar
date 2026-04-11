package app

import (
	"time"

	"github.com/Carlos0934/billar/internal/core"
)

// ServiceAgreementDTO is the canonical read model returned from AgreementService operations.
type ServiceAgreementDTO struct {
	ID                string  `json:"id" toon:"id"`
	CustomerProfileID string  `json:"customer_profile_id" toon:"customer_profile_id"`
	Name              string  `json:"name" toon:"name"`
	Description       string  `json:"description" toon:"description"`
	BillingMode       string  `json:"billing_mode" toon:"billing_mode"`
	HourlyRate        int64   `json:"hourly_rate" toon:"hourly_rate"`
	Currency          string  `json:"currency" toon:"currency"`
	Active            bool    `json:"active" toon:"active"`
	ValidFrom         *string `json:"valid_from,omitempty" toon:"valid_from"`
	ValidUntil        *string `json:"valid_until,omitempty" toon:"valid_until"`
	CreatedAt         string  `json:"created_at" toon:"created_at"`
	UpdatedAt         string  `json:"updated_at" toon:"updated_at"`
}

// CreateServiceAgreementCommand carries all inputs needed to create a new ServiceAgreement.
type CreateServiceAgreementCommand struct {
	CustomerProfileID string     `json:"customer_profile_id"`
	Name              string     `json:"name"`
	Description       string     `json:"description"`
	BillingMode       string     `json:"billing_mode"`
	HourlyRate        int64      `json:"hourly_rate"`
	Currency          string     `json:"currency"`
	ValidFrom         *time.Time `json:"valid_from,omitempty"`
	ValidUntil        *time.Time `json:"valid_until,omitempty"`
}

// UpdateServiceAgreementRateCommand carries the new hourly rate for an existing agreement.
type UpdateServiceAgreementRateCommand struct {
	HourlyRate int64 `json:"hourly_rate"`
}

func serviceAgreementToDTO(sa core.ServiceAgreement) ServiceAgreementDTO {
	dto := ServiceAgreementDTO{
		ID:                sa.ID,
		CustomerProfileID: sa.CustomerProfileID,
		Name:              sa.Name,
		Description:       sa.Description,
		BillingMode:       string(sa.BillingMode),
		HourlyRate:        sa.HourlyRate,
		Currency:          sa.Currency,
		Active:            sa.Active,
		CreatedAt:         formatServiceAgreementTime(sa.CreatedAt),
		UpdatedAt:         formatServiceAgreementTime(sa.UpdatedAt),
	}

	if sa.ValidFrom != nil {
		s := sa.ValidFrom.UTC().Format(time.RFC3339)
		dto.ValidFrom = &s
	}
	if sa.ValidUntil != nil {
		s := sa.ValidUntil.UTC().Format(time.RFC3339)
		dto.ValidUntil = &s
	}
	return dto
}

func formatServiceAgreementTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}
