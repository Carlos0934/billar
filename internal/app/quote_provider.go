package app

import "context"

type QuoteProvider struct {
	QuoteService
	pdf QuotePDFService
}

func NewQuoteProvider(quote QuoteService, pdf QuotePDFService) QuoteProvider {
	return QuoteProvider{QuoteService: quote, pdf: pdf}
}

func (p QuoteProvider) RenderQuotePDF(ctx context.Context, cmd RenderQuotePDFCommand) (QuoteRenderedFileDTO, error) {
	return p.pdf.RenderQuotePDF(ctx, cmd)
}
