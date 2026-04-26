package mcp

import (
	"encoding/json"
	"testing"

	"github.com/Carlos0934/billar/internal/app"
)

func TestNewDeleteAck(t *testing.T) {
	t.Parallel()

	ack := newDeleteAck("cus_1")
	if ack.ID != "cus_1" || ack.Action != "delete" || ack.Status != "ok" {
		t.Fatalf("ack = %+v, want cus_1/delete/ok", ack)
	}

	raw, err := json.Marshal(ack)
	if err != nil {
		t.Fatalf("marshal ack: %v", err)
	}
	if got, want := string(raw), `{"id":"cus_1","action":"delete","status":"ok"}`; got != want {
		t.Fatalf("json = %s, want %s", got, want)
	}
}

func TestNewInvoiceDiscardAck(t *testing.T) {
	t.Parallel()

	ack := newInvoiceDiscardAck("inv_42", app.DiscardResult{
		WasSoftDiscard: true,
		InvoiceNumber:  "INV-1",
		Invoice:        app.InvoiceDTO{ID: "inv_42", Status: "discarded"},
	})
	if ack.ID != "inv_42" || ack.Action != "soft_discarded" || !ack.WasSoftDiscard || ack.InvoiceNumber != "INV-1" || ack.Invoice.ID != "inv_42" {
		t.Fatalf("ack = %+v, want soft discard fields", ack)
	}

	raw, err := json.Marshal(ack)
	if err != nil {
		t.Fatalf("marshal ack: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("decode ack json: %v", err)
	}
	for _, key := range []string{"id", "action", "was_soft_discard", "invoice_number", "invoice"} {
		if _, ok := decoded[key]; !ok {
			t.Fatalf("json keys = %v, missing %q", decoded, key)
		}
	}

	hard := newInvoiceDiscardAck("inv_43", app.DiscardResult{Invoice: app.InvoiceDTO{ID: "inv_43"}})
	if hard.Action != "discarded" || hard.WasSoftDiscard {
		t.Fatalf("hard ack = %+v, want action discarded and WasSoftDiscard false", hard)
	}
}
