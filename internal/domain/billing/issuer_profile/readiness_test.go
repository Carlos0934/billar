package issuerprofile_test

import (
	"testing"
	"time"

	"github.com/Carlos0934/billar/internal/domain/billing/billing_values"
	"github.com/Carlos0934/billar/internal/domain/billing/issuer_profile"
)

func TestEvaluateIssuanceReadinessReturnsReadyWhenProfileIsComplete(t *testing.T) {
	euro := billingvalues.MustCurrencyCode("EUR")
	profile, err := issuerprofile.New(
		mustIssuerProfileID(t, "issuer-123"),
		validIdentity(t),
		issuerprofile.InvoiceDefaults{
			DefaultCurrency:     euro,
			InvoicePrefix:       "ACME-2026",
			PaymentInstructions: "Transfer within 15 days",
		},
		time.Date(2026, time.March, 30, 12, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	readiness := profile.EvaluateIssuanceReadiness()
	if !readiness.Ready {
		t.Fatal("expected complete profile to be ready")
	}
	if len(readiness.Missing) != 0 {
		t.Fatalf("missing = %v, want none", readiness.Missing)
	}
}

func TestEvaluateIssuanceReadinessReportsOrderedMissingRequirements(t *testing.T) {
	readiness := (&issuerprofile.IssuerProfile{}).EvaluateIssuanceReadiness()

	if readiness.Ready {
		t.Fatal("expected zero-value profile to be not ready")
	}

	want := []issuerprofile.MissingRequirement{
		issuerprofile.MissingLegalName,
		issuerprofile.MissingTaxID,
		issuerprofile.MissingBillingAddress,
		issuerprofile.MissingDirectContact,
		issuerprofile.MissingDefaultCurrency,
		issuerprofile.MissingInvoicePrefix,
		issuerprofile.MissingPaymentInstructions,
	}

	assertMissingRequirements(t, readiness.Missing, want)
}

func TestEvaluateIssuanceReadinessReportsMissingContactAndInvoiceDefaultsInOrder(t *testing.T) {
	identity := validIdentity(t)
	identity.Email = nil
	identity.Phone = nil

	profile, err := issuerprofile.New(
		mustIssuerProfileID(t, "issuer-123"),
		identity,
		issuerprofile.InvoiceDefaults{},
		time.Date(2026, time.March, 30, 12, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	readiness := profile.EvaluateIssuanceReadiness()
	if readiness.Ready {
		t.Fatal("expected profile missing contact and invoice defaults to be not ready")
	}

	want := []issuerprofile.MissingRequirement{
		issuerprofile.MissingDirectContact,
		issuerprofile.MissingInvoicePrefix,
		issuerprofile.MissingPaymentInstructions,
	}

	assertMissingRequirements(t, readiness.Missing, want)
}

func TestEvaluateIssuanceReadinessAllowsPhoneWithoutWebsite(t *testing.T) {
	phone := mustPhoneNumber(t, "+1 809 555 1234")
	identity := validIdentity(t)
	identity.Email = nil
	identity.Phone = &phone
	identity.Website = ""

	profile, err := issuerprofile.New(
		mustIssuerProfileID(t, "issuer-123"),
		identity,
		issuerprofile.InvoiceDefaults{
			InvoicePrefix:       "INV",
			PaymentInstructions: "Pay by ACH",
		},
		time.Date(2026, time.March, 30, 12, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	readiness := profile.EvaluateIssuanceReadiness()
	if !readiness.Ready {
		t.Fatalf("expected phone-only contact to satisfy readiness, missing = %v", readiness.Missing)
	}
}

func assertMissingRequirements(t *testing.T, got, want []issuerprofile.MissingRequirement) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("missing length = %d, want %d (%v)", len(got), len(want), got)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("missing[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func mustPhoneNumber(t *testing.T, value string) billingvalues.PhoneNumber {
	t.Helper()

	phone, err := billingvalues.NewPhoneNumber(value)
	if err != nil {
		t.Fatalf("NewPhoneNumber() error = %v", err)
	}

	return phone
}
