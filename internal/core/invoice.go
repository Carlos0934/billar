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
	Description        string
	QuantityMin        int64
	UnitRate           Money
	AmountMinor        int64
	TaxMinor           int64
	UnitPriceDisplay   string
	QuantityDisplay    string
}

type ImportInvoiceLineParams struct {
	Description      string
	AmountMinor      int64
	TaxMinor         int64
	QuantityDisplay  string
	UnitPriceDisplay string
	Currency         string
}

func NewImportedInvoiceLine(params ImportInvoiceLineParams) (InvoiceLine, error) {
	if strings.TrimSpace(params.Description) == "" {
		return InvoiceLine{}, errors.New("invoice line description is required")
	}
	if params.AmountMinor < 0 {
		return InvoiceLine{}, errors.New("invoice line amount_minor must be non-negative")
	}
	if params.TaxMinor < 0 {
		return InvoiceLine{}, errors.New("invoice line tax_minor must be non-negative")
	}
	if strings.TrimSpace(params.Currency) == "" {
		return InvoiceLine{}, errors.New("invoice line currency is required")
	}
	line := InvoiceLine{
		ID:               generateInvoiceLineID(),
		Description:      strings.TrimSpace(params.Description),
		UnitRate:         Money{Currency: strings.TrimSpace(params.Currency)},
		AmountMinor:      params.AmountMinor,
		TaxMinor:         params.TaxMinor,
		UnitPriceDisplay: strings.TrimSpace(params.UnitPriceDisplay),
		QuantityDisplay:  strings.TrimSpace(params.QuantityDisplay),
	}
	if line.ID == "" {
		return InvoiceLine{}, errors.New("failed to generate invoice line id")
	}
	return line, nil
}

type InvoiceLineParams struct {
	InvoiceID          string
	ServiceAgreementID string
	TimeEntryID        string
	Description        string
	QuantityMin        int64
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
	if params.QuantityMin < 0 {
		return InvoiceLine{}, errors.New("invoice line quantity must be non-negative")
	}

	line := InvoiceLine{
		ID:                 generateInvoiceLineID(),
		InvoiceID:          strings.TrimSpace(params.InvoiceID),
		ServiceAgreementID: strings.TrimSpace(params.ServiceAgreementID),
		TimeEntryID:        strings.TrimSpace(params.TimeEntryID),
		Description:        strings.TrimSpace(params.Description),
		QuantityMin:        params.QuantityMin,
		UnitRate:           params.UnitRate,
	}
	if line.ID == "" {
		return InvoiceLine{}, errors.New("failed to generate invoice line id")
	}
	return line, nil
}

func NewManualInvoiceLine(invoiceID, serviceAgreementID, description string, quantityMin int64, unitRate Money, invoiceCurrency string) (InvoiceLine, error) {
	if strings.TrimSpace(invoiceID) == "" {
		return InvoiceLine{}, errors.New("invoice line invoice id is required")
	}
	if strings.TrimSpace(serviceAgreementID) == "" {
		return InvoiceLine{}, errors.New("invoice line service agreement id is required")
	}
	if strings.TrimSpace(description) == "" {
		return InvoiceLine{}, errors.New("invoice line description is required")
	}
	if quantityMin <= 0 {
		return InvoiceLine{}, errors.New("invoice line quantity must be positive")
	}
	if !unitRate.IsPositive() {
		return InvoiceLine{}, errors.New("invoice line unit rate is required")
	}
	if strings.TrimSpace(invoiceCurrency) == "" {
		return InvoiceLine{}, errors.New("invoice currency is required")
	}
	if unitRate.Currency != strings.TrimSpace(invoiceCurrency) {
		return InvoiceLine{}, fmt.Errorf("invoice line currency %q must match invoice currency %q", unitRate.Currency, strings.TrimSpace(invoiceCurrency))
	}
	line := InvoiceLine{
		ID:                 generateInvoiceLineID(),
		InvoiceID:          strings.TrimSpace(invoiceID),
		ServiceAgreementID: strings.TrimSpace(serviceAgreementID),
		Description:        strings.TrimSpace(description),
		QuantityMin:        quantityMin,
		UnitRate:           unitRate,
	}
	if line.ID == "" {
		return InvoiceLine{}, errors.New("failed to generate invoice line id")
	}
	return line, nil
}

func (l InvoiceLine) LineTotal(entries ...TimeEntry) Money {
	if l.AmountMinor != 0 || (l.TimeEntryID == "" && l.ServiceAgreementID == "" && l.UnitRate.Amount == 0) {
		return Money{Amount: l.AmountMinor, Currency: l.UnitRate.Currency}
	}
	quantityMin := l.QuantityMin
	if quantityMin == 0 && len(entries) > 0 {
		quantityMin = int64(entries[0].Hours) * 60 / minutesPerHour
	}
	return Money{Amount: l.UnitRate.Amount * quantityMin / 60, Currency: l.UnitRate.Currency}
}

