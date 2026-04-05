package app

import (
	"time"

	"github.com/Carlos0934/billar/internal/core"
)

type CustomerProfileDTO struct {
	ID              string `json:"id" toon:"id"`
	LegalEntityID   string `json:"legal_entity_id" toon:"legal_entity_id"`
	Status          string `json:"status" toon:"status"`
	DefaultCurrency string `json:"default_currency" toon:"default_currency"`
	Notes           string `json:"notes" toon:"notes"`
	CreatedAt       string `json:"created_at" toon:"created_at"`
	UpdatedAt       string `json:"updated_at" toon:"updated_at"`
}

func customerProfileToDTO(profile core.CustomerProfile) CustomerProfileDTO {
	return CustomerProfileDTO{
		ID:              profile.ID,
		LegalEntityID:   profile.LegalEntityID,
		Status:          string(profile.Status),
		DefaultCurrency: profile.DefaultCurrency,
		Notes:           profile.Notes,
		CreatedAt:       formatCustomerProfileTime(profile.CreatedAt),
		UpdatedAt:       formatCustomerProfileTime(profile.UpdatedAt),
	}
}

func formatCustomerProfileTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

// CreateCustomerProfileCommand represents the input for creating a customer profile.
type CreateCustomerProfileCommand struct {
	LegalEntityID   string `json:"legal_entity_id"`
	DefaultCurrency string `json:"default_currency"`
	Notes           string `json:"notes"`
}

// PatchCustomerProfileCommand represents a partial update to a customer profile.
// Pointer fields distinguish between "not sent" (nil) and "clear field" (empty string).
type PatchCustomerProfileCommand struct {
	Status          *string `json:"status,omitempty"`
	DefaultCurrency *string `json:"default_currency,omitempty"`
	Notes           *string `json:"notes,omitempty"`
}

func patchToCoreCustomerProfilePatch(cmd PatchCustomerProfileCommand) core.CustomerProfilePatch {
	patch := core.CustomerProfilePatch{}
	if cmd.Status != nil {
		s := core.CustomerProfileStatus(*cmd.Status)
		patch.Status = &s
	}
	if cmd.DefaultCurrency != nil {
		patch.DefaultCurrency = cmd.DefaultCurrency
	}
	if cmd.Notes != nil {
		patch.Notes = cmd.Notes
	}
	return patch
}
