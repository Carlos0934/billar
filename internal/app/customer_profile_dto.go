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
// The legal-entity is created inline; legal_entity_id must not be provided by callers.
type CreateCustomerProfileCommand struct {
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
	Notes           string `json:"notes"`
}

// PatchCustomerProfileCommand represents a partial update to a customer profile.
// Pointer fields distinguish between "not sent" (nil) and "clear field" (empty string).
// Legal-entity fields are cascaded to the linked legal entity when present.
type PatchCustomerProfileCommand struct {
	// Profile-specific fields.
	Status          *string `json:"status,omitempty"`
	DefaultCurrency *string `json:"default_currency,omitempty"`
	Notes           *string `json:"notes,omitempty"`
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

// hasLegalEntityFields reports whether any LE fields were provided in the patch command.
func (cmd PatchCustomerProfileCommand) hasLegalEntityFields() bool {
	return cmd.LegalEntityType != nil || cmd.LegalName != nil || cmd.TradeName != nil ||
		cmd.TaxID != nil || cmd.Email != nil || cmd.Phone != nil ||
		cmd.Website != nil || cmd.BillingAddress != nil
}

// toLegalEntityPatch converts the LE fields of the command to a PatchLegalEntityCommand.
func (cmd PatchCustomerProfileCommand) toLegalEntityPatch() PatchLegalEntityCommand {
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
