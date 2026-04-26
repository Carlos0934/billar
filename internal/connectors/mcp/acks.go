package mcp

import "github.com/Carlos0934/billar/internal/app"

type DeleteAck struct {
	ID     string `json:"id" toon:"id"`
	Action string `json:"action" toon:"action"`
	Status string `json:"status" toon:"status"`
}

type InvoiceDiscardAck struct {
	ID             string         `json:"id" toon:"id"`
	Action         string         `json:"action" toon:"action"`
	WasSoftDiscard bool           `json:"was_soft_discard" toon:"was_soft_discard"`
	InvoiceNumber  string         `json:"invoice_number,omitempty" toon:"invoice_number"`
	Invoice        app.InvoiceDTO `json:"invoice" toon:"invoice"`
}

func newDeleteAck(id string) DeleteAck {
	return DeleteAck{ID: id, Action: "delete", Status: "ok"}
}

func newInvoiceDiscardAck(id string, result app.DiscardResult) InvoiceDiscardAck {
	action := "discarded"
	if result.WasSoftDiscard {
		action = "soft_discarded"
	}
	return InvoiceDiscardAck{
		ID:             id,
		Action:         action,
		WasSoftDiscard: result.WasSoftDiscard,
		InvoiceNumber:  result.InvoiceNumber,
		Invoice:        result.Invoice,
	}
}
