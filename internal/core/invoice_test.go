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
		{status: InvoiceStatusDiscarded, want: true},
		{status: InvoiceStatus("voided"), want: false},
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

func TestInvoiceIssue_HappyPath(t *testing.T) {
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

	issuedAt := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	if err := invoice.Issue("INV-2026-0001", issuedAt); err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	if invoice.Status != InvoiceStatusIssued {
		t.Fatalf("Status = %q, want issued", invoice.Status)
	}
	if invoice.InvoiceNumber != "INV-2026-0001" {
		t.Fatalf("InvoiceNumber = %q, want INV-2026-0001", invoice.InvoiceNumber)
	}
	if !invoice.IssuedAt.Equal(issuedAt) {
		t.Fatalf("IssuedAt = %s, want %s", invoice.IssuedAt, issuedAt)
	}
	if !invoice.UpdatedAt.Equal(issuedAt) {
		t.Fatalf("UpdatedAt = %s, want %s", invoice.UpdatedAt, issuedAt)
	}

	if err := invoice.Issue("INV-2026-0002", issuedAt.Add(time.Hour)); err == nil {
		t.Fatal("Issue() error = nil, want reissue rejected")
	}
}

func TestInvoiceIssue_RejectsNonDraft(t *testing.T) {
	t.Parallel()

	invoice := Invoice{Status: InvoiceStatusIssued}
	if err := invoice.Issue("INV-1", time.Now().UTC()); err == nil {
		t.Fatal("Issue() error = nil, want non-draft rejected")
	}
}

func TestInvoiceIssue_RejectsInvalidInputs(t *testing.T) {
	t.Parallel()

	invoice := Invoice{Status: InvoiceStatusDraft}
	if err := invoice.Issue("", time.Now().UTC()); err == nil {
		t.Fatal("Issue() error = nil, want blank number rejected")
	}
	if err := invoice.Issue("INV-1", time.Time{}); err == nil {
		t.Fatal("Issue() error = nil, want zero issued time rejected")
	}
}

func TestInvoiceIsIssued(t *testing.T) {
	t.Parallel()

	issued := Invoice{Status: InvoiceStatusIssued}
	if !issued.IsIssued() {
		t.Fatal("IsIssued() = false, want true")
	}
	if (Invoice{Status: InvoiceStatusDraft}).IsIssued() {
		t.Fatal("IsIssued() = true for draft invoice")
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

func TestInvoiceDiscard_IssuedToDiscarded(t *testing.T) {
	t.Parallel()

	invoice := Invoice{Status: InvoiceStatusIssued}
	discardedAt := time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC)

	if err := invoice.Discard(discardedAt); err != nil {
		t.Fatalf("Discard() error = %v", err)
	}
	if invoice.Status != InvoiceStatusDiscarded {
		t.Fatalf("Status = %q, want discarded", invoice.Status)
	}
	if !invoice.DiscardedAt.Equal(discardedAt) {
		t.Fatalf("DiscardedAt = %v, want %v", invoice.DiscardedAt, discardedAt)
	}
}

func TestInvoiceDiscard_RejectsDraft(t *testing.T) {
	t.Parallel()

	invoice := Invoice{Status: InvoiceStatusDraft}
	if err := invoice.Discard(time.Now().UTC()); err == nil {
		t.Fatal("Discard() error = nil, want draft rejection")
	}
}

func TestInvoiceDiscard_RejectsAlreadyDiscarded(t *testing.T) {
	t.Parallel()

	invoice := Invoice{Status: InvoiceStatusDiscarded}
	if err := invoice.Discard(time.Now().UTC()); err == nil {
		t.Fatal("Discard() error = nil, want already-discarded rejection")
	}
}

func TestInvoiceIsDiscarded(t *testing.T) {
	t.Parallel()

	discarded := Invoice{Status: InvoiceStatusDiscarded}
	if !discarded.IsDiscarded() {
		t.Fatal("IsDiscarded() = false, want true")
	}
	if (Invoice{Status: InvoiceStatusDraft}).IsDiscarded() {
		t.Fatal("IsDiscarded() = true for draft invoice")
	}
	if (Invoice{Status: InvoiceStatusIssued}).IsDiscarded() {
		t.Fatal("IsDiscarded() = true for issued invoice")
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
