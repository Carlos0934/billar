package pdf

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/Carlos0934/billar/internal/app"
)

func TestRendererProducesSixByEightPDF(t *testing.T) {
	doc := fixtureInvoiceDocument(2)
	renderer := Renderer{}

	got, err := renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if !bytes.HasPrefix(got, []byte("%PDF-")) {
		t.Fatalf("Render() prefix = %q, want %%PDF-", string(got[:min(len(got), 5)]))
	}
	if len(got) < 1000 {
		t.Fatalf("Render() length = %d, want non-trivial PDF", len(got))
	}
	if !bytes.Contains(got, []byte("/MediaBox [0 0 432.00 576.00]")) && !bytes.Contains(got, []byte("/MediaBox [0 0 432 576]")) {
		t.Fatalf("Render() PDF does not declare 6x8in page MediaBox")
	}
	for _, want := range []string{"Period: 2026-04-01 - 2026-04-30", "Due: 2026-05-15"} {
		if !bytes.Contains(got, []byte(want)) {
			t.Fatalf("Render() PDF missing %q", want)
		}
	}
}

func TestRendererPaginatesManyLinesAndRepeatsHeader(t *testing.T) {
	doc := fixtureInvoiceDocument(42)
	renderer := Renderer{}

	got, err := renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if pages := bytes.Count(got, []byte("/Type /Page")); pages < 2 {
		t.Fatalf("Render() page markers = %d, want at least 2 pages", pages)
	}
	if headers := bytes.Count(got, []byte("Description")); headers < 2 {
		t.Fatalf("Render() table header count = %d, want repeated header", headers)
	}
}

func fixtureInvoiceDocument(lines int) app.InvoiceDocumentDTO {
	doc := app.InvoiceDocumentDTO{
		InvoiceID: "inv_123", InvoiceNumber: "INV-2026-0001", Status: "issued", Currency: "USD", PeriodStart: "2026-04-01T00:00:00Z", PeriodEnd: "2026-04-30T00:00:00Z", DueDate: "2026-05-15T00:00:00Z", CreatedAt: "2026-04-10T00:00:00Z", IssuedAt: "2026-04-11T00:00:00Z",
		Issuer:   app.InvoiceDocumentPartyDTO{LegalName: "Issuer Inc", TaxID: "I-123", Email: "issuer@example.test", BillingAddress: app.AddressDTO{Street: "Issuer St", City: "Santo Domingo", Country: "DO"}},
		Customer: app.InvoiceDocumentPartyDTO{LegalName: "Customer LLC", TaxID: "C-123", Email: "billing@example.test", BillingAddress: app.AddressDTO{Street: "Customer St", City: "Santo Domingo", Country: "DO"}},
	}
	for i := 0; i < lines; i++ {
		doc.Lines = append(doc.Lines, app.InvoiceDocumentLineDTO{Description: fmt.Sprintf("Consulting line %02d", i+1), QuantityMin: 60, UnitRateAmount: 10000, UnitRateCurrency: "USD", LineTotalAmount: 10000, LineTotalCurrency: "USD"})
		doc.Subtotal += 10000
	}
	doc.GrandTotal = doc.Subtotal
	doc.Notes = strings.Repeat("Thank you. ", 2)
	return doc
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