type Invoice struct {
	ID                   string
	InvoiceNumber        string
	CustomerID           string
	Status               InvoiceStatus
	Currency             string
	Lines                []InvoiceLine
	InvoiceDate          time.Time
	PeriodStart          time.Time
	PeriodEnd            time.Time
	DueDate              time.Time
	Notes                string
	PaymentTerms         string
	PaymentCommunication string
	ImportSource         string
	ExternalNumber       string
	ImportedAt           time.Time
	IssuedAt             time.Time
	DiscardedAt          time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type ImportInvoiceParams struct {
	CustomerID           string
	InvoiceNumber        string
	InvoiceDate          time.Time
	DueDate              time.Time
	Currency             string
	PaymentTerms         string
	PaymentCommunication string
	ImportSource         string
	ExternalNumber       string
	ImportedAt           time.Time
	Lines                []InvoiceLine
	SubtotalMinor        int64
	TaxTotalMinor        int64
	GrandTotalMinor      int64
}

func NewImportedInvoice(params ImportInvoiceParams) (Invoice, error) {
	if strings.TrimSpace(params.CustomerID) == "" {
		return Invoice{}, errors.New("invoice customer id is required")
	}
	if strings.TrimSpace(params.InvoiceNumber) == "" {
		return Invoice{}, errors.New("invoice number is required")
	}
	if params.InvoiceDate.IsZero() {
		return Invoice{}, errors.New("invoice_date is required")
	}
	if params.DueDate.IsZero() {
		return Invoice{}, errors.New("due_date is required")
	}
	if params.DueDate.Before(params.InvoiceDate) {
		return Invoice{}, errors.New("due_date must be on or after invoice_date")
	}
	if strings.TrimSpace(params.Currency) == "" {
		return Invoice{}, errors.New("invoice currency is required")
	}
	if len(params.Lines) == 0 {
		return Invoice{}, errors.New("invoice must have at least one line")
	}
	if params.GrandTotalMinor <= 0 {
		return Invoice{}, errors.New("grand_total_minor must be positive")
	}
	var subtotal, taxTotal int64
	for _, line := range params.Lines {
		if line.AmountMinor < 0 || line.TaxMinor < 0 {
			return Invoice{}, errors.New("invoice line amounts must be non-negative")
		}
		if line.UnitRate.Currency != strings.TrimSpace(params.Currency) {
			return Invoice{}, fmt.Errorf("invoice line currency %q must match invoice currency %q", line.UnitRate.Currency, strings.TrimSpace(params.Currency))
		}
		subtotal += line.AmountMinor
		taxTotal += line.TaxMinor
	}
	if subtotal != params.SubtotalMinor || taxTotal != params.TaxTotalMinor || subtotal+taxTotal != params.GrandTotalMinor {
		return Invoice{}, errors.New("totals do not balance")
	}
	now := params.ImportedAt.UTC()
	if now.IsZero() {
		now = params.InvoiceDate.UTC()
	}
	inv := Invoice{ID: generateInvoiceID(), InvoiceNumber: strings.TrimSpace(params.InvoiceNumber), CustomerID: strings.TrimSpace(params.CustomerID), Status: InvoiceStatusIssued, Currency: strings.TrimSpace(params.Currency), Lines: make([]InvoiceLine, len(params.Lines)), InvoiceDate: params.InvoiceDate.UTC(), PeriodStart: params.InvoiceDate.UTC(), PeriodEnd: params.InvoiceDate.UTC(), DueDate: params.DueDate.UTC(), PaymentTerms: strings.TrimSpace(params.PaymentTerms), PaymentCommunication: strings.TrimSpace(params.PaymentCommunication), ImportSource: strings.TrimSpace(params.ImportSource), ExternalNumber: strings.TrimSpace(params.ExternalNumber), ImportedAt: now, IssuedAt: params.InvoiceDate.UTC(), CreatedAt: params.InvoiceDate.UTC(), UpdatedAt: params.InvoiceDate.UTC()}
	if inv.ID == "" {
		return Invoice{}, errors.New("failed to generate invoice id")
	}
	for i, line := range params.Lines {
		line.InvoiceID = inv.ID
		inv.Lines[i] = line
	}
	return inv, nil
}

type InvoiceParams struct {
	CustomerID  string
	Status      InvoiceStatus
	Currency    string
	Lines       []InvoiceLine
	PeriodStart time.Time
	PeriodEnd   time.Time
	DueDate     time.Time
	Notes       string
	CreatedAt   time.Time
}

type InvoiceMetadataPatch struct {
	InvoiceDate          time.Time
	PeriodStart          time.Time
	PeriodEnd            time.Time
	DueDate              time.Time
	PaymentTerms         string
	PaymentCommunication string
	Notes                string
	ExternalNumber       string
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
	if !params.PeriodStart.IsZero() && !params.PeriodEnd.IsZero() && params.PeriodEnd.Before(params.PeriodStart) {
		return Invoice{}, errors.New("period_end must be on or after period_start")
	}
	if !params.DueDate.IsZero() && !params.PeriodEnd.IsZero() && params.DueDate.Before(params.PeriodEnd) {
		return Invoice{}, errors.New("due_date must be on or after period_end")
	}
	if !params.DueDate.IsZero() && params.PeriodEnd.IsZero() && !params.PeriodStart.IsZero() && params.DueDate.Before(params.PeriodStart) {
		return Invoice{}, errors.New("due_date must be on or after period_start")
	}
	if len(params.Notes) > 4000 {
		return Invoice{}, errors.New("invoice notes must be 4000 characters or fewer")
	}

	now := time.Now().UTC()
	inv := Invoice{
		ID:            generateInvoiceID(),
		InvoiceNumber: "",
		CustomerID:    strings.TrimSpace(params.CustomerID),
		Status:        params.Status,
		Currency:      strings.TrimSpace(params.Currency),
		Lines:         make([]InvoiceLine, len(params.Lines)),
		PeriodStart:   params.PeriodStart.UTC(),
		PeriodEnd:     params.PeriodEnd.UTC(),
		DueDate:       params.DueDate.UTC(),
		Notes:         params.Notes,
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

func (i *Invoice) UpdateMetadata(patch InvoiceMetadataPatch) error {
	if i == nil {
		return errors.New("invoice is required")
	}
	if i.IsDiscarded() {
		return errors.New("discarded invoices cannot be updated")
	}
	if !i.IsDraft() && !i.IsIssued() {
		return errors.New("invoice status is invalid")
	}

	updated := *i
	if !patch.InvoiceDate.IsZero() {
		updated.InvoiceDate = patch.InvoiceDate.UTC()
	}
	if !patch.PeriodStart.IsZero() {
		updated.PeriodStart = patch.PeriodStart.UTC()
	}
	if !patch.PeriodEnd.IsZero() {
		updated.PeriodEnd = patch.PeriodEnd.UTC()
	}
	if !patch.DueDate.IsZero() {
		updated.DueDate = patch.DueDate.UTC()
	}
	if patch.PaymentTerms != "" {
		updated.PaymentTerms = strings.TrimSpace(patch.PaymentTerms)
	}
	if patch.PaymentCommunication != "" {
		updated.PaymentCommunication = strings.TrimSpace(patch.PaymentCommunication)
	}
	if patch.Notes != "" {
		updated.Notes = patch.Notes
	}
	if patch.ExternalNumber != "" {
		updated.ExternalNumber = strings.TrimSpace(patch.ExternalNumber)
	}

	if !updated.PeriodStart.IsZero() && !updated.PeriodEnd.IsZero() && updated.PeriodEnd.Before(updated.PeriodStart) {
		return errors.New("period_end must be on or after period_start")
	}
	if !updated.DueDate.IsZero() && !updated.PeriodEnd.IsZero() && updated.DueDate.Before(updated.PeriodEnd) {
		return errors.New("due_date must be on or after period_end")
	}
	if !updated.DueDate.IsZero() && updated.PeriodEnd.IsZero() && !updated.PeriodStart.IsZero() && updated.DueDate.Before(updated.PeriodStart) {
		return errors.New("due_date must be on or after period_start")
	}
	if len(updated.Notes) > 4000 {
		return errors.New("invoice notes must be 4000 characters or fewer")
	}

	updated.UpdatedAt = time.Now().UTC()
	*i = updated
	return nil
}

func (i *Invoice) AddLine(line InvoiceLine) error {
	if i == nil {
		return errors.New("invoice is required")
	}
	if line.UnitRate.Currency != i.Currency {
		return fmt.Errorf("invoice line currency %q must match invoice currency %q", line.UnitRate.Currency, i.Currency)
	}
	line.InvoiceID = i.ID
	i.Lines = append(i.Lines, line)
	i.UpdatedAt = time.Now().UTC()
	return nil
}

func (i *Invoice) RemoveLine(lineID string) error {
	if i == nil {
		return errors.New("invoice is required")
	}
	lineID = strings.TrimSpace(lineID)
	if lineID == "" {
		return errors.New("invoice line id is required")
	}
	for idx, line := range i.Lines {
		if line.ID == lineID {
			if len(i.Lines) <= 1 {
				return errors.New("cannot remove last invoice line")
			}
			i.Lines = append(i.Lines[:idx], i.Lines[idx+1:]...)
			i.UpdatedAt = time.Now().UTC()
			return nil
		}
	}
	return errors.New("invoice line not found")
}

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
		if line.AmountMinor != 0 || (line.TimeEntryID == "" && line.ServiceAgreementID == "" && line.UnitRate.Amount == 0) {
			total.Amount += line.LineTotal().Amount
			continue
		}
		if line.QuantityMin == 0 {
			entry, ok := entryByID[line.TimeEntryID]
			if !ok {
				continue
			}
			total.Amount += line.LineTotal(entry).Amount
			continue
		}
		total.Amount += line.LineTotal().Amount
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
	PeriodStart   time.Time
	PeriodEnd     time.Time
	DueDate       time.Time
	CreatedAt     time.Time
}

func generateInvoiceID() string {
	return generatePrefixedID(invoiceIDPrefix, invoiceIDBytes)
}

func generateInvoiceLineID() string {
	return generatePrefixedID(invoiceLineIDPrefix, invoiceLineIDBytes)
}
