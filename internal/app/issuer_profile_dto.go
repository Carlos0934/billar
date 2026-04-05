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
// The legal-entity is created inline; legal_entity_id must not be provided by callers.
type CreateIssuerProfileCommand struct {
	// Legal entity fields — supplied inline; the service creates the entity automatically.
	LegalEntityType string     `json:"type"`
	LegalName       string     `json:"legal_name"`
	TradeName       string     `json:"trade_name"`
	TaxID           string     `json:"tax_id"`
	Email           string     `json:"email"`
	Phone           string     `json:"phone"`
	Website         string     `json:"website"`
	BillingAddress  AddressDTO `json:"billing_address"`
	// Profile-specific fields.
	DefaultCurrency string `json:"default_currency"`
	DefaultNotes    string `json:"default_notes"`
}

// PatchIssuerProfileCommand represents a partial update to an issuer profile.
// Pointer fields distinguish between "not sent" (nil) and "clear field" (empty string).
// Legal-entity fields are cascaded to the linked legal entity when present.
type PatchIssuerProfileCommand struct {
	// Profile-specific fields.
	DefaultCurrency *string `json:"default_currency,omitempty"`
	DefaultNotes    *string `json:"default_notes,omitempty"`
	// Legal entity fields — cascaded to the linked entity when present.
	LegalEntityType *string     `json:"type,omitempty"`
	LegalName       *string     `json:"legal_name,omitempty"`
	TradeName       *string     `json:"trade_name,omitempty"`
	TaxID           *string     `json:"tax_id,omitempty"`
	Email           *string     `json:"email,omitempty"`
	Phone           *string     `json:"phone,omitempty"`
	Website         *string     `json:"website,omitempty"`
	BillingAddress  *AddressDTO `json:"billing_address,omitempty"`
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

// hasLegalEntityFields reports whether any LE fields were provided in the patch command.
func (cmd PatchIssuerProfileCommand) hasLegalEntityFields() bool {
	return cmd.LegalEntityType != nil || cmd.LegalName != nil || cmd.TradeName != nil ||
		cmd.TaxID != nil || cmd.Email != nil || cmd.Phone != nil ||
		cmd.Website != nil || cmd.BillingAddress != nil
}

// toLegalEntityPatch converts the LE fields of the command to a PatchLegalEntityCommand.
func (cmd PatchIssuerProfileCommand) toLegalEntityPatch() PatchLegalEntityCommand {
	le := PatchLegalEntityCommand{
		Type:      cmd.LegalEntityType,
		LegalName: cmd.LegalName,
		TradeName: cmd.TradeName,
		TaxID:     cmd.TaxID,
		Email:     cmd.Email,
		Phone:     cmd.Phone,
		Website:   cmd.Website,
	}
	if cmd.BillingAddress != nil {
		le.BillingAddress = cmd.BillingAddress
	}
	return le
}
