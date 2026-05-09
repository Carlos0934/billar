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
	for _, want := range []string{"INVOICE DATE", "04/11/2026", "DUE DATE", "05/15/2026"} {
		if !bytes.Contains(got, []byte(want)) {
			t.Fatalf("Render() PDF missing %q", want)
		}
	}
}

func TestRendererFormatsImportedInvoiceLikePolishedSummary(t *testing.T) {
	doc := fixtureInvoiceDocument(0)
	doc.InvoiceNumber = "INV/2026/00001"
	doc.PaymentComm = "INV/2026/00001"
	doc.Customer.BillingAddress.City = "Cancún"
	doc.Lines = []app.InvoiceDocumentLineDTO{{Description: "Resolve and implement enhancement to improve billing PDF rendering with enough detail to wrap cleanly", QuantityDisplay: "160.00", UnitPriceDisplay: "15.6250", LineTotalAmount: 250000, LineTotalCurrency: "USD"}}
	doc.Subtotal = 250000
	doc.GrandTotal = 250000

	got, err := (Renderer{}).Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	for _, want := range []string{"WORK DESCRIPTION", "HOURS", "160.00", "UNIT PRICE", "USD$ 15.6250", "TAXES", "AMOUNT", "USD$ 2,500.00", "PAYMENT COMMUNICATION"} {
		if !bytes.Contains(got, []byte(want)) {
			t.Fatalf("Render() PDF missing %q", want)
		}
	}
	for _, notWant := range []string{"QTY / HOURS", "USD 250000", "USD$ 250000", "Resolve and implement enhancement…", "CancÃºn"} {
		if bytes.Contains(got, []byte(notWant)) {
			t.Fatalf("Render() PDF contains %q", notWant)
		}
	}
}

func TestRendererMatchesReferenceDesignTextContract(t *testing.T) {
	doc := fixtureInvoiceDocument(0)
	doc.InvoiceNumber = "INV/2026/00003"
	doc.PaymentComm = "INV/2026/00003"
	doc.InvoiceDate = "2026-03-02T00:00:00Z"
	doc.DueDate = "2026-03-16T00:00:00Z"
	doc.Customer.BillingAddress.City = "Cancún"
	doc.Lines = []app.InvoiceDocumentLineDTO{{Description: "Resolve and implement enhancement to improve billing PDF rendering with enough detail to wrap cleanly instead of being truncated in a table row", QuantityDisplay: "160.00", UnitPriceDisplay: "15.6250", LineTotalAmount: 250000, LineTotalCurrency: "USD"}}
	doc.Subtotal = 250000
	doc.GrandTotal = 250000

	got, err := (Renderer{}).Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	for _, want := range []string{
		"Invoice INV/2026/00003",
		"INVOICE DATE", "03/02/2026",
		"DUE DATE", "03/16/2026",
		"PAYMENT COMMUNICATION",
		"BILL TO", "FROM",
		"WORK DESCRIPTION",
		"HOURS", "160.00",
		"UNIT PRICE", "USD$ 15.6250",
		"TAXES", "USD$ 0.00",
		"AMOUNT", "USD$ 2,500.00",
		"TOTAL",
		"Cancun",
	} {
		if !bytes.Contains(got, []byte(want)) {
			t.Fatalf("Render() PDF missing %q", want)
		}
	}
	for _, notWant := range []string{"QTY / HOURS", "Description Qty Rate Total", "Status:", "USD 250000", "USD$ 250000", "250000", "CancÃºn"} {
		if bytes.Contains(got, []byte(notWant)) {
			t.Fatalf("Render() PDF contains %q", notWant)
		}
	}
}

