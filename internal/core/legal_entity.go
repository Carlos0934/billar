package core

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	EntityTypeCompany    EntityType = "company"
	EntityTypeIndividual EntityType = "individual"

	legalEntityIDPrefix = "le_"
	legalEntityIDBytes  = 16
)

type EntityType string

func (t EntityType) IsValid() bool {
	switch t {
	case EntityTypeCompany, EntityTypeIndividual:
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

type LegalEntity struct {
	ID             string
	Type           EntityType
	LegalName      string
	TradeName      string
	TaxID          string
	Email          string
	Phone          string
	Website        string
	BillingAddress Address
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type LegalEntityParams struct {
	Type           EntityType
	LegalName      string
	TradeName      string
	TaxID          string
	Email          string
	Phone          string
	Website        string
	BillingAddress Address
}

type LegalEntityPatch struct {
	Type           *EntityType
	LegalName      *string
	TradeName      *string
	TaxID          *string
	Email          *string
	Phone          *string
	Website        *string
	BillingAddress *Address
}

func NewLegalEntity(params LegalEntityParams) (LegalEntity, error) {
	if strings.TrimSpace(params.LegalName) == "" {
		return LegalEntity{}, errors.New("legal entity legal name is required")
	}
	if !params.Type.IsValid() {
		return LegalEntity{}, fmt.Errorf("invalid entity type: %s", params.Type)
	}

	now := time.Now().UTC()
	entity := LegalEntity{
		ID:             generateLegalEntityID(),
		Type:           params.Type,
		LegalName:      strings.TrimSpace(params.LegalName),
		TradeName:      params.TradeName,
		TaxID:          params.TaxID,
		Email:          params.Email,
		Phone:          params.Phone,
		Website:        params.Website,
		BillingAddress: params.BillingAddress,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if entity.ID == "" {
		return LegalEntity{}, errors.New("failed to generate legal entity id")
	}

	return entity, nil
}

func (e *LegalEntity) ApplyPatch(patch LegalEntityPatch) {
	changed := false

	if patch.Type != nil {
		e.Type = *patch.Type
		changed = true
	}
	if patch.LegalName != nil {
		e.LegalName = *patch.LegalName
		changed = true
	}
	if patch.TradeName != nil {
		e.TradeName = *patch.TradeName
		changed = true
	}
	if patch.TaxID != nil {
		e.TaxID = *patch.TaxID
		changed = true
	}
	if patch.Email != nil {
		e.Email = *patch.Email
		changed = true
	}
	if patch.Phone != nil {
		e.Phone = *patch.Phone
		changed = true
	}
	if patch.Website != nil {
		e.Website = *patch.Website
		changed = true
	}
	if patch.BillingAddress != nil {
		e.BillingAddress = *patch.BillingAddress
		changed = true
	}

	if changed {
		e.UpdatedAt = time.Now().UTC()
	}
}

func (e LegalEntity) Validate() error {
	if strings.TrimSpace(e.LegalName) == "" {
		return errors.New("legal entity legal name is required")
	}
	if !e.Type.IsValid() {
		return fmt.Errorf("invalid entity type: %s", e.Type)
	}
	return nil
}

func (e LegalEntity) ValidateDelete() error {
	return nil
}

func generateLegalEntityID() string {
	return generatePrefixedID(legalEntityIDPrefix, legalEntityIDBytes)
}
