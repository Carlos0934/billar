package app

type InvoiceDocumentDTO struct {
	InvoiceID     string                   `json:"invoice_id" toon:"invoice_id"`
	InvoiceNumber string                   `json:"invoice_number" toon:"invoice_number"`
	Status        string                   `json:"status" toon:"status"`
	Currency      string                   `json:"currency" toon:"currency"`
	PeriodStart   string                   `json:"period_start" toon:"period_start"`
	PeriodEnd     string                   `json:"period_end" toon:"period_end"`
	DueDate       string                   `json:"due_date" toon:"due_date"`
	IssuedAt      string                   `json:"issued_at" toon:"issued_at"`
	CreatedAt     string                   `json:"created_at" toon:"created_at"`
	Issuer        InvoiceDocumentPartyDTO  `json:"issuer" toon:"issuer"`
	Customer      InvoiceDocumentPartyDTO  `json:"customer" toon:"customer"`
	Lines         []InvoiceDocumentLineDTO `json:"lines" toon:"lines"`
	Subtotal      int64                    `json:"subtotal" toon:"subtotal"`
	GrandTotal    int64                    `json:"grand_total" toon:"grand_total"`
	Notes         string                   `json:"notes" toon:"notes"`
}

type InvoiceDocumentPartyDTO struct {
	LegalName      string     `json:"legal_name" toon:"legal_name"`
	TradeName      string     `json:"trade_name" toon:"trade_name"`
	TaxID          string     `json:"tax_id" toon:"tax_id"`
	Email          string     `json:"email" toon:"email"`
	Phone          string     `json:"phone" toon:"phone"`
	Website        string     `json:"website" toon:"website"`
	BillingAddress AddressDTO `json:"billing_address" toon:"billing_address"`
}

type InvoiceDocumentLineDTO struct {
	Description       string `json:"description" toon:"description"`
	QuantityMin       int64  `json:"quantity_min" toon:"quantity_min"`
	UnitRateAmount    int64  `json:"unit_rate_amount" toon:"unit_rate_amount"`
	UnitRateCurrency  string `json:"unit_rate_currency" toon:"unit_rate_currency"`
	LineTotalAmount   int64  `json:"line_total_amount" toon:"line_total_amount"`
	LineTotalCurrency string `json:"line_total_currency" toon:"line_total_currency"`
}

type RenderedFileDTO struct {
	InvoiceID string `json:"invoice_id" toon:"invoice_id"`
	Filename  string `json:"filename" toon:"filename"`
	Path      string `json:"path" toon:"path"`
	MimeType  string `json:"mime_type" toon:"mime_type"`
	SizeBytes int64  `json:"size_bytes" toon:"size_bytes"`
}

type RenderInvoicePDFCommand struct {
	InvoiceID  string `json:"invoice_id" toon:"invoice_id"`
	Filename   string `json:"filename" toon:"filename"`
	OutputPath string `json:"output_path" toon:"output_path"`
}
