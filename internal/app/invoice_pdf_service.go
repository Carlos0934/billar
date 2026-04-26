package app

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Carlos0934/billar/internal/core"
)

const invoicePDFMimeType = "application/pdf"

type PDFRenderer interface {
	Render(ctx context.Context, doc InvoiceDocumentDTO) ([]byte, error)
}

type FileWriter interface {
	Resolve(filename, outputPath string) (string, error)
	Write(absPath string, data []byte) (int64, error)
}

type DefaultIssuerProfileStore interface {
	IssuerProfileStore
	GetDefault(ctx context.Context) (*core.IssuerProfile, error)
}

type InvoicePDFService struct {
	invoices      InvoiceStore
	entries       TimeEntryStore
	customers     CustomerProfileStore
	issuers       DefaultIssuerProfileStore
	legalEntities LegalEntityStore
	renderer      PDFRenderer
	writer        FileWriter
}

func NewInvoicePDFService(invoices InvoiceStore, entries TimeEntryStore, customers CustomerProfileStore, issuers DefaultIssuerProfileStore, legalEntities LegalEntityStore, renderer PDFRenderer, writer FileWriter) InvoicePDFService {
	return InvoicePDFService{invoices: invoices, entries: entries, customers: customers, issuers: issuers, legalEntities: legalEntities, renderer: renderer, writer: writer}
}

func (s InvoicePDFService) RenderInvoicePDF(ctx context.Context, cmd RenderInvoicePDFCommand) (RenderedFileDTO, error) {
	if strings.TrimSpace(cmd.InvoiceID) == "" {
		return RenderedFileDTO{}, errors.New("invoice id is required")
	}
	if s.invoices == nil || s.entries == nil || s.customers == nil || s.issuers == nil || s.legalEntities == nil || s.renderer == nil || s.writer == nil {
		return RenderedFileDTO{}, errors.New("invoice pdf service dependencies are required")
	}

	doc, err := s.buildDocument(ctx, strings.TrimSpace(cmd.InvoiceID))
	if err != nil {
		return RenderedFileDTO{}, fmt.Errorf("build invoice pdf document: %w", err)
	}

	pdfBytes, err := s.renderer.Render(ctx, doc)
	if err != nil {
		return RenderedFileDTO{}, fmt.Errorf("render invoice pdf: %w", err)
	}

	filename := strings.TrimSpace(cmd.Filename)
	if filename == "" && strings.TrimSpace(cmd.OutputPath) == "" {
		filename = defaultInvoicePDFFilename(doc)
	}
	absPath, err := s.writer.Resolve(filename, strings.TrimSpace(cmd.OutputPath))
	if err != nil {
		return RenderedFileDTO{}, fmt.Errorf("resolve invoice pdf path: %w", err)
	}
	size, err := s.writer.Write(absPath, pdfBytes)
	if err != nil {
		return RenderedFileDTO{}, fmt.Errorf("write invoice pdf: %w", err)
	}

	return RenderedFileDTO{InvoiceID: doc.InvoiceID, Filename: filepath.Base(absPath), Path: absPath, MimeType: invoicePDFMimeType, SizeBytes: size}, nil
}

func (s InvoicePDFService) buildDocument(ctx context.Context, invoiceID string) (InvoiceDocumentDTO, error) {
	invoice, err := s.invoices.GetByID(ctx, invoiceID)
	if err != nil {
		if errors.Is(err, ErrInvoiceNotFound) {
			return InvoiceDocumentDTO{}, ErrInvoiceNotFound
		}
		return InvoiceDocumentDTO{}, fmt.Errorf("get invoice %s: %w", invoiceID, err)
	}
	if invoice == nil {
		return InvoiceDocumentDTO{}, fmt.Errorf("get invoice %s: %w", invoiceID, ErrInvoiceNotFound)
	}

	entries := make([]core.TimeEntry, 0, len(invoice.Lines))
	for _, line := range invoice.Lines {
		if strings.TrimSpace(line.TimeEntryID) == "" {
			continue
		}
		entry, err := s.entries.GetByID(ctx, line.TimeEntryID)
		if err != nil {
			return InvoiceDocumentDTO{}, fmt.Errorf("get time entry %s: %w", line.TimeEntryID, err)
		}
		if entry != nil {
			entries = append(entries, *entry)
		}
	}
	invDTO := invoiceToDTO(*invoice, entries)

	customer, err := s.customers.GetByID(ctx, invoice.CustomerID)
	if err != nil {
		return InvoiceDocumentDTO{}, fmt.Errorf("get customer profile %s: %w", invoice.CustomerID, err)
	}
	if customer == nil {
		return InvoiceDocumentDTO{}, fmt.Errorf("get customer profile %s: %w", invoice.CustomerID, ErrCustomerProfileNotFound)
	}
	customerEntity, err := s.legalEntities.GetByID(ctx, customer.LegalEntityID)
	if err != nil {
		return InvoiceDocumentDTO{}, fmt.Errorf("get customer legal entity %s: %w", customer.LegalEntityID, err)
	}

	issuer, err := s.issuers.GetDefault(ctx)
	if err != nil {
		return InvoiceDocumentDTO{}, fmt.Errorf("get issuer profile: %w", err)
	}
	if issuer == nil {
		return InvoiceDocumentDTO{}, fmt.Errorf("get issuer profile: %w", ErrIssuerProfileNotFound)
	}
	issuerEntity, err := s.legalEntities.GetByID(ctx, issuer.LegalEntityID)
	if err != nil {
		return InvoiceDocumentDTO{}, fmt.Errorf("get issuer legal entity %s: %w", issuer.LegalEntityID, err)
	}

	lines := make([]InvoiceDocumentLineDTO, 0, len(invDTO.Lines))
	for _, line := range invDTO.Lines {
		lines = append(lines, InvoiceDocumentLineDTO{Description: line.Description, QuantityMin: line.QuantityMin, UnitRateAmount: line.UnitRateAmount, UnitRateCurrency: line.UnitRateCurrency, LineTotalAmount: line.LineTotalAmount, LineTotalCurrency: line.LineTotalCurrency})
	}

	return InvoiceDocumentDTO{InvoiceID: invDTO.ID, InvoiceNumber: invDTO.InvoiceNumber, Status: invDTO.Status, Currency: invDTO.Currency, PeriodStart: invDTO.PeriodStart, PeriodEnd: invDTO.PeriodEnd, DueDate: invDTO.DueDate, IssuedAt: invDTO.IssuedAt, CreatedAt: invDTO.CreatedAt, Issuer: invoiceDocumentParty(*issuerEntity), Customer: invoiceDocumentParty(*customerEntity), Lines: lines, Subtotal: invDTO.Subtotal, GrandTotal: invDTO.GrandTotal, Notes: invDTO.Notes}, nil
}

func invoiceDocumentParty(entity core.LegalEntity) InvoiceDocumentPartyDTO {
	return InvoiceDocumentPartyDTO{LegalName: entity.LegalName, TradeName: entity.TradeName, TaxID: entity.TaxID, Email: entity.Email, Phone: entity.Phone, Website: entity.Website, BillingAddress: addressToDTO(entity.BillingAddress)}
}

func defaultInvoicePDFFilename(doc InvoiceDocumentDTO) string {
	identity := strings.TrimSpace(doc.InvoiceNumber)
	if identity == "" {
		identity = strings.TrimSpace(doc.InvoiceID)
	}
	identity = strings.NewReplacer("/", "-", `\\`, "-", " ", "-").Replace(identity)
	return "invoice-" + identity + ".pdf"
}
