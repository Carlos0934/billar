package app

import (
	"reflect"
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

func TestInvoiceToDTO_MapsOperationalMetadata(t *testing.T) {
	t.Parallel()

	line := core.InvoiceLine{ID: "inl_123", InvoiceID: "inv_123", Description: "Imported work", UnitRate: core.Money{Currency: "USD"}, AmountMinor: 250000}
	invoice := core.Invoice{
		ID:                   "inv_123",
		InvoiceNumber:        "INV-2026-0001",
		CustomerID:           "cus_123",
		Status:               core.InvoiceStatusIssued,
		Currency:             "USD",
		Lines:                []core.InvoiceLine{line},
		InvoiceDate:          time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		PeriodStart:          time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		PeriodEnd:            time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC),
		DueDate:              time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC),
		PaymentTerms:         "Net 15",
		PaymentCommunication: "Use invoice INV-2026-0001",
		ExternalNumber:       "EXT-001",
		ImportSource:         "manual-pdf-extract",
		ImportedAt:           time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC),
	}

	dto := invoiceToDTO(invoice, nil)
	if dto.InvoiceDate != "2026-05-01T00:00:00Z" || dto.PaymentTerms != "Net 15" || dto.PaymentCommunication != "Use invoice INV-2026-0001" {
		t.Fatalf("dto operational metadata = (%q,%q,%q), want invoice date and payment metadata", dto.InvoiceDate, dto.PaymentTerms, dto.PaymentCommunication)
	}
	if dto.ExternalNumber != "EXT-001" || dto.ImportSource != "manual-pdf-extract" || dto.ImportedAt != "2026-05-01T12:00:00Z" {
		t.Fatalf("dto import metadata = (%q,%q,%q), want imported identity", dto.ExternalNumber, dto.ImportSource, dto.ImportedAt)
	}
}

func TestInvoiceDTOOperationalMetadataTags(t *testing.T) {
	t.Parallel()

	typ := reflect.TypeOf(InvoiceDTO{})
	fields := map[string]string{
		"InvoiceDate":          "invoice_date",
		"PaymentTerms":         "payment_terms",
		"PaymentCommunication": "payment_communication",
		"ExternalNumber":       "external_number",
		"ImportSource":         "import_source",
		"ImportedAt":           "imported_at",
	}
	for fieldName, tagName := range fields {
		field, ok := typ.FieldByName(fieldName)
		if !ok {
			t.Fatalf("InvoiceDTO missing field %s", fieldName)
		}
		if got := field.Tag.Get("json"); got != tagName {
			t.Fatalf("%s json tag = %q, want %q", fieldName, got, tagName)
		}
		if got := field.Tag.Get("toon"); got != tagName {
			t.Fatalf("%s toon tag = %q, want %q", fieldName, got, tagName)
		}
	}
}
