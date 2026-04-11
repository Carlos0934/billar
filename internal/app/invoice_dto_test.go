package app

import (
	"testing"
	"time"

	"github.com/Carlos0934/billar/internal/core"
)

func TestInvoiceLineToDTO_DerivesFieldsFromLockedTimeEntry(t *testing.T) {
	t.Parallel()

	rate, err := core.NewMoney(10000, "USD")
	if err != nil {
		t.Fatalf("core.NewMoney(): %v", err)
	}
	hours, err := core.NewHours(15000)
	if err != nil {
		t.Fatalf("core.NewHours(): %v", err)
	}
	line := core.InvoiceLine{ID: "inl_123", InvoiceID: "inv_123", ServiceAgreementID: "sa_123", TimeEntryID: "te_123", UnitRate: rate}
	entry := core.TimeEntry{ID: "te_123", Description: "Work done", Hours: hours, Date: time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)}

	dto := invoiceLineToDTO(line, entry)
	if dto.Description != "Work done" {
		t.Fatalf("Description = %q, want Work done", dto.Description)
	}
	if dto.QuantityMin != 90 {
		t.Fatalf("QuantityMin = %d, want 90", dto.QuantityMin)
	}
	if dto.LineTotalAmount != 15000 {
		t.Fatalf("LineTotalAmount = %d, want 15000", dto.LineTotalAmount)
	}
}

func TestInvoiceToDTO_MapsHeaderAndLines(t *testing.T) {
	t.Parallel()

	rate, err := core.NewMoney(10000, "USD")
	if err != nil {
		t.Fatalf("core.NewMoney(): %v", err)
	}
	hours, err := core.NewHours(15000)
	if err != nil {
		t.Fatalf("core.NewHours(): %v", err)
	}
	entry := core.TimeEntry{ID: "te_123", Description: "Work done", Hours: hours, Date: time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)}
	line, err := core.NewInvoiceLine(core.InvoiceLineParams{InvoiceID: "inv_seed", ServiceAgreementID: "sa_123", TimeEntryID: entry.ID, UnitRate: rate})
	if err != nil {
		t.Fatalf("core.NewInvoiceLine(): %v", err)
	}
	invoice, err := core.NewInvoice(core.InvoiceParams{CustomerID: "cus_123", Status: core.InvoiceStatusDraft, Currency: "USD", Lines: []core.InvoiceLine{line}})
	if err != nil {
		t.Fatalf("core.NewInvoice(): %v", err)
	}

	dto := invoiceToDTO(invoice, []core.TimeEntry{entry})
	if dto.CustomerID != "cus_123" {
		t.Fatalf("CustomerID = %q, want cus_123", dto.CustomerID)
	}
	if dto.Status != string(core.InvoiceStatusDraft) {
		t.Fatalf("Status = %q, want draft", dto.Status)
	}
	if len(dto.Lines) != 1 {
		t.Fatalf("len(Lines) = %d, want 1", len(dto.Lines))
	}
	if dto.Lines[0].TimeEntryID != entry.ID {
		t.Fatalf("TimeEntryID = %q, want %q", dto.Lines[0].TimeEntryID, entry.ID)
	}
}
