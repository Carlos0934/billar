package app

type UpdateInvoiceMetadataCommand struct {
	InvoiceID            string `json:"invoice_id"`
	InvoiceDate          string `json:"invoice_date,omitempty"`
	DueDate              string `json:"due_date,omitempty"`
	PaymentTerms         string `json:"payment_terms,omitempty"`
	PaymentCommunication string `json:"payment_communication,omitempty"`
}
