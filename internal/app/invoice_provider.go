package app

import "context"

type InvoiceProvider struct {
	InvoiceService
	pdf InvoicePDFService
}

func NewInvoiceProvider(invoice InvoiceService, pdf InvoicePDFService) InvoiceProvider {
	return InvoiceProvider{InvoiceService: invoice, pdf: pdf}
}

func (p InvoiceProvider) RenderInvoicePDF(ctx context.Context, cmd RenderInvoicePDFCommand) (RenderedFileDTO, error) {
	return p.pdf.RenderInvoicePDF(ctx, cmd)
}
