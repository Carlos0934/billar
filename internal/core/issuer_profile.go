package core

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"
)

const (
	issuerProfileIDPrefix   = "iss_"
	issuerProfileIDBytes    = 16
	issuerProfileIDHexChars = 32
)

type IssuerProfile struct {
	ID              string
	LegalEntityID   string
	DefaultCurrency string
	DefaultNotes    string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type IssuerProfileParams struct {
	LegalEntityID   string
	DefaultCurrency string
	DefaultNotes    string
}

type IssuerProfilePatch struct {
	DefaultCurrency *string
	DefaultNotes    *string
}

func NewIssuerProfile(params IssuerProfileParams) (IssuerProfile, error) {
	if strings.TrimSpace(params.LegalEntityID) == "" {
		return IssuerProfile{}, errors.New("issuer profile legal entity id is required")
	}
	if strings.TrimSpace(params.DefaultCurrency) == "" {
		return IssuerProfile{}, errors.New("issuer profile default currency is required")
	}

	now := time.Now().UTC()
	profile := IssuerProfile{
		ID:              generateIssuerProfileID(),
		LegalEntityID:   strings.TrimSpace(params.LegalEntityID),
		DefaultCurrency: strings.TrimSpace(params.DefaultCurrency),
		DefaultNotes:    params.DefaultNotes,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if profile.ID == "" {
		return IssuerProfile{}, errors.New("failed to generate issuer profile id")
	}

	return profile, nil
}

func (p *IssuerProfile) ApplyPatch(patch IssuerProfilePatch) {
	changed := false

	if patch.DefaultCurrency != nil {
		p.DefaultCurrency = *patch.DefaultCurrency
		changed = true
	}
	if patch.DefaultNotes != nil {
		p.DefaultNotes = *patch.DefaultNotes
		changed = true
	}

	if changed {
		p.UpdatedAt = time.Now().UTC()
	}
}

func (p IssuerProfile) Validate() error {
	if strings.TrimSpace(p.LegalEntityID) == "" {
		return errors.New("issuer profile legal entity id is required")
	}
	if strings.TrimSpace(p.DefaultCurrency) == "" {
		return errors.New("issuer profile default currency is required")
	}
	return nil
}

func (p IssuerProfile) ValidateDelete() error {
	return nil
}

func generateIssuerProfileID() string {
	buf := make([]byte, issuerProfileIDBytes)
	if _, err := rand.Read(buf); err != nil {
		return ""
	}

	encoded := hex.EncodeToString(buf)
	if len(encoded) != issuerProfileIDHexChars {
		return ""
	}

	return issuerProfileIDPrefix + encoded
}
