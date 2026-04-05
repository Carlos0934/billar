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

func addressToDTO(address core.Address) AddressDTO {
	return AddressDTO{
		Street:     address.Street,
		City:       address.City,
		State:      address.State,
		PostalCode: address.PostalCode,
		Country:    address.Country,
	}
}

func addressFromDTO(dto AddressDTO) core.Address {
	return core.Address{
		Street:     dto.Street,
		City:       dto.City,
		State:      dto.State,
		PostalCode: dto.PostalCode,
		Country:    dto.Country,
	}
}

type LegalEntityDTO struct {
	ID             string     `json:"id" toon:"id"`
	Type           string     `json:"type" toon:"type"`
	LegalName      string     `json:"legal_name" toon:"legal_name"`
	TradeName      string     `json:"trade_name" toon:"trade_name"`
	TaxID          string     `json:"tax_id" toon:"tax_id"`
	Email          string     `json:"email" toon:"email"`
	Phone          string     `json:"phone" toon:"phone"`
	Website        string     `json:"website" toon:"website"`
	BillingAddress AddressDTO `json:"billing_address" toon:"billing_address"`
	CreatedAt      string     `json:"created_at" toon:"created_at"`
	UpdatedAt      string     `json:"updated_at" toon:"updated_at"`
}

func legalEntityToDTO(entity core.LegalEntity) LegalEntityDTO {
	return LegalEntityDTO{
		ID:             entity.ID,
		Type:           string(entity.Type),
		LegalName:      entity.LegalName,
		TradeName:      entity.TradeName,
		TaxID:          entity.TaxID,
		Email:          entity.Email,
		Phone:          entity.Phone,
		Website:        entity.Website,
		BillingAddress: addressToDTO(entity.BillingAddress),
		CreatedAt:      formatLegalEntityTime(entity.CreatedAt),
		UpdatedAt:      formatLegalEntityTime(entity.UpdatedAt),
	}
}

func formatLegalEntityTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

// CreateLegalEntityCommand represents the input for creating a legal entity.
type CreateLegalEntityCommand struct {
	Type           string     `json:"type"`
	LegalName      string     `json:"legal_name"`
	TradeName      string     `json:"trade_name"`
	TaxID          string     `json:"tax_id"`
	Email          string     `json:"email"`
	Phone          string     `json:"phone"`
	Website        string     `json:"website"`
	BillingAddress AddressDTO `json:"billing_address"`
}

// PatchLegalEntityCommand represents a partial update to a legal entity.
// Pointer fields distinguish between "not sent" (nil) and "clear field" (empty string).
type PatchLegalEntityCommand struct {
	Type           *string     `json:"type,omitempty"`
	LegalName      *string     `json:"legal_name,omitempty"`
	TradeName      *string     `json:"trade_name,omitempty"`
	TaxID          *string     `json:"tax_id,omitempty"`
	Email          *string     `json:"email,omitempty"`
	Phone          *string     `json:"phone,omitempty"`
	Website        *string     `json:"website,omitempty"`
	BillingAddress *AddressDTO `json:"billing_address,omitempty"`
}

func patchToCoreLegalEntityPatch(cmd PatchLegalEntityCommand) core.LegalEntityPatch {
	patch := core.LegalEntityPatch{}
	if cmd.Type != nil {
		t := core.EntityType(*cmd.Type)
		patch.Type = &t
	}
	if cmd.LegalName != nil {
		patch.LegalName = cmd.LegalName
	}
	if cmd.TradeName != nil {
		patch.TradeName = cmd.TradeName
	}
	if cmd.TaxID != nil {
		patch.TaxID = cmd.TaxID
	}
	if cmd.Email != nil {
		patch.Email = cmd.Email
	}
	if cmd.Phone != nil {
		patch.Phone = cmd.Phone
	}
	if cmd.Website != nil {
		patch.Website = cmd.Website
	}
	if cmd.BillingAddress != nil {
		addr := addressFromDTO(*cmd.BillingAddress)
		patch.BillingAddress = &addr
	}
	return patch
}
