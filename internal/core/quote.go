package core

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	QuoteStatusDraft    QuoteStatus = "draft"
	QuoteStatusSent     QuoteStatus = "sent"
	QuoteStatusAccepted QuoteStatus = "accepted"
	QuoteStatusRejected QuoteStatus = "rejected"
	QuoteStatusExpired  QuoteStatus = "expired"

	quoteIDPrefix     = "quo_"
	quoteIDBytes      = 16
	quoteLineIDPrefix = "qol_"
	quoteLineIDBytes  = 16
)

var ErrAcceptedQuoteCannotBeDeleted = errors.New("accepted quote cannot be deleted")

type QuoteStatus string

func (s QuoteStatus) IsValid() bool {
	switch s {
	case QuoteStatusDraft, QuoteStatusSent, QuoteStatusAccepted, QuoteStatusRejected, QuoteStatusExpired:
		return true
	default:
		return false
	}
}

type Quote struct {
	ID         string
	CustomerID string
	Status     QuoteStatus
	Currency   string
	Notes      string
	Lines      []QuoteLine
	SentAt     time.Time
	AcceptedAt time.Time
	RejectedAt time.Time
	ExpiredAt  time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type QuoteParams struct {
	CustomerID string
	Currency   string
	Notes      string
}

func NewQuote(params QuoteParams) (Quote, error) {
	if strings.TrimSpace(params.CustomerID) == "" {
		return Quote{}, errors.New("quote customer id is required")
	}
	if strings.TrimSpace(params.Currency) == "" {
		return Quote{}, errors.New("quote currency is required")
	}
	now := time.Now().UTC()
	quote := Quote{
		ID:         generateQuoteID(),
		CustomerID: strings.TrimSpace(params.CustomerID),
		Status:     QuoteStatusDraft,
		Currency:   strings.TrimSpace(params.Currency),
		Notes:      params.Notes,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if quote.ID == "" {
		return Quote{}, errors.New("failed to generate quote id")
	}
	return quote, nil
}

func (q Quote) Total() Money {
	total := Money{Currency: q.Currency}
	for _, line := range q.Lines {
		lineTotal := line.LineTotal()
		if total.Currency == "" {
			total.Currency = lineTotal.Currency
		}
		total.Amount += lineTotal.Amount
	}
	return total
}

func (q *Quote) Send() error {
	if q.Status != QuoteStatusDraft {
		return fmt.Errorf("send quote: quote status %q cannot be sent", q.Status)
	}
	q.Status = QuoteStatusSent
	q.SentAt = time.Now().UTC()
	q.UpdatedAt = q.SentAt
	return nil
}

func (q *Quote) Accept() error {
	if q.Status != QuoteStatusSent {
		return fmt.Errorf("accept quote: quote status %q cannot be accepted", q.Status)
	}
	q.Status = QuoteStatusAccepted
	q.AcceptedAt = time.Now().UTC()
	q.UpdatedAt = q.AcceptedAt
	return nil
}

func (q *Quote) Reject() error {
	if q.Status != QuoteStatusDraft && q.Status != QuoteStatusSent {
		return fmt.Errorf("reject quote: quote status %q cannot be rejected", q.Status)
	}
	q.Status = QuoteStatusRejected
	q.RejectedAt = time.Now().UTC()
	q.UpdatedAt = q.RejectedAt
	return nil
}

func (q *Quote) Expire() error {
	if q.Status != QuoteStatusDraft && q.Status != QuoteStatusSent {
		return fmt.Errorf("expire quote: quote status %q cannot be expired", q.Status)
	}
	q.Status = QuoteStatusExpired
	q.ExpiredAt = time.Now().UTC()
	q.UpdatedAt = q.ExpiredAt
	return nil
}

func (q Quote) CanDelete() bool {
	return q.Status != QuoteStatusAccepted
}

func (q Quote) ValidateDelete() error {
	if !q.CanDelete() {
		return ErrAcceptedQuoteCannotBeDeleted
	}
	return nil
}

// CanConvertToInvoice intentionally exposes eligibility only. The conversion
// use case and future invoice source-quote provenance belong to a later app
// slice so core does not couple quotes to invoice creation.
func (q Quote) CanConvertToInvoice() bool {
	return q.Status == QuoteStatusAccepted
}

type QuoteLine struct {
	ID                 string
	QuoteID            string
	ServiceAgreementID string
	Description        string
	QuantityMin        int64
	UnitRate           Money
}

type QuoteLineParams struct {
	QuoteID            string
	ServiceAgreementID string
	Description        string
	QuantityMin        int64
	UnitRate           Money
}

func NewQuoteLine(params QuoteLineParams, quoteCurrency string) (QuoteLine, error) {
	if strings.TrimSpace(params.QuoteID) == "" {
		return QuoteLine{}, errors.New("quote line quote id is required")
	}
	if strings.TrimSpace(params.ServiceAgreementID) == "" {
		return QuoteLine{}, errors.New("quote line service agreement id is required")
	}
	if strings.TrimSpace(params.Description) == "" {
		return QuoteLine{}, errors.New("quote line description is required")
	}
	if params.QuantityMin <= 0 {
		return QuoteLine{}, errors.New("quote line quantity must be positive")
	}
	if !params.UnitRate.IsPositive() {
		return QuoteLine{}, errors.New("quote line unit rate is required")
	}
	if strings.TrimSpace(quoteCurrency) == "" {
		return QuoteLine{}, errors.New("quote currency is required")
	}
	if params.UnitRate.Currency != strings.TrimSpace(quoteCurrency) {
		return QuoteLine{}, fmt.Errorf("quote line currency %q must match quote currency %q", params.UnitRate.Currency, strings.TrimSpace(quoteCurrency))
	}
	line := QuoteLine{
		ID:                 generateQuoteLineID(),
		QuoteID:            strings.TrimSpace(params.QuoteID),
		ServiceAgreementID: strings.TrimSpace(params.ServiceAgreementID),
		Description:        strings.TrimSpace(params.Description),
		QuantityMin:        params.QuantityMin,
		UnitRate:           params.UnitRate,
	}
	if line.ID == "" {
		return QuoteLine{}, errors.New("failed to generate quote line id")
	}
	return line, nil
}

func (l QuoteLine) LineTotal() Money {
	return Money{Amount: l.UnitRate.Amount * l.QuantityMin / 60, Currency: l.UnitRate.Currency}
}

type QuoteSummary struct {
	ID         string
	CustomerID string
	Status     QuoteStatus
	Currency   string
	Total      Money
	CreatedAt  time.Time
}

func generateQuoteID() string {
	return generatePrefixedID(quoteIDPrefix, quoteIDBytes)
}

func generateQuoteLineID() string {
	return generatePrefixedID(quoteLineIDPrefix, quoteLineIDBytes)
}
