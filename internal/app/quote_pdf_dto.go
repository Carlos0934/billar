package app

type QuoteProposalDocumentDTO struct {
	QuoteID   string                         `json:"quote_id" toon:"quote_id"`
	Status    string                         `json:"status" toon:"status"`
	Currency  string                         `json:"currency" toon:"currency"`
	CreatedAt string                         `json:"created_at" toon:"created_at"`
	SentAt    string                         `json:"sent_at" toon:"sent_at"`
	Issuer    QuoteProposalDocumentPartyDTO  `json:"issuer" toon:"issuer"`
	Customer  QuoteProposalDocumentPartyDTO  `json:"customer" toon:"customer"`
	Lines     []QuoteProposalDocumentLineDTO `json:"lines" toon:"lines"`
	Total     int64                          `json:"total" toon:"total"`
	Notes     string                         `json:"notes" toon:"notes"`
}

type QuoteProposalDocumentPartyDTO struct {
	LegalName      string     `json:"legal_name" toon:"legal_name"`
	TradeName      string     `json:"trade_name" toon:"trade_name"`
	TaxID          string     `json:"tax_id" toon:"tax_id"`
	Email          string     `json:"email" toon:"email"`
	Phone          string     `json:"phone" toon:"phone"`
	Website        string     `json:"website" toon:"website"`
	BillingAddress AddressDTO `json:"billing_address" toon:"billing_address"`
}

type QuoteProposalDocumentLineDTO struct {
	Description       string `json:"description" toon:"description"`
	QuantityMin       int64  `json:"quantity_min" toon:"quantity_min"`
	UnitRateAmount    int64  `json:"unit_rate_amount" toon:"unit_rate_amount"`
	UnitRateCurrency  string `json:"unit_rate_currency" toon:"unit_rate_currency"`
	LineTotalAmount   int64  `json:"line_total_amount" toon:"line_total_amount"`
	LineTotalCurrency string `json:"line_total_currency" toon:"line_total_currency"`
}

type QuoteRenderedFileDTO struct {
	QuoteID   string `json:"quote_id" toon:"quote_id"`
	Filename  string `json:"filename" toon:"filename"`
	Path      string `json:"path" toon:"path"`
	MimeType  string `json:"mime_type" toon:"mime_type"`
	SizeBytes int64  `json:"size_bytes" toon:"size_bytes"`
}

type RenderQuotePDFCommand struct {
	QuoteID    string `json:"quote_id" toon:"quote_id"`
	Filename   string `json:"filename" toon:"filename"`
	OutputPath string `json:"output_path" toon:"output_path"`
}
