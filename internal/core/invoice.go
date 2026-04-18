package core

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	InvoiceStatusDraft     InvoiceStatus = "draft"
	InvoiceStatusIssued    InvoiceStatus = "issued"
	InvoiceStatusDiscarded InvoiceStatus = "discarded"

	invoiceIDPrefix     = "inv_"
	invoiceIDBytes      = 16
	invoiceLineIDPrefix = "inl_"
	invoiceLineIDBytes  = 16
	minutesPerHour      = int64(10000)
)

type InvoiceStatus string

func (s InvoiceStatus) IsValid() bool {
	switch s {
	case InvoiceStatusDraft, InvoiceStatusIssued, InvoiceStatusDiscarded:
		return true
	default:
		return false
	}
}

type InvoiceLine struct {
	ID                 string
	InvoiceID          string
	ServiceAgreementID string
	TimeEntryID        string
	UnitRate           Money
}

type InvoiceLineParams struct {
	InvoiceID          string
	ServiceAgreementID string
	TimeEntryID        string
	UnitRate           Money
}

func NewInvoiceLine(params InvoiceLineParams) (InvoiceLine, error) {
	if strings.TrimSpace(params.InvoiceID) == "" {
		return InvoiceLine{}, errors.New("invoice line invoice id is required")
	}
	if strings.TrimSpace(params.ServiceAgreementID) == "" {
		return InvoiceLine{}, errors.New("invoice line service agreement id is required")
	}
	if strings.TrimSpace(params.TimeEntryID) == "" {
		return InvoiceLine{}, errors.New("invoice line time entry id is required")
	}
	if !params.UnitRate.IsPositive() {
		return InvoiceLine{}, errors.New("invoice line unit rate is required")
	}

	line := InvoiceLine{
		ID:                 generateInvoiceLineID(),
		InvoiceID:          strings.TrimSpace(params.InvoiceID),
		ServiceAgreementID: strings.TrimSpace(params.ServiceAgreementID),
		TimeEntryID:        strings.TrimSpace(params.TimeEntryID),
		UnitRate:           params.UnitRate,
	}
	if line.ID == "" {
		return InvoiceLine{}, errors.New("failed to generate invoice line id")
	}
	return line, nil
}

func (l InvoiceLine) LineTotal(entry TimeEntry) Money {
	return Money{Amount: l.UnitRate.Amount * int64(entry.Hours) / minutesPerHour, Currency: l.UnitRate.Currency}
}

type Invoice struct {
	ID            string
	InvoiceNumber string
	CustomerID    string
	Status        InvoiceStatus
	Currency      string
	Lines         []InvoiceLine
	IssuedAt      time.Time
	DiscardedAt   time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type InvoiceParams struct {
	CustomerID string
	Status     InvoiceStatus
	Currency   string
	Lines      []InvoiceLine
	CreatedAt  time.Time
}

func NewInvoice(params InvoiceParams) (Invoice, error) {
	if strings.TrimSpace(params.CustomerID) == "" {
		return Invoice{}, errors.New("invoice customer id is required")
	}
	if strings.TrimSpace(params.Currency) == "" {
		return Invoice{}, errors.New("invoice currency is required")
	}
	if !params.Status.IsValid() {
		return Invoice{}, errors.New("invoice status is invalid")
	}
	if len(params.Lines) == 0 {
		return Invoice{}, errors.New("invoice must have at least one line")
	}

	now := time.Now().UTC()
	inv := Invoice{
		ID:            generateInvoiceID(),
		InvoiceNumber: "",
		CustomerID:    strings.TrimSpace(params.CustomerID),
		Status:        params.Status,
		Currency:      strings.TrimSpace(params.Currency),
		Lines:         make([]InvoiceLine, len(params.Lines)),
		IssuedAt:      time.Time{},
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if !params.CreatedAt.IsZero() {
		inv.CreatedAt = params.CreatedAt.UTC()
		inv.UpdatedAt = params.CreatedAt.UTC()
	}
	if inv.ID == "" {
		return Invoice{}, errors.New("failed to generate invoice id")
	}

	seenCurrency := ""
	for i, line := range params.Lines {
		if line.UnitRate.Currency != inv.Currency {
			return Invoice{}, fmt.Errorf("invoice line currency %q must match invoice currency %q", line.UnitRate.Currency, inv.Currency)
		}
		if seenCurrency == "" {
			seenCurrency = line.UnitRate.Currency
		}
		if line.UnitRate.Currency != seenCurrency {
			return Invoice{}, errors.New("invoice lines must share the same currency")
		}
		line.InvoiceID = inv.ID
		inv.Lines[i] = line
	}

	return inv, nil
}

func (i Invoice) IsDraft() bool { return i.Status == InvoiceStatusDraft }

func (i Invoice) IsIssued() bool { return i.Status == InvoiceStatusIssued }

func (i Invoice) IsDiscarded() bool { return i.Status == InvoiceStatusDiscarded }

func (i *Invoice) Discard(now time.Time) error {
	if i == nil {
		return errors.New("invoice is required")
	}
	if i.IsDraft() {
		return errors.New("draft invoices must be hard-deleted")
	}
	if i.IsDiscarded() {
		return errors.New("invoice is already discarded")
	}
	if now.IsZero() {
		return errors.New("discard timestamp is required")
	}
	i.Status = InvoiceStatusDiscarded
	i.DiscardedAt = now.UTC()
	i.UpdatedAt = now.UTC()
	return nil
}

func (i *Invoice) Issue(number string, issuedAt time.Time) error {
	if i == nil {
		return errors.New("invoice is required")
	}
	if strings.TrimSpace(number) == "" {
		return errors.New("invoice number is required")
	}
	if issuedAt.IsZero() {
		return errors.New("invoice issued at is required")
	}
	if !i.IsDraft() {
		return errors.New("invoice is not draft")
	}

	i.InvoiceNumber = strings.TrimSpace(number)
	i.Status = InvoiceStatusIssued
	i.IssuedAt = issuedAt.UTC()
	i.UpdatedAt = issuedAt.UTC()
	return nil
}

func (i Invoice) Total(entries []TimeEntry) Money {
	total := Money{Currency: i.Currency}
	entryByID := make(map[string]TimeEntry, len(entries))
	for _, entry := range entries {
		entryByID[entry.ID] = entry
	}
	for _, line := range i.Lines {
		entry, ok := entryByID[line.TimeEntryID]
		if !ok {
			continue
		}
		total.Amount += line.LineTotal(entry).Amount
	}
	return total
}

// InvoiceSummary is a lightweight view of an invoice for list operations
// (no line items, grand_total computed by the store).
type InvoiceSummary struct {
	ID            string
	InvoiceNumber string
	CustomerID    string
	Status        InvoiceStatus
	Currency      string
	GrandTotal    int64
	CreatedAt     time.Time
}

func generateInvoiceID() string {
	return generatePrefixedID(invoiceIDPrefix, invoiceIDBytes)
}

func generateInvoiceLineID() string {
	return generatePrefixedID(invoiceLineIDPrefix, invoiceLineIDBytes)
}
