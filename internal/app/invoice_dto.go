package app

import (
	"fmt"
	"time"

	"github.com/Carlos0934/billar/internal/core"
)

type InvoiceDTO struct {
	ID            string           `json:"id" toon:"id"`
	InvoiceNumber string           `json:"invoice_number" toon:"invoice_number"`
	CustomerID    string           `json:"customer_id" toon:"customer_id"`
	Status        string           `json:"status" toon:"status"`
	Currency      string           `json:"currency" toon:"currency"`
	IsDraft       bool             `json:"is_draft" toon:"is_draft"`
	IsIssued      bool             `json:"is_issued" toon:"is_issued"`
	IsDiscarded   bool             `json:"is_discarded" toon:"is_discarded"`
	Lines         []InvoiceLineDTO `json:"lines" toon:"lines"`
	Subtotal      int64            `json:"subtotal" toon:"subtotal"`
	GrandTotal    int64            `json:"grand_total" toon:"grand_total"`
	IssuedAt      string           `json:"issued_at" toon:"issued_at"`
	DiscardedAt   string           `json:"discarded_at" toon:"discarded_at"`
	CreatedAt     string           `json:"created_at" toon:"created_at"`
	UpdatedAt     string           `json:"updated_at" toon:"updated_at"`
}

type InvoiceLineDTO struct {
	ID                 string `json:"id" toon:"id"`
	InvoiceID          string `json:"invoice_id" toon:"invoice_id"`
	ServiceAgreementID string `json:"service_agreement_id" toon:"service_agreement_id"`
	TimeEntryID        string `json:"time_entry_id" toon:"time_entry_id"`
	Description        string `json:"description" toon:"description"`
	QuantityMin        int64  `json:"quantity_min" toon:"quantity_min"`
	UnitRateAmount     int64  `json:"unit_rate_amount" toon:"unit_rate_amount"`
	UnitRateCurrency   string `json:"unit_rate_currency" toon:"unit_rate_currency"`
	LineTotalAmount    int64  `json:"line_total_amount" toon:"line_total_amount"`
	LineTotalCurrency  string `json:"line_total_currency" toon:"line_total_currency"`
}

type InvoiceSummaryDTO struct {
	ID            string `json:"id" toon:"id"`
	InvoiceNumber string `json:"invoice_number" toon:"invoice_number"`
	CustomerID    string `json:"customer_id" toon:"customer_id"`
	Status        string `json:"status" toon:"status"`
	Currency      string `json:"currency" toon:"currency"`
	GrandTotal    int64  `json:"grand_total" toon:"grand_total"`
	CreatedAt     string `json:"created_at" toon:"created_at"`
}

type CreateDraftFromUnbilledCommand struct {
	CustomerProfileID string `json:"customer_profile_id"`
}

type IssueInvoiceCommand struct {
	InvoiceID string `json:"invoice_id"`
}

type DiscardInvoiceCommand struct {
	InvoiceID string `json:"invoice_id"`
}

// DiscardResult captures the outcome of a discard operation so connectors can
// show appropriate messages (e.g. soft-discard warning for issued invoices).
type DiscardResult struct {
	WasSoftDiscard bool
	InvoiceNumber  string
	Invoice        InvoiceDTO
}

func invoiceToDTO(inv core.Invoice, entries []core.TimeEntry) InvoiceDTO {
	lineMap := make(map[string]core.TimeEntry, len(entries))
	for _, entry := range entries {
		lineMap[entry.ID] = entry
	}

	dto := InvoiceDTO{
		ID:            inv.ID,
		InvoiceNumber: inv.InvoiceNumber,
		CustomerID:    inv.CustomerID,
		Status:        string(inv.Status),
		Currency:      inv.Currency,
		IsDraft:       inv.IsDraft(),
		IsIssued:      inv.IsIssued(),
		IsDiscarded:   inv.IsDiscarded(),
		IssuedAt:      formatInvoiceTime(inv.IssuedAt),
		DiscardedAt:   formatInvoiceTime(inv.DiscardedAt),
		CreatedAt:     formatInvoiceTime(inv.CreatedAt),
		UpdatedAt:     formatInvoiceTime(inv.UpdatedAt),
	}
	for _, line := range inv.Lines {
		entry := lineMap[line.TimeEntryID]
		lineDTO := invoiceLineToDTO(line, entry)
		dto.Lines = append(dto.Lines, lineDTO)
		dto.Subtotal += lineDTO.LineTotalAmount
	}
	dto.GrandTotal = dto.Subtotal
	return dto
}

func invoiceLineToDTO(line core.InvoiceLine, entry core.TimeEntry) InvoiceLineDTO {
	total := line.LineTotal(entry)
	return InvoiceLineDTO{
		ID:                 line.ID,
		InvoiceID:          line.InvoiceID,
		ServiceAgreementID: line.ServiceAgreementID,
		TimeEntryID:        line.TimeEntryID,
		Description:        entry.Description,
		QuantityMin:        int64(entry.Hours) * 60 / 10000,
		UnitRateAmount:     line.UnitRate.Amount,
		UnitRateCurrency:   line.UnitRate.Currency,
		LineTotalAmount:    total.Amount,
		LineTotalCurrency:  total.Currency,
	}
}

func formatInvoiceTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return fmt.Sprintf("%s", t.UTC().Format(time.RFC3339))
}
