package app

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Carlos0934/billar/internal/core"
)

func TestInvoicePDFServiceRenderInvoicePDFHappyPath(t *testing.T) {
	ctx := context.Background()
	fx := newInvoicePDFFixture(t)
	renderer := &stubPDFRenderer{bytes: []byte("%PDF-test")}
	writer := &stubFileWriter{resolvedPath: "/tmp/inv.pdf", size: 9}
	svc := NewInvoicePDFService(fx.invoices, fx.entries, fx.customers, fx.issuers, fx.legalEntities, renderer, writer)

	got, err := svc.RenderInvoicePDF(ctx, RenderInvoicePDFCommand{InvoiceID: fx.invoice.ID, OutputPath: "out/inv.pdf"})
	if err != nil {
		t.Fatalf("RenderInvoicePDF() error = %v", err)
	}

	if got.InvoiceID != fx.invoice.ID || got.Filename != "inv.pdf" || got.Path != "/tmp/inv.pdf" || got.MimeType != "application/pdf" || got.SizeBytes != 9 {
		t.Fatalf("RenderInvoicePDF() = %+v, want metadata for written PDF", got)
	}
	if renderer.calls != 1 || writer.resolveCalls != 1 || writer.writeCalls != 1 {
		t.Fatalf("calls renderer=%d resolve=%d write=%d, want 1 each", renderer.calls, writer.resolveCalls, writer.writeCalls)
	}
	if renderer.doc.Customer.LegalName != "Customer LLC" || renderer.doc.Issuer.LegalName != "Issuer Inc" {
		t.Fatalf("document parties = issuer %q customer %q", renderer.doc.Issuer.LegalName, renderer.doc.Customer.LegalName)
	}
	if len(renderer.doc.Lines) != 2 || renderer.doc.Subtotal != 24000 || renderer.doc.GrandTotal != 24000 {
		t.Fatalf("document totals/lines = lines %d subtotal %d grand %d", len(renderer.doc.Lines), renderer.doc.Subtotal, renderer.doc.GrandTotal)
	}
	if renderer.doc.PeriodStart != "2026-04-01T00:00:00Z" || renderer.doc.PeriodEnd != "2026-04-30T00:00:00Z" || renderer.doc.DueDate != "2026-05-15T00:00:00Z" || renderer.doc.Notes != "Persisted invoice notes" {
		t.Fatalf("document metadata = (%q,%q,%q,%q)", renderer.doc.PeriodStart, renderer.doc.PeriodEnd, renderer.doc.DueDate, renderer.doc.Notes)
	}
	if writer.wrotePath != "/tmp/inv.pdf" || string(writer.wroteBytes) != "%PDF-test" {
		t.Fatalf("writer got path=%q bytes=%q", writer.wrotePath, string(writer.wroteBytes))
	}
}

func TestInvoicePDFServiceRenderInvoicePDFUsesPersistedManualLineSnapshots(t *testing.T) {
	ctx := context.Background()
	fx := newInvoicePDFFixture(t)
	manual, _ := core.NewManualInvoiceLine(fx.invoice.ID, "sa_1", "Manual setup", 60, core.Money{Amount: 5000, Currency: "USD"}, "USD")
	fx.invoice.Lines = []core.InvoiceLine{manual}
	fx.invoices.invoice = &fx.invoice
	fx.entries.entries = map[string]*core.TimeEntry{}
	renderer := &stubPDFRenderer{bytes: []byte("%PDF-manual")}
	writer := &stubFileWriter{resolvedPath: "/tmp/manual.pdf", size: 11}
	svc := NewInvoicePDFService(fx.invoices, fx.entries, fx.customers, fx.issuers, fx.legalEntities, renderer, writer)

	_, err := svc.RenderInvoicePDF(ctx, RenderInvoicePDFCommand{InvoiceID: fx.invoice.ID, OutputPath: "manual.pdf"})
	if err != nil {
		t.Fatalf("RenderInvoicePDF() error = %v", err)
	}
	if len(renderer.doc.Lines) != 1 || renderer.doc.Lines[0].Description != "Manual setup" || renderer.doc.Lines[0].LineTotalAmount != 5000 {
		t.Fatalf("rendered manual lines = %+v, want snapshot manual line", renderer.doc.Lines)
	}
	if renderer.doc.Subtotal != 5000 || renderer.doc.GrandTotal != 5000 {
		t.Fatalf("rendered totals = %d/%d, want 5000/5000", renderer.doc.Subtotal, renderer.doc.GrandTotal)
	}
}

