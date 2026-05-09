package app

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Carlos0934/billar/internal/core"
)

func TestQuotePDFServiceRenderQuotePDFHappyPath(t *testing.T) {
	ctx := context.Background()
	fx := newQuotePDFFixture(t)
	renderer := &stubQuoteProposalRenderer{bytes: []byte("%PDF-quote")}
	writer := &stubFileWriter{resolvedPath: "/tmp/proposals/quo_123.pdf", size: 10}
	svc := NewQuotePDFService(fx.quotes, fx.customers, fx.issuers, fx.legalEntities, renderer, writer)

	got, err := svc.RenderQuotePDF(ctx, RenderQuotePDFCommand{QuoteID: fx.quote.ID, OutputPath: "out/quo_123.pdf"})
	if err != nil {
		t.Fatalf("RenderQuotePDF() error = %v", err)
	}

	if got.QuoteID != fx.quote.ID || got.Filename != "quo_123.pdf" || got.Path != "/tmp/proposals/quo_123.pdf" || got.MimeType != "application/pdf" || got.SizeBytes != 10 {
		t.Fatalf("RenderQuotePDF() = %+v, want quote PDF metadata", got)
	}
	if renderer.calls != 1 || writer.resolveCalls != 1 || writer.writeCalls != 1 {
		t.Fatalf("calls renderer=%d resolve=%d write=%d, want 1 each", renderer.calls, writer.resolveCalls, writer.writeCalls)
	}
	if renderer.doc.QuoteID != fx.quote.ID || renderer.doc.Status != "sent" || renderer.doc.Currency != "USD" || renderer.doc.Notes != "Proposal notes" {
		t.Fatalf("document identity = %+v", renderer.doc)
	}
	if renderer.doc.Customer.LegalName != "Customer LLC" || renderer.doc.Issuer.LegalName != "Issuer Inc" {
		t.Fatalf("document parties = issuer %q customer %q", renderer.doc.Issuer.LegalName, renderer.doc.Customer.LegalName)
	}
	if len(renderer.doc.Lines) != 2 || renderer.doc.Lines[0].Description != "Discovery" || renderer.doc.Lines[1].Description != "Implementation" {
		t.Fatalf("document lines = %+v", renderer.doc.Lines)
	}
	if renderer.doc.Total != 30000 || renderer.doc.Lines[0].LineTotalAmount != 10000 || renderer.doc.Lines[1].LineTotalAmount != 20000 {
		t.Fatalf("document totals = total %d lines %+v", renderer.doc.Total, renderer.doc.Lines)
	}
	if writer.resolvedOutputPath != "out/quo_123.pdf" || writer.wrotePath != "/tmp/proposals/quo_123.pdf" || string(writer.wroteBytes) != "%PDF-quote" {
		t.Fatalf("writer got output=%q path=%q bytes=%q", writer.resolvedOutputPath, writer.wrotePath, string(writer.wroteBytes))
	}
}

func TestQuotePDFServiceRenderQuotePDFUsesDefaultFilename(t *testing.T) {
	ctx := context.Background()
	fx := newQuotePDFFixture(t)
	renderer := &stubQuoteProposalRenderer{bytes: []byte("%PDF-default")}
	writer := &stubFileWriter{resolvedPath: "/tmp/quote-quo_123.pdf", size: 12}
	svc := NewQuotePDFService(fx.quotes, fx.customers, fx.issuers, fx.legalEntities, renderer, writer)

	got, err := svc.RenderQuotePDF(ctx, RenderQuotePDFCommand{QuoteID: " quo_123 "})
	if err != nil {
		t.Fatalf("RenderQuotePDF() error = %v", err)
	}

	if writer.resolvedFilename != "quote-quo_123.pdf" || got.Filename != "quote-quo_123.pdf" {
		t.Fatalf("default filename writer=%q dto=%q", writer.resolvedFilename, got.Filename)
	}
}

