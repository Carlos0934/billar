package core

import (
	"errors"
	"strings"
	"testing"
)

func TestNewQuoteDefaultsAndID(t *testing.T) {
	t.Parallel()

	quote, err := NewQuote(QuoteParams{CustomerID: " cus_123 ", Currency: " USD ", Notes: "offer"})
	if err != nil {
		t.Fatalf("NewQuote() error = %v", err)
	}
	if !strings.HasPrefix(quote.ID, "quo_") {
		t.Fatalf("Quote ID = %q, want quo_ prefix", quote.ID)
	}
	if quote.CustomerID != "cus_123" {
		t.Fatalf("CustomerID = %q, want cus_123", quote.CustomerID)
	}
	if quote.Status != QuoteStatusDraft {
		t.Fatalf("Status = %q, want draft", quote.Status)
	}
	if quote.Total().Amount != 0 || quote.Total().Currency != "USD" {
		t.Fatalf("Total() = %+v, want 0 USD", quote.Total())
	}
}

func TestQuoteLineValidationAndTotals(t *testing.T) {
	t.Parallel()

	quote, err := NewQuote(QuoteParams{CustomerID: "cus_123", Currency: "USD"})
	if err != nil {
		t.Fatalf("NewQuote() error = %v", err)
	}
	line, err := NewQuoteLine(QuoteLineParams{QuoteID: quote.ID, ServiceAgreementID: "sa_123", Description: "Discovery", QuantityMin: 60, UnitRate: Money{Amount: 1000, Currency: "USD"}}, quote.Currency)
	if err != nil {
		t.Fatalf("NewQuoteLine() error = %v", err)
	}
	second, err := NewQuoteLine(QuoteLineParams{QuoteID: quote.ID, ServiceAgreementID: "sa_456", Description: "Support", QuantityMin: 30, UnitRate: Money{Amount: 666, Currency: "USD"}}, quote.Currency)
	if err != nil {
		t.Fatalf("NewQuoteLine() second error = %v", err)
	}
	quote.Lines = []QuoteLine{line, second}

	if !strings.HasPrefix(line.ID, "qol_") {
		t.Fatalf("Line ID = %q, want qol_ prefix", line.ID)
	}
	if got := line.LineTotal(); got.Amount != 1000 || got.Currency != "USD" {
		t.Fatalf("LineTotal() = %+v, want 1000 USD", got)
	}
	if got := quote.Total(); got.Amount != 1333 || got.Currency != "USD" {
		t.Fatalf("Total() = %+v, want 1333 USD", got)
	}
}

func TestNewQuoteLineRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		params QuoteLineParams
		want   string
	}{
		{name: "missing quote", params: QuoteLineParams{ServiceAgreementID: "sa_123", Description: "Work", QuantityMin: 60, UnitRate: Money{Amount: 1000, Currency: "USD"}}, want: "quote id is required"},
		{name: "missing agreement", params: QuoteLineParams{QuoteID: "quo_123", Description: "Work", QuantityMin: 60, UnitRate: Money{Amount: 1000, Currency: "USD"}}, want: "service agreement id is required"},
		{name: "missing description", params: QuoteLineParams{QuoteID: "quo_123", ServiceAgreementID: "sa_123", QuantityMin: 60, UnitRate: Money{Amount: 1000, Currency: "USD"}}, want: "description is required"},
		{name: "zero quantity", params: QuoteLineParams{QuoteID: "quo_123", ServiceAgreementID: "sa_123", Description: "Work", UnitRate: Money{Amount: 1000, Currency: "USD"}}, want: "quantity must be positive"},
		{name: "currency mismatch", params: QuoteLineParams{QuoteID: "quo_123", ServiceAgreementID: "sa_123", Description: "Work", QuantityMin: 60, UnitRate: Money{Amount: 1000, Currency: "EUR"}}, want: "must match quote currency"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := NewQuoteLine(tc.params, "USD")
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("NewQuoteLine() error = %v, want containing %q", err, tc.want)
			}
		})
	}
}

func TestQuoteLifecycleTransitions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		act  func(*Quote) error
		want QuoteStatus
	}{
		{name: "send draft", act: (*Quote).Send, want: QuoteStatusSent},
		{name: "reject draft", act: (*Quote).Reject, want: QuoteStatusRejected},
		{name: "expire draft", act: (*Quote).Expire, want: QuoteStatusExpired},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			quote := Quote{Status: QuoteStatusDraft}
			if err := tc.act(&quote); err != nil {
				t.Fatalf("transition error = %v", err)
			}
			if quote.Status != tc.want {
				t.Fatalf("Status = %q, want %q", quote.Status, tc.want)
			}
		})
	}
}

func TestQuoteTerminalLifecycleGuards(t *testing.T) {
	t.Parallel()

	quote := Quote{Status: QuoteStatusSent}
	if err := quote.Accept(); err != nil {
		t.Fatalf("Accept() error = %v", err)
	}
	if quote.Status != QuoteStatusAccepted {
		t.Fatalf("Status = %q, want accepted", quote.Status)
	}
	if err := quote.Reject(); err == nil {
		t.Fatal("Reject() error = nil, want terminal status error")
	}
}

func TestQuoteDeleteAndConversionEligibility(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status     QuoteStatus
		canDelete  bool
		canConvert bool
	}{
		{status: QuoteStatusDraft, canDelete: true},
		{status: QuoteStatusSent, canDelete: true},
		{status: QuoteStatusRejected, canDelete: true},
		{status: QuoteStatusExpired, canDelete: true},
		{status: QuoteStatusAccepted, canConvert: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(string(tc.status), func(t *testing.T) {
			t.Parallel()

			quote := Quote{Status: tc.status}
			if got := quote.CanDelete(); got != tc.canDelete {
				t.Fatalf("CanDelete() = %v, want %v", got, tc.canDelete)
			}
			if got := quote.CanConvertToInvoice(); got != tc.canConvert {
				t.Fatalf("CanConvertToInvoice() = %v, want %v", got, tc.canConvert)
			}
			if err := quote.ValidateDelete(); (err == nil) != tc.canDelete {
				t.Fatalf("ValidateDelete() error = %v, canDelete %v", err, tc.canDelete)
			} else if err != nil && !errors.Is(err, ErrAcceptedQuoteCannotBeDeleted) {
				t.Fatalf("ValidateDelete() error = %v, want ErrAcceptedQuoteCannotBeDeleted", err)
			}
		})
	}
}
