package app

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Carlos0934/billar/internal/core"
)

const quotePDFMimeType = "application/pdf"

type QuoteProposalRenderer interface {
	RenderQuoteProposal(ctx context.Context, doc QuoteProposalDocumentDTO) ([]byte, error)
}

type QuotePDFService struct {
	quotes        QuoteStore
	customers     CustomerProfileStore
	issuers       DefaultIssuerProfileStore
	legalEntities LegalEntityStore
	renderer      QuoteProposalRenderer
	writer        FileWriter
}

func NewQuotePDFService(quotes QuoteStore, customers CustomerProfileStore, issuers DefaultIssuerProfileStore, legalEntities LegalEntityStore, renderer QuoteProposalRenderer, writer FileWriter) QuotePDFService {
	return QuotePDFService{quotes: quotes, customers: customers, issuers: issuers, legalEntities: legalEntities, renderer: renderer, writer: writer}
}

func (s QuotePDFService) RenderQuotePDF(ctx context.Context, cmd RenderQuotePDFCommand) (QuoteRenderedFileDTO, error) {
	quoteID := strings.TrimSpace(cmd.QuoteID)
	if quoteID == "" {
		return QuoteRenderedFileDTO{}, errors.New("quote id is required")
	}
	if s.quotes == nil || s.customers == nil || s.issuers == nil || s.legalEntities == nil || s.renderer == nil || s.writer == nil {
		return QuoteRenderedFileDTO{}, errors.New("quote pdf service dependencies are required")
	}

	doc, err := s.buildDocument(ctx, quoteID)
	if err != nil {
		return QuoteRenderedFileDTO{}, fmt.Errorf("build quote pdf document: %w", err)
	}

	pdfBytes, err := s.renderer.RenderQuoteProposal(ctx, doc)
	if err != nil {
		return QuoteRenderedFileDTO{}, fmt.Errorf("render quote pdf: %w", err)
	}

	filename := strings.TrimSpace(cmd.Filename)
	if filename == "" && strings.TrimSpace(cmd.OutputPath) == "" {
		filename = defaultQuotePDFFilename(doc)
	}
	absPath, err := s.writer.Resolve(filename, strings.TrimSpace(cmd.OutputPath))
	if err != nil {
		return QuoteRenderedFileDTO{}, fmt.Errorf("resolve quote pdf path: %w", err)
	}
	size, err := s.writer.Write(absPath, pdfBytes)
	if err != nil {
		return QuoteRenderedFileDTO{}, fmt.Errorf("write quote pdf: %w", err)
	}

	return QuoteRenderedFileDTO{QuoteID: doc.QuoteID, Filename: filepath.Base(absPath), Path: absPath, MimeType: quotePDFMimeType, SizeBytes: size}, nil
}

func (s QuotePDFService) buildDocument(ctx context.Context, quoteID string) (QuoteProposalDocumentDTO, error) {
	quote, err := s.quotes.GetByID(ctx, quoteID)
	if err != nil {
		if errors.Is(err, ErrQuoteNotFound) {
			return QuoteProposalDocumentDTO{}, ErrQuoteNotFound
		}
		return QuoteProposalDocumentDTO{}, fmt.Errorf("get quote %s: %w", quoteID, err)
	}
	if quote == nil {
		return QuoteProposalDocumentDTO{}, fmt.Errorf("get quote %s: %w", quoteID, ErrQuoteNotFound)
	}

	customer, err := s.customers.GetByID(ctx, quote.CustomerID)
	if err != nil {
		return QuoteProposalDocumentDTO{}, fmt.Errorf("get customer profile %s: %w", quote.CustomerID, err)
	}
	if customer == nil {
		return QuoteProposalDocumentDTO{}, fmt.Errorf("get customer profile %s: %w", quote.CustomerID, ErrCustomerProfileNotFound)
	}
	customerEntity, err := s.legalEntities.GetByID(ctx, customer.LegalEntityID)
	if err != nil {
		return QuoteProposalDocumentDTO{}, fmt.Errorf("get customer legal entity %s: %w", customer.LegalEntityID, err)
	}

	issuer, err := s.issuers.GetDefault(ctx)
	if err != nil {
		return QuoteProposalDocumentDTO{}, fmt.Errorf("get issuer profile: %w", err)
	}
	if issuer == nil {
		return QuoteProposalDocumentDTO{}, fmt.Errorf("get issuer profile: %w", ErrIssuerProfileNotFound)
	}
	issuerEntity, err := s.legalEntities.GetByID(ctx, issuer.LegalEntityID)
	if err != nil {
		return QuoteProposalDocumentDTO{}, fmt.Errorf("get issuer legal entity %s: %w", issuer.LegalEntityID, err)
	}

	quoteDTO := quoteToDTO(*quote)
	lines := make([]QuoteProposalDocumentLineDTO, 0, len(quoteDTO.Lines))
	for _, line := range quoteDTO.Lines {
		lines = append(lines, QuoteProposalDocumentLineDTO{Description: line.Description, QuantityMin: line.QuantityMin, UnitRateAmount: line.UnitRateAmount, UnitRateCurrency: line.UnitRateCurrency, LineTotalAmount: line.LineTotalAmount, LineTotalCurrency: line.LineTotalCurrency})
	}

	return QuoteProposalDocumentDTO{QuoteID: quoteDTO.ID, Status: quoteDTO.Status, Currency: quoteDTO.Currency, CreatedAt: quoteDTO.CreatedAt, SentAt: formatQuoteTime(quote.SentAt), Issuer: quoteProposalDocumentParty(*issuerEntity), Customer: quoteProposalDocumentParty(*customerEntity), Lines: lines, Total: quoteDTO.Total, Notes: quoteDTO.Notes}, nil
}

func quoteProposalDocumentParty(entity core.LegalEntity) QuoteProposalDocumentPartyDTO {
	return QuoteProposalDocumentPartyDTO{LegalName: entity.LegalName, TradeName: entity.TradeName, TaxID: entity.TaxID, Email: entity.Email, Phone: entity.Phone, Website: entity.Website, BillingAddress: addressToDTO(entity.BillingAddress)}
}

func defaultQuotePDFFilename(doc QuoteProposalDocumentDTO) string {
	identity := strings.TrimSpace(doc.QuoteID)
	identity = strings.NewReplacer("/", "-", `\\`, "-", " ", "-").Replace(identity)
	return "quote-" + identity + ".pdf"
}
