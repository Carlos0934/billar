package app

import (
	"fmt"
	"time"

	"github.com/Carlos0934/billar/internal/core"
)

type QuoteDTO struct {
	ID                  string         `json:"id" toon:"id"`
	CustomerID          string         `json:"customer_id" toon:"customer_id"`
	Status              string         `json:"status" toon:"status"`
	Currency            string         `json:"currency" toon:"currency"`
	Notes               string         `json:"notes" toon:"notes"`
	Lines               []QuoteLineDTO `json:"lines" toon:"lines"`
	Total               int64          `json:"total" toon:"total"`
	CanDelete           bool           `json:"can_delete" toon:"can_delete"`
	CanConvertToInvoice bool           `json:"can_convert_to_invoice" toon:"can_convert_to_invoice"`
	CreatedAt           string         `json:"created_at" toon:"created_at"`
	UpdatedAt           string         `json:"updated_at" toon:"updated_at"`
}

type QuoteLineDTO struct {
	ID                 string `json:"id" toon:"id"`
	QuoteID            string `json:"quote_id" toon:"quote_id"`
	ServiceAgreementID string `json:"service_agreement_id" toon:"service_agreement_id"`
	Description        string `json:"description" toon:"description"`
	QuantityMin        int64  `json:"quantity_min" toon:"quantity_min"`
	UnitRateAmount     int64  `json:"unit_rate_amount" toon:"unit_rate_amount"`
	UnitRateCurrency   string `json:"unit_rate_currency" toon:"unit_rate_currency"`
	LineTotalAmount    int64  `json:"line_total_amount" toon:"line_total_amount"`
	LineTotalCurrency  string `json:"line_total_currency" toon:"line_total_currency"`
}

type QuoteSummaryDTO struct {
	ID         string `json:"id" toon:"id"`
	CustomerID string `json:"customer_id" toon:"customer_id"`
	Status     string `json:"status" toon:"status"`
	Currency   string `json:"currency" toon:"currency"`
	Total      int64  `json:"total" toon:"total"`
	CreatedAt  string `json:"created_at" toon:"created_at"`
}

type CreateQuoteCommand struct {
	CustomerID string `json:"customer_id"`
	Currency   string `json:"currency"`
	Notes      string `json:"notes,omitempty"`
}

type ListQuotesCommand struct {
	CustomerID string `json:"customer_id"`
	Status     string `json:"status,omitempty"`
}

type AddQuoteLineCommand struct {
	QuoteID            string `json:"quote_id"`
	ServiceAgreementID string `json:"service_agreement_id"`
	Description        string `json:"description"`
	QuantityMin        int64  `json:"quantity_min"`
}

func quoteToDTO(quote core.Quote) QuoteDTO {
	dto := QuoteDTO{
		ID:                  quote.ID,
		CustomerID:          quote.CustomerID,
		Status:              string(quote.Status),
		Currency:            quote.Currency,
		Notes:               quote.Notes,
		CanDelete:           quote.CanDelete(),
		CanConvertToInvoice: quote.CanConvertToInvoice(),
		CreatedAt:           formatQuoteTime(quote.CreatedAt),
		UpdatedAt:           formatQuoteTime(quote.UpdatedAt),
	}
	for _, line := range quote.Lines {
		lineDTO := quoteLineToDTO(line)
		dto.Lines = append(dto.Lines, lineDTO)
		dto.Total += lineDTO.LineTotalAmount
	}
	if len(quote.Lines) == 0 {
		dto.Total = quote.Total().Amount
	}
	return dto
}

func quoteLineToDTO(line core.QuoteLine) QuoteLineDTO {
	total := line.LineTotal()
	return QuoteLineDTO{
		ID:                 line.ID,
		QuoteID:            line.QuoteID,
		ServiceAgreementID: line.ServiceAgreementID,
		Description:        line.Description,
		QuantityMin:        line.QuantityMin,
		UnitRateAmount:     line.UnitRate.Amount,
		UnitRateCurrency:   line.UnitRate.Currency,
		LineTotalAmount:    total.Amount,
		LineTotalCurrency:  total.Currency,
	}
}

func quoteSummaryToDTO(summary core.QuoteSummary) QuoteSummaryDTO {
	return QuoteSummaryDTO{
		ID:         summary.ID,
		CustomerID: summary.CustomerID,
		Status:     string(summary.Status),
		Currency:   summary.Currency,
		Total:      summary.Total.Amount,
		CreatedAt:  formatQuoteTime(summary.CreatedAt),
	}
}

func formatQuoteTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return fmt.Sprintf("%s", t.UTC().Format(time.RFC3339))
}