func TestQuotePDFServiceRenderQuotePDFMissingData(t *testing.T) {
	ctx := context.Background()

	t.Run("quote not found", func(t *testing.T) {
		fx := newQuotePDFFixture(t)
		fx.quotes.quote = nil
		svc := NewQuotePDFService(fx.quotes, fx.customers, fx.issuers, fx.legalEntities, &stubQuoteProposalRenderer{}, &stubFileWriter{})
		_, err := svc.RenderQuotePDF(ctx, RenderQuotePDFCommand{QuoteID: "quo_missing"})
		if !errors.Is(err, ErrQuoteNotFound) {
			t.Fatalf("RenderQuotePDF() error = %v, want ErrQuoteNotFound", err)
		}
	})

	t.Run("customer legal entity missing", func(t *testing.T) {
		fx := newQuotePDFFixture(t)
		delete(fx.legalEntities.entities, fx.customerEntity.ID)
		renderer := &stubQuoteProposalRenderer{bytes: []byte("%PDF")}
		writer := &stubFileWriter{resolvedPath: "/tmp/quote.pdf"}
		svc := NewQuotePDFService(fx.quotes, fx.customers, fx.issuers, fx.legalEntities, renderer, writer)
		_, err := svc.RenderQuotePDF(ctx, RenderQuotePDFCommand{QuoteID: fx.quote.ID, OutputPath: "quote.pdf"})
		if err == nil || !strings.Contains(err.Error(), fx.customer.LegalEntityID) || !strings.Contains(err.Error(), "legal entity") {
			t.Fatalf("RenderQuotePDF() error = %v, want legal entity id", err)
		}
		if renderer.calls != 0 || writer.writeCalls != 0 {
			t.Fatalf("renderer/writer calls = %d/%d, want 0/0", renderer.calls, writer.writeCalls)
		}
	})

	t.Run("issuer profile missing", func(t *testing.T) {
		fx := newQuotePDFFixture(t)
		fx.issuers.profile = nil
		svc := NewQuotePDFService(fx.quotes, fx.customers, fx.issuers, fx.legalEntities, &stubQuoteProposalRenderer{}, &stubFileWriter{})
		_, err := svc.RenderQuotePDF(ctx, RenderQuotePDFCommand{QuoteID: fx.quote.ID})
		if !errors.Is(err, ErrIssuerProfileNotFound) {
			t.Fatalf("RenderQuotePDF() error = %v, want ErrIssuerProfileNotFound", err)
		}
	})
}

func TestQuotePDFServiceRenderQuotePDFRenderAndWriteFailures(t *testing.T) {
	ctx := context.Background()
	fx := newQuotePDFFixture(t)

	t.Run("renderer failure", func(t *testing.T) {
		renderer := &stubQuoteProposalRenderer{err: errors.New("boom render")}
		writer := &stubFileWriter{resolvedPath: "/tmp/quote.pdf"}
		svc := NewQuotePDFService(fx.quotes, fx.customers, fx.issuers, fx.legalEntities, renderer, writer)
		_, err := svc.RenderQuotePDF(ctx, RenderQuotePDFCommand{QuoteID: fx.quote.ID, OutputPath: "quote.pdf"})
		if err == nil || !strings.Contains(err.Error(), "render quote pdf") || !strings.Contains(err.Error(), "boom render") {
			t.Fatalf("RenderQuotePDF() error = %v, want render wrapper", err)
		}
		if writer.writeCalls != 0 {
			t.Fatalf("writer calls = %d, want 0", writer.writeCalls)
		}
	})

	t.Run("writer failure", func(t *testing.T) {
		renderer := &stubQuoteProposalRenderer{bytes: []byte("%PDF")}
		writer := &stubFileWriter{resolvedPath: "/tmp/quote.pdf", writeErr: errors.New("disk full")}
		svc := NewQuotePDFService(fx.quotes, fx.customers, fx.issuers, fx.legalEntities, renderer, writer)
		_, err := svc.RenderQuotePDF(ctx, RenderQuotePDFCommand{QuoteID: fx.quote.ID, OutputPath: "quote.pdf"})
		if err == nil || !strings.Contains(err.Error(), "write quote pdf") || !strings.Contains(err.Error(), "disk full") {
			t.Fatalf("RenderQuotePDF() error = %v, want write wrapper", err)
		}
	})
}

