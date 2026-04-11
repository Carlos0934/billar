package core

import (
	"strings"
	"testing"
	"time"
)

func TestInvoiceStatusIsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status InvoiceStatus
		want   bool
	}{
		{status: InvoiceStatusDraft, want: true},
		{status: InvoiceStatusIssued, want: true},
		{status: InvoiceStatusVoided, want: true},
		{status: InvoiceStatus("pending"), want: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(string(tt.status), func(t *testing.T) {
			t.Parallel()

			if got := tt.status.IsValid(); got != tt.want {
				t.Fatalf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewInvoiceLine(t *testing.T) {
	t.Parallel()

	rate, err := NewMoney(10000, "USD")
	if err != nil {
		t.Fatalf("NewMoney(): %v", err)
	}

	line, err := NewInvoiceLine(InvoiceLineParams{
		InvoiceID:          "inv_123",
		ServiceAgreementID: "sa_123",
		TimeEntryID:        "te_123",
		UnitRate:           rate,
	})
	if err != nil {
		t.Fatalf("NewInvoiceLine() error = %v", err)
	}
	if !strings.HasPrefix(line.ID, "inl_") {
		t.Fatalf("ID = %q, want inl_ prefix", line.ID)
	}
	if line.InvoiceID != "inv_123" || line.ServiceAgreementID != "sa_123" || line.TimeEntryID != "te_123" {
		t.Fatalf("NewInvoiceLine() = %#v, want all IDs preserved", line)
	}

	_, err = NewInvoiceLine(InvoiceLineParams{InvoiceID: "", ServiceAgreementID: "sa_123", TimeEntryID: "te_123", UnitRate: rate})
	if err == nil {
		t.Fatal("NewInvoiceLine() error = nil, want blank invoice id rejected")
	}
	_, err = NewInvoiceLine(InvoiceLineParams{InvoiceID: "inv_123", ServiceAgreementID: "", TimeEntryID: "te_123", UnitRate: rate})
	if err == nil {
		t.Fatal("NewInvoiceLine() error = nil, want blank service agreement id rejected")
	}
	_, err = NewInvoiceLine(InvoiceLineParams{InvoiceID: "inv_123", ServiceAgreementID: "sa_123", TimeEntryID: "", UnitRate: rate})
	if err == nil {
		t.Fatal("NewInvoiceLine() error = nil, want blank time entry id rejected")
	}
}

func TestInvoiceLineLineTotal(t *testing.T) {
	t.Parallel()

	rate, err := NewMoney(10000, "USD")
	if err != nil {
		t.Fatalf("NewMoney(): %v", err)
	}
	line := InvoiceLine{UnitRate: rate}

	hours, err := NewHours(15000)
	if err != nil {
		t.Fatalf("NewHours(): %v", err)
	}
	entry := TimeEntry{Hours: hours}

	total := line.LineTotal(entry)
	if total.Amount != 15000 {
		t.Fatalf("LineTotal() amount = %d, want 15000", total.Amount)
	}
	if total.Currency != "USD" {
		t.Fatalf("LineTotal() currency = %q, want USD", total.Currency)
	}
}

func TestNewInvoice(t *testing.T) {
	t.Parallel()

	rate, err := NewMoney(10000, "USD")
	if err != nil {
		t.Fatalf("NewMoney(): %v", err)
	}
	hours, err := NewHours(15000)
	if err != nil {
		t.Fatalf("NewHours(): %v", err)
	}
	entry := TimeEntry{ID: "te_123", Hours: hours}
	line, err := NewInvoiceLine(InvoiceLineParams{InvoiceID: "inv_seed", ServiceAgreementID: "sa_123", TimeEntryID: entry.ID, UnitRate: rate})
	if err != nil {
		t.Fatalf("NewInvoiceLine(): %v", err)
	}

	invoice, err := NewInvoice(InvoiceParams{
		CustomerID: "cus_123",
		Status:     InvoiceStatusDraft,
		Currency:   "USD",
		Lines:      []InvoiceLine{line},
		CreatedAt:  time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("NewInvoice() error = %v", err)
	}
	if !strings.HasPrefix(invoice.ID, "inv_") {
		t.Fatalf("ID = %q, want inv_ prefix", invoice.ID)
	}
	if !invoice.IsDraft() {
		t.Fatal("IsDraft() = false, want true")
	}
	if len(invoice.Lines) != 1 {
		t.Fatalf("len(Lines) = %d, want 1", len(invoice.Lines))
	}
	if invoice.Lines[0].InvoiceID != invoice.ID {
		t.Fatalf("line InvoiceID = %q, want %q", invoice.Lines[0].InvoiceID, invoice.ID)
	}
	total := invoice.Total([]TimeEntry{entry})
	if total.Amount != 15000 {
		t.Fatalf("Total() amount = %d, want 15000", total.Amount)
	}

	_, err = NewInvoice(InvoiceParams{CustomerID: "cus_123", Status: InvoiceStatusDraft, Currency: "USD", Lines: nil})
	if err == nil {
		t.Fatal("NewInvoice() error = nil, want zero lines rejected")
	}

	otherRate, err := NewMoney(10000, "EUR")
	if err != nil {
		t.Fatalf("NewMoney(otherRate): %v", err)
	}
	badLine, err := NewInvoiceLine(InvoiceLineParams{InvoiceID: "inv_seed", ServiceAgreementID: "sa_456", TimeEntryID: "te_456", UnitRate: otherRate})
	if err != nil {
		t.Fatalf("NewInvoiceLine(other): %v", err)
	}
	_, err = NewInvoice(InvoiceParams{CustomerID: "cus_123", Status: InvoiceStatusDraft, Currency: "USD", Lines: []InvoiceLine{badLine}})
	if err == nil {
		t.Fatal("NewInvoice() error = nil, want currency mismatch rejected")
	}
}

func TestInvoiceNewInvoiceLineErrors(t *testing.T) {
	t.Parallel()

	_, err := NewMoney(10000, "USD")
	if err != nil {
		t.Fatalf("NewMoney(): %v", err)
	}
	_, err = NewInvoice(InvoiceParams{CustomerID: "", Status: InvoiceStatusDraft, Currency: "USD", Lines: []InvoiceLine{{}}})
	if err == nil {
		t.Fatal("NewInvoice() error = nil, want blank customer id rejected")
	}
}

func TestNewInvoiceRejectsInvalidStatus(t *testing.T) {
	t.Parallel()

	if _, err := NewInvoice(InvoiceParams{CustomerID: "cus_123", Status: InvoiceStatus("pending"), Currency: "USD", Lines: []InvoiceLine{{InvoiceID: "inv_x", UnitRate: Money{Amount: 1, Currency: "USD"}}}}); err == nil {
		t.Fatal("NewInvoice() error = nil, want invalid status rejected")
	}
}

func TestInvoiceDiscardHelpers(t *testing.T) {
	t.Parallel()

	rate, err := NewMoney(10000, "USD")
	if err != nil {
		t.Fatalf("NewMoney(): %v", err)
	}
	hours, err := NewHours(15000)
	if err != nil {
		t.Fatalf("NewHours(): %v", err)
	}
	entry := TimeEntry{ID: "te_123", Hours: hours}
	line, err := NewInvoiceLine(InvoiceLineParams{InvoiceID: "inv_seed", ServiceAgreementID: "sa_123", TimeEntryID: entry.ID, UnitRate: rate})
	if err != nil {
		t.Fatalf("NewInvoiceLine(): %v", err)
	}
	invoice, err := NewInvoice(InvoiceParams{CustomerID: "cus_123", Status: InvoiceStatusDraft, Currency: "USD", Lines: []InvoiceLine{line}})
	if err != nil {
		t.Fatalf("NewInvoice(): %v", err)
	}
	if !invoice.IsDraft() {
		t.Fatal("IsDraft() = false, want true")
	}
	if got := invoice.Total([]TimeEntry{entry}); got.Amount == 0 {
		t.Fatal("Total() amount = 0, want positive")
	}
	if _, err := NewMoney(0, "USD"); err == nil {
		t.Fatal("NewMoney() zero amount should fail")
	}
}