func TestInvoicePDFServiceRenderInvoicePDFUsesDefaultFilename(t *testing.T) {
	ctx := context.Background()
	fx := newInvoicePDFFixture(t)
	fx.invoice.InvoiceNumber = "INV-2026-0007"
	fx.invoices.invoice = &fx.invoice
	renderer := &stubPDFRenderer{bytes: []byte("%PDF-default")}
	writer := &stubFileWriter{resolvedPath: "/tmp/invoice-INV-2026-0007.pdf", size: 12}
	svc := NewInvoicePDFService(fx.invoices, fx.entries, fx.customers, fx.issuers, fx.legalEntities, renderer, writer)

	got, err := svc.RenderInvoicePDF(ctx, RenderInvoicePDFCommand{InvoiceID: fx.invoice.ID})
	if err != nil {
		t.Fatalf("RenderInvoicePDF() error = %v", err)
	}

	if writer.resolvedFilename != "invoice-INV-2026-0007.pdf" || got.Filename != "invoice-INV-2026-0007.pdf" {
		t.Fatalf("default filename writer=%q dto=%q", writer.resolvedFilename, got.Filename)
	}
}

func TestInvoicePDFServiceRenderInvoicePDFMissingLegalEntity(t *testing.T) {
	ctx := context.Background()
	fx := newInvoicePDFFixture(t)
	delete(fx.legalEntities.entities, fx.customerEntity.ID)
	renderer := &stubPDFRenderer{bytes: []byte("%PDF-test")}
	writer := &stubFileWriter{resolvedPath: "/tmp/inv.pdf", size: 9}
	svc := NewInvoicePDFService(fx.invoices, fx.entries, fx.customers, fx.issuers, fx.legalEntities, renderer, writer)

	_, err := svc.RenderInvoicePDF(ctx, RenderInvoicePDFCommand{InvoiceID: fx.invoice.ID, OutputPath: "out/inv.pdf"})
	if err == nil {
		t.Fatal("RenderInvoicePDF() error = nil, want missing legal entity error")
	}
	if !strings.Contains(err.Error(), fx.customer.LegalEntityID) || !strings.Contains(err.Error(), "legal entity") {
		t.Fatalf("RenderInvoicePDF() error = %q, want legal entity id", err.Error())
	}
	if renderer.calls != 0 || writer.writeCalls != 0 {
		t.Fatalf("renderer/writer calls = %d/%d, want 0/0", renderer.calls, writer.writeCalls)
	}
}

func TestInvoicePDFServiceRenderInvoicePDFInvoiceNotFound(t *testing.T) {
	ctx := context.Background()
	fx := newInvoicePDFFixture(t)
	fx.invoices.err = ErrInvoiceNotFound
	svc := NewInvoicePDFService(fx.invoices, fx.entries, fx.customers, fx.issuers, fx.legalEntities, &stubPDFRenderer{}, &stubFileWriter{})

	_, err := svc.RenderInvoicePDF(ctx, RenderInvoicePDFCommand{InvoiceID: "inv_missing"})
	if !errors.Is(err, ErrInvoiceNotFound) {
		t.Fatalf("RenderInvoicePDF() error = %v, want wraps ErrInvoiceNotFound", err)
	}
}

func TestInvoicePDFServiceRenderInvoicePDFRenderAndWriteFailures(t *testing.T) {
	ctx := context.Background()
	fx := newInvoicePDFFixture(t)

	t.Run("renderer failure", func(t *testing.T) {
		renderer := &stubPDFRenderer{err: errors.New("boom render")}
		writer := &stubFileWriter{resolvedPath: "/tmp/inv.pdf"}
		svc := NewInvoicePDFService(fx.invoices, fx.entries, fx.customers, fx.issuers, fx.legalEntities, renderer, writer)
		_, err := svc.RenderInvoicePDF(ctx, RenderInvoicePDFCommand{InvoiceID: fx.invoice.ID, OutputPath: "out/inv.pdf"})
		if err == nil || !strings.Contains(err.Error(), "render invoice pdf") || !strings.Contains(err.Error(), "boom render") {
			t.Fatalf("RenderInvoicePDF() error = %v, want render wrapper", err)
		}
		if writer.writeCalls != 0 {
			t.Fatalf("writer calls = %d, want 0", writer.writeCalls)
		}
	})

	t.Run("writer failure", func(t *testing.T) {
		renderer := &stubPDFRenderer{bytes: []byte("%PDF-test")}
		writer := &stubFileWriter{resolvedPath: "/tmp/inv.pdf", writeErr: errors.New("disk full")}
		svc := NewInvoicePDFService(fx.invoices, fx.entries, fx.customers, fx.issuers, fx.legalEntities, renderer, writer)
		_, err := svc.RenderInvoicePDF(ctx, RenderInvoicePDFCommand{InvoiceID: fx.invoice.ID, OutputPath: "out/inv.pdf"})
		if err == nil || !strings.Contains(err.Error(), "write invoice pdf") || !strings.Contains(err.Error(), "disk full") {
			t.Fatalf("RenderInvoicePDF() error = %v, want write wrapper", err)
		}
	})
}

