package app

import (
	"time"

	"github.com/Carlos0934/billar/internal/core"
)

type IssuerProfileDTO struct {
	ID              string `json:"id" toon:"id"`
	LegalEntityID   string `json:"legal_entity_id" toon:"legal_entity_id"`
	DefaultCurrency string `json:"default_currency" toon:"default_currency"`
	DefaultNotes    string `json:"default_notes" toon:"default_notes"`
	CreatedAt       string `json:"created_at" toon:"created_at"`
	UpdatedAt       string `json:"updated_at" toon:"updated_at"`
}

func issuerProfileToDTO(profile core.IssuerProfile) IssuerProfileDTO {
	return IssuerProfileDTO{
		ID:              profile.ID,
		LegalEntityID:   profile.LegalEntityID,
		DefaultCurrency: profile.DefaultCurrency,
		DefaultNotes:    profile.DefaultNotes,
		CreatedAt:       formatIssuerProfileTime(profile.CreatedAt),
		UpdatedAt:       formatIssuerProfileTime(profile.UpdatedAt),
	}
}

func formatIssuerProfileTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

// CreateIssuerProfileCommand represents the input for creating an issuer profile.
type CreateIssuerProfileCommand struct {
	LegalEntityID   string `json:"legal_entity_id"`
	DefaultCurrency string `json:"default_currency"`
	DefaultNotes    string `json:"default_notes"`
}

// PatchIssuerProfileCommand represents a partial update to an issuer profile.
// Pointer fields distinguish between "not sent" (nil) and "clear field" (empty string).
type PatchIssuerProfileCommand struct {
	DefaultCurrency *string `json:"default_currency,omitempty"`
	DefaultNotes    *string `json:"default_notes,omitempty"`
}

func patchToCoreIssuerProfilePatch(cmd PatchIssuerProfileCommand) core.IssuerProfilePatch {
	patch := core.IssuerProfilePatch{}
	if cmd.DefaultCurrency != nil {
		patch.DefaultCurrency = cmd.DefaultCurrency
	}
	if cmd.DefaultNotes != nil {
		patch.DefaultNotes = cmd.DefaultNotes
	}
	return patch
}