type quotePDFFixture struct {
	quote          core.Quote
	customer       core.CustomerProfile
	issuer         core.IssuerProfile
	customerEntity core.LegalEntity
	issuerEntity   core.LegalEntity
	quotes         *stubPDFQuoteStore
	customers      *stubPDFCustomerStore
	issuers        *stubPDFIssuerStore
	legalEntities  *stubPDFLegalEntityStore
}

func newQuotePDFFixture(t *testing.T) quotePDFFixture {
	t.Helper()
	customerEntity, _ := core.NewLegalEntity(core.LegalEntityParams{Type: core.EntityTypeCompany, LegalName: "Customer LLC", TaxID: "C-123", Email: "billing@example.test", BillingAddress: core.Address{Street: "Customer St", City: "Santo Domingo", Country: "DO"}})
	issuerEntity, _ := core.NewLegalEntity(core.LegalEntityParams{Type: core.EntityTypeCompany, LegalName: "Issuer Inc", TaxID: "I-123", Email: "issuer@example.test", BillingAddress: core.Address{Street: "Issuer St", City: "Santo Domingo", Country: "DO"}})
	customer, _ := core.NewCustomerProfile(core.CustomerProfileParams{LegalEntityID: customerEntity.ID, DefaultCurrency: "USD"})
	issuer, _ := core.NewIssuerProfile(core.IssuerProfileParams{LegalEntityID: issuerEntity.ID, DefaultCurrency: "USD", DefaultNotes: "Thanks"})
	rate, _ := core.NewMoney(10000, "USD")
	line1, _ := core.NewQuoteLine(core.QuoteLineParams{QuoteID: "quo_123", ServiceAgreementID: "sa_1", Description: "Discovery", QuantityMin: 60, UnitRate: rate}, "USD")
	line2, _ := core.NewQuoteLine(core.QuoteLineParams{QuoteID: "quo_123", ServiceAgreementID: "sa_2", Description: "Implementation", QuantityMin: 120, UnitRate: rate}, "USD")
	quote := core.Quote{ID: "quo_123", CustomerID: customer.ID, Status: core.QuoteStatusSent, Currency: "USD", Notes: "Proposal notes", Lines: []core.QuoteLine{line1, line2}, SentAt: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), CreatedAt: time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC)}

	return quotePDFFixture{
		quote: quote, customer: customer, issuer: issuer, customerEntity: customerEntity, issuerEntity: issuerEntity,
		quotes:        &stubPDFQuoteStore{quote: &quote},
		customers:     &stubPDFCustomerStore{profiles: map[string]*core.CustomerProfile{customer.ID: &customer}},
		issuers:       &stubPDFIssuerStore{profile: &issuer},
		legalEntities: &stubPDFLegalEntityStore{entities: map[string]*core.LegalEntity{customerEntity.ID: &customerEntity, issuerEntity.ID: &issuerEntity}},
	}
}

type stubQuoteProposalRenderer struct {
	calls int
	doc   QuoteProposalDocumentDTO
	bytes []byte
	err   error
}

func (s *stubQuoteProposalRenderer) RenderQuoteProposal(ctx context.Context, doc QuoteProposalDocumentDTO) ([]byte, error) {
	_ = ctx
	s.calls++
	s.doc = doc
	return s.bytes, s.err
}

type stubPDFQuoteStore struct {
	quote *core.Quote
	err   error
}

func (s *stubPDFQuoteStore) Create(context.Context, *core.Quote) error { return nil }
func (s *stubPDFQuoteStore) GetByID(context.Context, string) (*core.Quote, error) {
	return s.quote, s.err
}
func (s *stubPDFQuoteStore) ListByCustomer(context.Context, string, ...core.QuoteStatus) ([]core.QuoteSummary, error) {
	return nil, nil
}
func (s *stubPDFQuoteStore) AddLine(context.Context, string, core.QuoteLine) error { return nil }
func (s *stubPDFQuoteStore) Update(context.Context, *core.Quote) error             { return nil }
func (s *stubPDFQuoteStore) Delete(context.Context, string) error                  { return nil }
