package app

type ImportIssuedInvoiceCommand struct {
	Payload ImportPayload `json:"payload"`
}

type ImportPayload struct {
	Schema               string                `json:"schema"`
	InvoiceNumber        string                `json:"invoice_number"`
	InvoiceDate          string                `json:"invoice_date"`
	DueDate              string                `json:"due_date"`
	Currency             string                `json:"currency"`
	PaymentTerms         string                `json:"payment_terms"`
	PaymentCommunication string                `json:"payment_communication"`
	Customer             ImportPayloadCustomer `json:"customer"`
	Issuer               ImportPayloadIssuer   `json:"issuer"`
	Lines                []ImportPayloadLine   `json:"lines"`
	Totals               ImportPayloadTotals   `json:"totals"`
	Source               ImportPayloadSource   `json:"source"`
}

type ImportPayloadCustomer struct {
	CustomerProfileID string         `json:"customer_profile_id"`
	LegalName         string         `json:"legal_name"`
	TradeName         string         `json:"trade_name"`
	TaxID             string         `json:"tax_id"`
	Email             string         `json:"email"`
	Phone             string         `json:"phone"`
	Website           string         `json:"website"`
	BillingAddress    map[string]any `json:"billing_address"`
}

type ImportPayloadIssuer struct {
	IssuerProfileID string         `json:"issuer_profile_id"`
	LegalName       string         `json:"legal_name"`
	TradeName       string         `json:"trade_name"`
	TaxID           string         `json:"tax_id"`
	Email           string         `json:"email"`
	Phone           string         `json:"phone"`
	Website         string         `json:"website"`
	BillingAddress  map[string]any `json:"billing_address"`
}

type ImportPayloadLine struct {
	Description        string `json:"description"`
	QuantityDisplay    string `json:"quantity_display"`
	UnitPriceDisplay   string `json:"unit_price_display"`
	AmountMinor        int64  `json:"amount_minor"`
	TaxMinor           int64  `json:"tax_minor"`
	ServiceAgreementID string `json:"service_agreement_id"`
}

type ImportPayloadTotals struct {
	SubtotalMinor   int64 `json:"subtotal_minor"`
	TaxTotalMinor   int64 `json:"tax_total_minor"`
	GrandTotalMinor int64 `json:"grand_total_minor"`
}

type ImportPayloadSource struct {
	System     string `json:"system"`
	ExternalID string `json:"external_id"`
	ImportedAt string `json:"imported_at"`
}