type invoicePDFFixture struct {
	invoice        core.Invoice
	customer       core.CustomerProfile
	issuer         core.IssuerProfile
	customerEntity core.LegalEntity
	issuerEntity   core.LegalEntity
	invoices       *stubPDFInvoiceStore
	entries        *stubPDFTimeEntryStore
	customers      *stubPDFCustomerStore
	issuers        *stubPDFIssuerStore
	legalEntities  *stubPDFLegalEntityStore
}

func newInvoicePDFFixture(t *testing.T) invoicePDFFixture {
	t.Helper()
	customerEntity, _ := core.NewLegalEntity(core.LegalEntityParams{Type: core.EntityTypeCompany, LegalName: "Customer LLC", TaxID: "C-123", Email: "billing@example.test", BillingAddress: core.Address{Street: "Customer St", City: "Santo Domingo", Country: "DO"}})
	issuerEntity, _ := core.NewLegalEntity(core.LegalEntityParams{Type: core.EntityTypeCompany, LegalName: "Issuer Inc", TaxID: "I-123", Email: "issuer@example.test", BillingAddress: core.Address{Street: "Issuer St", City: "Santo Domingo", Country: "DO"}})
	customer, _ := core.NewCustomerProfile(core.CustomerProfileParams{LegalEntityID: customerEntity.ID, DefaultCurrency: "USD"})
	issuer, _ := core.NewIssuerProfile(core.IssuerProfileParams{LegalEntityID: issuerEntity.ID, DefaultCurrency: "USD", DefaultNotes: "Thanks"})
	entry1, _ := core.NewTimeEntry(core.TimeEntryParams{CustomerProfileID: customer.ID, ServiceAgreementID: "sa_1", Description: "Strategy", Hours: core.Hours(10000), Billable: true, Date: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)})
	entry2, _ := core.NewTimeEntry(core.TimeEntryParams{CustomerProfileID: customer.ID, ServiceAgreementID: "sa_1", Description: "Implementation", Hours: core.Hours(20000), Billable: true, Date: time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC)})
	rate, _ := core.NewMoney(8000, "USD")
	line1, _ := core.NewInvoiceLine(core.InvoiceLineParams{InvoiceID: "inv_seed", ServiceAgreementID: "sa_1", TimeEntryID: entry1.ID, UnitRate: rate})
	line2, _ := core.NewInvoiceLine(core.InvoiceLineParams{InvoiceID: "inv_seed", ServiceAgreementID: "sa_1", TimeEntryID: entry2.ID, UnitRate: rate})
	invoice, _ := core.NewInvoice(core.InvoiceParams{CustomerID: customer.ID, Status: core.InvoiceStatusDraft, Currency: "USD", Lines: []core.InvoiceLine{line1, line2}, PeriodStart: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), PeriodEnd: time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC), DueDate: time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC), Notes: "Persisted invoice notes", CreatedAt: time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)})
	entry1.AssignToInvoice(invoice.ID)
	entry2.AssignToInvoice(invoice.ID)

	return invoicePDFFixture{
		invoice: invoice, customer: customer, issuer: issuer, customerEntity: customerEntity, issuerEntity: issuerEntity,
		invoices:      &stubPDFInvoiceStore{invoice: &invoice},
		entries:       &stubPDFTimeEntryStore{entries: map[string]*core.TimeEntry{entry1.ID: &entry1, entry2.ID: &entry2}},
		customers:     &stubPDFCustomerStore{profiles: map[string]*core.CustomerProfile{customer.ID: &customer}},
		issuers:       &stubPDFIssuerStore{profile: &issuer},
		legalEntities: &stubPDFLegalEntityStore{entities: map[string]*core.LegalEntity{customerEntity.ID: &customerEntity, issuerEntity.ID: &issuerEntity}},
	}
}

type stubPDFRenderer struct {
	calls int
	doc   InvoiceDocumentDTO
	bytes []byte
	err   error
}