func TestMoneyFormatsMinorUnits(t *testing.T) {
	for _, tc := range []struct {
		amount   int64
		currency string
		want     string
	}{
		{250000, "USD", "USD$ 2,500.00"},
		{0, "USD", "USD$ 0.00"},
		{-12345, "EUR", "EUR€ -123.45"},
	} {
		if got := money(tc.amount, tc.currency); got != tc.want {
			t.Fatalf("money(%d, %q) = %q, want %q", tc.amount, tc.currency, got, tc.want)
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
	if headers := bytes.Count(got, []byte("WORK DESCRIPTION")); headers < 2 {
		t.Fatalf("Render() work description header count = %d, want repeated header", headers)
	}
}

func TestRendererRendersQuoteProposalVocabularyPartiesLinesAndTotals(t *testing.T) {
	doc := fixtureQuoteProposalDocument(2)

	got, err := (Renderer{}).RenderQuoteProposal(context.Background(), doc)
	if err != nil {
		t.Fatalf("RenderQuoteProposal() error = %v", err)
	}
	if !bytes.HasPrefix(got, []byte("%PDF-")) {
		t.Fatalf("RenderQuoteProposal() prefix = %q, want %%PDF-", string(got[:min(len(got), 5)]))
	}
	if len(got) < 1000 {
		t.Fatalf("RenderQuoteProposal() length = %d, want non-trivial PDF", len(got))
	}

	for _, want := range []string{
		"QUOTE PROPOSAL",
		"Quote quo_123",
		"QUOTE DATE", "04/10/2026",
		"VALIDITY", "SENT 04/11/2026",
		"PROPOSED TO", "FROM",
		"Customer LLC", "Issuer Inc",
		"Implementation workshop", "Support retainer",
		"ESTIMATED HOURS", "2", "5",
		"UNIT RATE", "USD$ 100.00", "USD$ 200.00",
		"PROPOSAL TOTAL", "USD$ 1,200.00",
	} {
		if !bytes.Contains(got, []byte(want)) {
			t.Fatalf("RenderQuoteProposal() PDF missing %q", want)
		}
	}
	for _, notWant := range []string{"INVOICE", "BILL TO", "PAYMENT COMMUNICATION", "DUE DATE", "USD 120000", "USD$ 120000"} {
		if bytes.Contains(got, []byte(notWant)) {
			t.Fatalf("RenderQuoteProposal() PDF contains %q", notWant)
		}
	}
}

func TestRendererRendersQuoteProposalUsingLineDerivedTotal(t *testing.T) {
	doc := fixtureQuoteProposalDocument(0)
	doc.Total = 999999
	doc.Lines = []app.QuoteProposalDocumentLineDTO{
		{Description: "Discovery sprint", QuantityMin: 90, UnitRateAmount: 10000, UnitRateCurrency: "USD", LineTotalAmount: 15000, LineTotalCurrency: "USD"},
		{Description: "Architecture plan", QuantityMin: 120, UnitRateAmount: 12500, UnitRateCurrency: "USD", LineTotalAmount: 25000, LineTotalCurrency: "USD"},
	}

	got, err := (Renderer{}).RenderQuoteProposal(context.Background(), doc)
	if err != nil {
		t.Fatalf("RenderQuoteProposal() error = %v", err)
	}

	for _, want := range []string{"Discovery sprint", "Architecture plan", "USD$ 400.00"} {
		if !bytes.Contains(got, []byte(want)) {
			t.Fatalf("RenderQuoteProposal() PDF missing %q", want)
		}
	}
	for _, notWant := range []string{"USD$ 9,999.99", "INVOICE"} {
		if bytes.Contains(got, []byte(notWant)) {
			t.Fatalf("RenderQuoteProposal() PDF contains %q", notWant)
		}
	}
}

func fixtureInvoiceDocument(lines int) app.InvoiceDocumentDTO {
	doc := app.InvoiceDocumentDTO{
		InvoiceID: "inv_123", InvoiceNumber: "INV-2026-0001", Status: "issued", Currency: "USD", PeriodStart: "2026-04-01T00:00:00Z", PeriodEnd: "2026-04-30T00:00:00Z", DueDate: "2026-05-15T00:00:00Z", InvoiceDate: "2026-04-11T00:00:00Z", CreatedAt: "2026-04-10T00:00:00Z", IssuedAt: "2026-04-11T00:00:00Z",
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

func fixtureQuoteProposalDocument(lines int) app.QuoteProposalDocumentDTO {
	doc := app.QuoteProposalDocumentDTO{
		QuoteID:   "quo_123",
		Status:    "sent",
		Currency:  "USD",
		CreatedAt: "2026-04-10T00:00:00Z",
		SentAt:    "2026-04-11T00:00:00Z",
		Issuer:    app.QuoteProposalDocumentPartyDTO{LegalName: "Issuer Inc", TaxID: "I-123", Email: "issuer@example.test", Website: "https://issuer.example", BillingAddress: app.AddressDTO{Street: "Issuer St", City: "Santo Domingo", Country: "DO"}},
		Customer:  app.QuoteProposalDocumentPartyDTO{LegalName: "Customer LLC", TaxID: "C-123", Email: "billing@example.test", BillingAddress: app.AddressDTO{Street: "Customer St", City: "Santo Domingo", Country: "DO"}},
		Notes:     "This proposal is valid for 30 days.",
	}
	fixtures := []app.QuoteProposalDocumentLineDTO{
		{Description: "Implementation workshop", QuantityMin: 120, UnitRateAmount: 10000, UnitRateCurrency: "USD", LineTotalAmount: 20000, LineTotalCurrency: "USD"},
		{Description: "Support retainer", QuantityMin: 300, UnitRateAmount: 20000, UnitRateCurrency: "USD", LineTotalAmount: 100000, LineTotalCurrency: "USD"},
	}
	for i := 0; i < lines && i < len(fixtures); i++ {
		doc.Lines = append(doc.Lines, fixtures[i])
		doc.Total += fixtures[i].LineTotalAmount
	}
	return doc
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
