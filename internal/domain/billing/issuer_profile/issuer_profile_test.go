package issuerprofile_test

import (
	"errors"
	"testing"
	"time"

	"github.com/Carlos0934/billar/internal/domain/billing/billing_values"
	"github.com/Carlos0934/billar/internal/domain/billing/issuer_profile"
)

func TestNewIssuerProfileCreatesProfileWithDefaultUSD(t *testing.T) {
	issuerID := mustIssuerProfileID(t, "issuer-123")
	createdAt := time.Date(2026, time.March, 30, 12, 0, 0, 0, time.UTC)

	profile, err := issuerprofile.New(issuerID, validIdentity(t), issuerprofile.InvoiceDefaults{}, createdAt)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if got := profile.ID().String(); got != issuerID.String() {
		t.Fatalf("id = %q, want %q", got, issuerID.String())
	}

	if got := profile.LegalName(); got != "Acme Billing LLC" {
		t.Fatalf("legal name = %q, want %q", got, "Acme Billing LLC")
	}

	if got := profile.TaxID().String(); got != "RNC-123456789" {
		t.Fatalf("tax id = %q, want %q", got, "RNC-123456789")
	}

	if got := profile.BillingAddress().Line1(); got != "123 Main Street" {
		t.Fatalf("address line1 = %q, want %q", got, "123 Main Street")
	}

	if got := profile.DefaultCurrency().String(); got != "USD" {
		t.Fatalf("default currency = %q, want %q", got, "USD")
	}

	if got := profile.CreatedAt(); !got.Equal(createdAt) {
		t.Fatalf("created at = %v, want %v", got, createdAt)
	}

	if got := profile.UpdatedAt(); !got.Equal(createdAt) {
		t.Fatalf("updated at = %v, want %v", got, createdAt)
	}
}

