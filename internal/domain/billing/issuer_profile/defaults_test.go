package issuerprofile_test

import (
	"testing"
	"time"

	"github.com/Carlos0934/billar/internal/domain/billing/billing_values"
	"github.com/Carlos0934/billar/internal/domain/billing/issuer_profile"
)

func TestUpdateInvoiceDefaultsUpdatesDefaultsWithoutMutatingIdentity(t *testing.T) {
	profile := newIssuerProfile(t)
	updatedAt := time.Date(2026, time.March, 30, 15, 0, 0, 0, time.UTC)
	euro := billingvalues.MustCurrencyCode("EUR")

	if err := profile.UpdateInvoiceDefaults(issuerprofile.InvoiceDefaults{
		DefaultCurrency:     euro,
		InvoicePrefix:       "ACME-2026",
		PaymentInstructions: "Transfer within 15 days",
	}, updatedAt); err != nil {
		t.Fatalf("UpdateInvoiceDefaults() error = %v", err)
	}

	if got := profile.DefaultCurrency().String(); got != "EUR" {
		t.Fatalf("default currency = %q, want %q", got, "EUR")
	}

	if got := profile.InvoicePrefix(); got != "ACME-2026" {
		t.Fatalf("invoice prefix = %q, want %q", got, "ACME-2026")
	}

	if got := profile.PaymentInstructions(); got != "Transfer within 15 days" {
		t.Fatalf("payment instructions = %q, want %q", got, "Transfer within 15 days")
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

	if got := profile.UpdatedAt(); !got.Equal(updatedAt) {
		t.Fatalf("updated at = %v, want %v", got, updatedAt)
	}
}

func TestUpdateInvoiceDefaultsDefaultsCurrencyToUSDWhenOmitted(t *testing.T) {
	profile := newIssuerProfile(t)
	updatedAt := time.Date(2026, time.March, 30, 16, 0, 0, 0, time.UTC)

	if err := profile.UpdateInvoiceDefaults(issuerprofile.InvoiceDefaults{
		InvoicePrefix:       "INV",
		PaymentInstructions: "Pay by ACH",
	}, updatedAt); err != nil {
		t.Fatalf("UpdateInvoiceDefaults() error = %v", err)
	}

	if got := profile.DefaultCurrency().String(); got != "USD" {
		t.Fatalf("default currency = %q, want %q", got, "USD")
	}
}

func newIssuerProfile(t *testing.T) *issuerprofile.IssuerProfile {
	t.Helper()

	profile, err := issuerprofile.New(
		mustIssuerProfileID(t, "issuer-123"),
		validIdentity(t),
		issuerprofile.InvoiceDefaults{},
		time.Date(2026, time.March, 30, 12, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	return profile
}
