package serviceagreement_test

import (
	"errors"
	"testing"
	"time"

	"github.com/Carlos0934/billar/internal/domain/billing/billing_values"
	"github.com/Carlos0934/billar/internal/domain/billing/service_agreement"
)

func TestNewServiceAgreementCreatesHourlyAgreementWithDefaultUSD(t *testing.T) {
	agreementID := mustServiceAgreementID(t, "agr-123")
	customerID := mustCustomerID(t, "cust-123")
	createdAt := time.Date(2026, time.March, 30, 10, 0, 0, 0, time.UTC)

	agreement, err := serviceagreement.New(serviceagreement.CreateParams{
		ID:               agreementID,
		CustomerID:       customerID,
		BillingMode:      serviceagreement.ModeHourly,
		HourlyRateAmount: 125000,
		CreatedAt:        createdAt,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if got := agreement.ID().String(); got != agreementID.String() {
		t.Fatalf("id = %q, want %q", got, agreementID.String())
	}

	if got := agreement.CustomerID().String(); got != customerID.String() {
		t.Fatalf("customer id = %q, want %q", got, customerID.String())
	}

	if got := agreement.BillingMode(); got != serviceagreement.ModeHourly {
		t.Fatalf("billing mode = %q, want %q", got, serviceagreement.ModeHourly)
	}

	if got := agreement.HourlyRate().Amount(); got != 125000 {
		t.Fatalf("hourly rate amount = %d, want %d", got, 125000)
	}

	if got := agreement.Currency().String(); got != "USD" {
		t.Fatalf("currency = %q, want %q", got, "USD")
	}

	if agreement.Validity() != nil {
		t.Fatal("expected nil validity when omitted")
	}

	if !agreement.IsBillableOn(createdAt) {
		t.Fatal("expected active agreement without validity window to be billable")
	}
}

func TestNewServiceAgreementRejectsUnsupportedModeOrNonPositiveRate(t *testing.T) {
	agreementID := mustServiceAgreementID(t, "agr-123")
	customerID := mustCustomerID(t, "cust-123")
	createdAt := time.Date(2026, time.March, 30, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name             string
		billingMode      serviceagreement.BillingMode
		HourlyRateAmount int64
		wantErr          error
	}{
		{name: "unsupported billing mode", billingMode: serviceagreement.BillingMode("fixed_fee"), HourlyRateAmount: 125000, wantErr: serviceagreement.ErrBillingModeMustBeHourly},
		{name: "zero rate", billingMode: serviceagreement.ModeHourly, HourlyRateAmount: 0, wantErr: serviceagreement.ErrHourlyRateMustBePositive},
		{name: "negative rate", billingMode: serviceagreement.ModeHourly, HourlyRateAmount: -1, wantErr: serviceagreement.ErrHourlyRateMustBePositive},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := serviceagreement.New(serviceagreement.CreateParams{
				ID:               agreementID,
				CustomerID:       customerID,
				BillingMode:      tt.billingMode,
				HourlyRateAmount: tt.HourlyRateAmount,
				CreatedAt:        createdAt,
			})
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestServiceAgreementBillabilityAndRateChanges(t *testing.T) {
	agreementID := mustServiceAgreementID(t, "agr-123")
	customerID := mustCustomerID(t, "cust-123")
	createdAt := time.Date(2026, time.March, 30, 10, 0, 0, 0, time.UTC)
	validity := mustDateRange(
		t,
		time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, time.April, 30, 0, 0, 0, 0, time.UTC),
	)

	agreement, err := serviceagreement.New(serviceagreement.CreateParams{
		ID:               agreementID,
		CustomerID:       customerID,
		BillingMode:      serviceagreement.ModeHourly,
		HourlyRateAmount: 125000,
		Validity:         &validity,
		CreatedAt:        createdAt,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if !agreement.IsBillableOn(time.Date(2026, time.April, 15, 12, 0, 0, 0, time.UTC)) {
		t.Fatal("expected in-range active agreement to be billable")
	}

	if agreement.IsBillableOn(time.Date(2026, time.May, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatal("expected out-of-range agreement to be not billable")
	}

	changedAt := createdAt.Add(3 * time.Hour)
	if err := agreement.ChangeHourlyRate(150000, changedAt); err != nil {
		t.Fatalf("ChangeHourlyRate() error = %v", err)
	}

	if got := agreement.HourlyRate().Amount(); got != 150000 {
		t.Fatalf("hourly rate amount = %d, want %d", got, 150000)
	}

	if got := agreement.UpdatedAt(); !got.Equal(changedAt) {
		t.Fatalf("updated at after rate change = %v, want %v", got, changedAt)
	}

	deactivatedAt := changedAt.Add(3 * time.Hour)
	if err := agreement.Deactivate(deactivatedAt); err != nil {
		t.Fatalf("Deactivate() error = %v", err)
	}

	if agreement.IsBillableOn(time.Date(2026, time.April, 15, 0, 0, 0, 0, time.UTC)) {
		t.Fatal("expected inactive agreement to be not billable")
	}
}

func mustServiceAgreementID(t *testing.T, value string) billingvalues.ServiceAgreementID {
	t.Helper()

	id, err := billingvalues.NewServiceAgreementID(value)
	if err != nil {
		t.Fatalf("NewServiceAgreementID() error = %v", err)
	}

	return id
}

func mustCustomerID(t *testing.T, value string) billingvalues.CustomerID {
	t.Helper()

	id, err := billingvalues.NewCustomerID(value)
	if err != nil {
		t.Fatalf("NewCustomerID() error = %v", err)
	}

	return id
}

func mustDateRange(t *testing.T, start, end time.Time) billingvalues.DateRange {
	t.Helper()

	dateRange, err := billingvalues.NewDateRange(start, end)
	if err != nil {
		t.Fatalf("NewDateRange() error = %v", err)
	}

	return dateRange
}