func TestNewIssuerProfileRejectsInvalidIdentityData(t *testing.T) {
	issuerID := mustIssuerProfileID(t, "issuer-123")
	createdAt := time.Date(2026, time.March, 30, 12, 0, 0, 0, time.UTC)
	identity := validIdentity(t)

	tests := []struct {
		name    string
		mutate  func(identity *issuerprofile.Identity)
		wantErr error
	}{
		{
			name: "missing legal name",
			mutate: func(identity *issuerprofile.Identity) {
				identity.LegalName = "   "
			},
			wantErr: issuerprofile.ErrLegalNameRequired,
		},
		{
			name: "missing tax id",
			mutate: func(identity *issuerprofile.Identity) {
				identity.TaxID = billingvalues.TaxIdentifier{}
			},
			wantErr: issuerprofile.ErrTaxIDRequired,
		},
		{
			name: "missing billing address",
			mutate: func(identity *issuerprofile.Identity) {
				identity.BillingAddress = billingvalues.Address{}
			},
			wantErr: issuerprofile.ErrBillingAddressRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			candidate := identity
			tt.mutate(&candidate)

			_, err := issuerprofile.New(issuerID, candidate, issuerprofile.InvoiceDefaults{}, createdAt)
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestUpdateIdentityKeepsAggregateIdentityStable(t *testing.T) {
	profile := newIssuerProfile(t)
	updatedAt := time.Date(2026, time.March, 30, 18, 0, 0, 0, time.UTC)
	updatedIdentity := validIdentity(t)
	updatedIdentity.LegalName = "Acme Holdings LLC"
	updatedIdentity.BillingAddress = mustAddress(t, billingvalues.AddressInput{
		Line1:      "456 Ocean Drive",
		City:       "Punta Cana",
		PostalCode: "23000",
		Country:    mustCountryCode(t, "do"),
	})

	if err := profile.UpdateIdentity(updatedIdentity, updatedAt); err != nil {
		t.Fatalf("UpdateIdentity() error = %v", err)
	}

	if got := profile.ID().String(); got != "issuer-123" {
		t.Fatalf("id = %q, want %q", got, "issuer-123")
	}

	if got := profile.LegalName(); got != "Acme Holdings LLC" {
		t.Fatalf("legal name = %q, want %q", got, "Acme Holdings LLC")
	}

	if got := profile.BillingAddress().Line1(); got != "456 Ocean Drive" {
		t.Fatalf("address line1 = %q, want %q", got, "456 Ocean Drive")
	}

	if got := profile.DefaultCurrency().String(); got != "USD" {
		t.Fatalf("default currency = %q, want %q", got, "USD")
	}

	if got := profile.CreatedAt(); !got.Equal(time.Date(2026, time.March, 30, 12, 0, 0, 0, time.UTC)) {
		t.Fatalf("created at = %v, want original time", got)
	}

	if got := profile.UpdatedAt(); !got.Equal(updatedAt) {
		t.Fatalf("updated at = %v, want %v", got, updatedAt)
	}
}

func TestUpdateIdentityRejectsInvalidReplacement(t *testing.T) {
	profile := newIssuerProfile(t)
	updatedAt := time.Date(2026, time.March, 30, 18, 0, 0, 0, time.UTC)
	identity := validIdentity(t)

	tests := []struct {
		name    string
		mutate  func(identity *issuerprofile.Identity)
		wantErr error
	}{
		{
			name: "missing legal name",
			mutate: func(identity *issuerprofile.Identity) {
				identity.LegalName = ""
			},
			wantErr: issuerprofile.ErrLegalNameRequired,
		},
		{
			name: "missing tax id",
			mutate: func(identity *issuerprofile.Identity) {
				identity.TaxID = billingvalues.TaxIdentifier{}
			},
			wantErr: issuerprofile.ErrTaxIDRequired,
		},
		{
			name: "missing billing address",
			mutate: func(identity *issuerprofile.Identity) {
				identity.BillingAddress = billingvalues.Address{}
			},
			wantErr: issuerprofile.ErrBillingAddressRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			candidate := identity
			tt.mutate(&candidate)

			err := profile.UpdateIdentity(candidate, updatedAt)
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func validIdentity(t *testing.T) issuerprofile.Identity {
	t.Helper()

	taxID, err := billingvalues.NewTaxIdentifier("RNC-123456789")
	if err != nil {
		t.Fatalf("NewTaxIdentifier() error = %v", err)
	}

	country, err := billingvalues.NewCountryCode("do")
	if err != nil {
		t.Fatalf("NewCountryCode() error = %v", err)
	}

	address, err := billingvalues.NewAddress(billingvalues.AddressInput{
		Line1:      "123 Main Street",
		City:       "Santo Domingo",
		PostalCode: "10101",
		Country:    country,
	})
	if err != nil {
		t.Fatalf("NewAddress() error = %v", err)
	}

	email, err := billingvalues.NewEmailAddress("billing@example.com")
	if err != nil {
		t.Fatalf("NewEmailAddress() error = %v", err)
	}

	return issuerprofile.Identity{
		LegalName:      "Acme Billing LLC",
		TaxID:          taxID,
		BillingAddress: address,
		Email:          &email,
		Website:        "https://acme.example.com",
	}
}

func mustAddress(t *testing.T, input billingvalues.AddressInput) billingvalues.Address {
	t.Helper()

	address, err := billingvalues.NewAddress(input)
	if err != nil {
		t.Fatalf("NewAddress() error = %v", err)
	}

	return address
}

func mustCountryCode(t *testing.T, value string) billingvalues.CountryCode {
	t.Helper()

	code, err := billingvalues.NewCountryCode(value)
	if err != nil {
		t.Fatalf("NewCountryCode() error = %v", err)
	}

	return code
}

func mustIssuerProfileID(t *testing.T, value string) billingvalues.IssuerProfileID {
	t.Helper()

	id, err := billingvalues.NewIssuerProfileID(value)
	if err != nil {
		t.Fatalf("NewIssuerProfileID() error = %v", err)
	}

	return id
}
