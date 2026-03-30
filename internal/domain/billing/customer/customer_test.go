package customer_test

import (
	"errors"
	"testing"
	"time"

	"github.com/Carlos0934/billar/internal/domain/billing/billing_values"
	"github.com/Carlos0934/billar/internal/domain/billing/customer"
)

func TestNewCustomerCreatesActiveInvoiceReadyCustomerWithDefaultUSD(t *testing.T) {
	customerID := mustCustomerID(t, "cust-123")
	createdAt := time.Date(2026, time.March, 30, 9, 0, 0, 0, time.UTC)

	c, err := customer.New(customer.CreateParams{
		ID:          customerID,
		Type:        customer.TypeCompany,
		BillingName: "Acme LLC",
		CreatedAt:   createdAt,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if got := c.ID().String(); got != customerID.String() {
		t.Fatalf("id = %q, want %q", got, customerID.String())
	}

	if got := c.BillingName(); got != "Acme LLC" {
		t.Fatalf("billing name = %q, want %q", got, "Acme LLC")
	}

	if got := c.DefaultCurrency().String(); got != "USD" {
		t.Fatalf("default currency = %q, want %q", got, "USD")
	}

	if got := c.Status(); got != customer.StatusActive {
		t.Fatalf("status = %q, want %q", got, customer.StatusActive)
	}

	if !c.IsInvoiceReady() {
		t.Fatal("expected active customer with billing name to be invoice-ready")
	}

	if got := c.CreatedAt(); !got.Equal(createdAt) {
		t.Fatalf("created at = %v, want %v", got, createdAt)
	}

	if got := c.UpdatedAt(); !got.Equal(createdAt) {
		t.Fatalf("updated at = %v, want %v", got, createdAt)
	}
}

func TestNewCustomerRejectsMissingBillingName(t *testing.T) {
	customerID := mustCustomerID(t, "cust-123")

	_, err := customer.New(customer.CreateParams{
		ID:        customerID,
		Type:      customer.TypeCompany,
		CreatedAt: time.Date(2026, time.March, 30, 9, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatal("expected error for missing billing name")
	}
	if !errors.Is(err, customer.ErrBillingNameRequired) {
		t.Fatalf("err = %v, want %v", err, customer.ErrBillingNameRequired)
	}
}

func TestCustomerLifecycleAffectsInvoiceReadiness(t *testing.T) {
	customerID := mustCustomerID(t, "cust-123")
	createdAt := time.Date(2026, time.March, 30, 9, 0, 0, 0, time.UTC)

	c, err := customer.New(customer.CreateParams{
		ID:          customerID,
		Type:        customer.TypeCompany,
		BillingName: "Acme LLC",
		CreatedAt:   createdAt,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	deactivatedAt := createdAt.Add(2 * time.Hour)
	if err := c.Deactivate(deactivatedAt); err != nil {
		t.Fatalf("Deactivate() error = %v", err)
	}

	if got := c.Status(); got != customer.StatusInactive {
		t.Fatalf("status = %q, want %q", got, customer.StatusInactive)
	}

	if c.IsInvoiceReady() {
		t.Fatal("expected inactive customer to be not invoice-ready")
	}

	if got := c.UpdatedAt(); !got.Equal(deactivatedAt) {
		t.Fatalf("updated at after deactivate = %v, want %v", got, deactivatedAt)
	}

	activatedAt := deactivatedAt.Add(2 * time.Hour)
	if err := c.Activate(activatedAt); err != nil {
		t.Fatalf("Activate() error = %v", err)
	}

	if got := c.Status(); got != customer.StatusActive {
		t.Fatalf("status = %q, want %q", got, customer.StatusActive)
	}

	if !c.IsInvoiceReady() {
		t.Fatal("expected reactivated customer to be invoice-ready")
	}

	if got := c.UpdatedAt(); !got.Equal(activatedAt) {
		t.Fatalf("updated at after activate = %v, want %v", got, activatedAt)
	}
}

func mustCustomerID(t *testing.T, value string) billingvalues.CustomerID {
	t.Helper()

	id, err := billingvalues.NewCustomerID(value)
	if err != nil {
		t.Fatalf("NewCustomerID() error = %v", err)
	}

	return id
}