func (s *stubPDFRenderer) Render(ctx context.Context, doc InvoiceDocumentDTO) ([]byte, error) {
	_ = ctx
	s.calls++
	s.doc = doc
	return s.bytes, s.err
}

type stubFileWriter struct {
	resolveCalls, writeCalls                                      int
	resolvedFilename, resolvedOutputPath, resolvedPath, wrotePath string
	wroteBytes                                                    []byte
	size                                                          int64
	resolveErr, writeErr                                          error
}

func (s *stubFileWriter) Resolve(filename, outputPath string) (string, error) {
	s.resolveCalls++
	s.resolvedFilename = filename
	s.resolvedOutputPath = outputPath
	return s.resolvedPath, s.resolveErr
}
func (s *stubFileWriter) Write(path string, data []byte) (int64, error) {
	s.writeCalls++
	s.wrotePath = path
	s.wroteBytes = append([]byte(nil), data...)
	return s.size, s.writeErr
}

type stubPDFInvoiceStore struct {
	invoice *core.Invoice
	err     error
}

func (s *stubPDFInvoiceStore) CreateDraft(context.Context, *core.Invoice, []*core.TimeEntry) error {
	return nil
}
func (s *stubPDFInvoiceStore) GetByID(context.Context, string) (*core.Invoice, error) {
	return s.invoice, s.err
}
func (s *stubPDFInvoiceStore) Update(context.Context, *core.Invoice) error { return nil }
func (s *stubPDFInvoiceStore) Delete(context.Context, string) error        { return nil }
func (s *stubPDFInvoiceStore) ListByCustomer(context.Context, string, ...core.InvoiceStatus) ([]core.InvoiceSummary, error) {
	return nil, nil
}
func (s *stubPDFInvoiceStore) AddLine(context.Context, string, core.InvoiceLine) error { return nil }
func (s *stubPDFInvoiceStore) RemoveLine(context.Context, string, string) error        { return nil }

type stubPDFTimeEntryStore struct{ entries map[string]*core.TimeEntry }

func (s *stubPDFTimeEntryStore) Save(context.Context, *core.TimeEntry) error { return nil }
func (s *stubPDFTimeEntryStore) GetByID(_ context.Context, id string) (*core.TimeEntry, error) {
	return s.entries[id], nil
}
func (s *stubPDFTimeEntryStore) Delete(context.Context, string) error { return nil }
func (s *stubPDFTimeEntryStore) ListByCustomerProfile(context.Context, string) ([]core.TimeEntry, error) {
	return nil, nil
}
func (s *stubPDFTimeEntryStore) ListUnbilled(context.Context, string) ([]core.TimeEntry, error) {
	return nil, nil
}

type stubPDFCustomerStore struct {
	profiles map[string]*core.CustomerProfile
}

func (s *stubPDFCustomerStore) List(context.Context, ListQuery) (ListResult[core.CustomerProfile], error) {
	return ListResult[core.CustomerProfile]{}, nil
}
func (s *stubPDFCustomerStore) Save(context.Context, *core.CustomerProfile) error { return nil }
func (s *stubPDFCustomerStore) GetByID(_ context.Context, id string) (*core.CustomerProfile, error) {
	return s.profiles[id], nil
}
func (s *stubPDFCustomerStore) Delete(context.Context, string) error { return nil }

type stubPDFIssuerStore struct{ profile *core.IssuerProfile }

func (s *stubPDFIssuerStore) Save(context.Context, *core.IssuerProfile) error { return nil }
func (s *stubPDFIssuerStore) GetByID(context.Context, string) (*core.IssuerProfile, error) {
	return s.profile, nil
}
func (s *stubPDFIssuerStore) GetDefault(context.Context) (*core.IssuerProfile, error) {
	return s.profile, nil
}
func (s *stubPDFIssuerStore) Delete(context.Context, string) error { return nil }

type stubPDFLegalEntityStore struct{ entities map[string]*core.LegalEntity }

func (s *stubPDFLegalEntityStore) List(context.Context, ListQuery) (ListResult[core.LegalEntity], error) {
	return ListResult[core.LegalEntity]{}, nil
}
func (s *stubPDFLegalEntityStore) Save(context.Context, *core.LegalEntity) error { return nil }
func (s *stubPDFLegalEntityStore) GetByID(_ context.Context, id string) (*core.LegalEntity, error) {
	e := s.entities[id]
	if e == nil {
		return nil, ErrLegalEntityNotFound
	}
	return e, nil
}
func (s *stubPDFLegalEntityStore) Delete(context.Context, string) error { return nil }
