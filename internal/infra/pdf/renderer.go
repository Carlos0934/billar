package pdf

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"codeberg.org/go-pdf/fpdf"
	"github.com/Carlos0934/billar/internal/app"
)

const (
	pageWidthIn  = 6.0
	pageHeightIn = 8.0
	marginIn     = 0.35
	lineHeightIn = 0.18
	rowHeightIn  = 0.24
)

type Renderer struct{}

func (Renderer) Render(ctx context.Context, doc app.InvoiceDocumentDTO) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	p := fpdf.NewCustom(&fpdf.InitType{UnitStr: "in", Size: fpdf.SizeType{Wd: pageWidthIn, Ht: pageHeightIn}})
	p.SetCompression(false)
	p.SetTitle("Invoice "+firstNonEmpty(doc.InvoiceNumber, doc.InvoiceID), false)
	p.SetMargins(marginIn, marginIn, marginIn)
	p.SetAutoPageBreak(false, marginIn)

	addPageWithHeader(p, doc)
	y := 2.55
	writeTableHeader(p, y)
	y += rowHeightIn
	for _, line := range doc.Lines {
		if y > 6.55 {
			addPageWithHeader(p, doc)
			y = 1.55
			writeTableHeader(p, y)
			y += rowHeightIn
		}
		writeLineRow(p, y, line)
		y += rowHeightIn
	}
	if y > 6.55 {
		addPageWithHeader(p, doc)
		y = 1.55
	}
	writeTotals(p, y+0.15, doc)

	var buf bytes.Buffer
	if err := p.Output(&buf); err != nil {
		return nil, fmt.Errorf("output pdf: %w", err)
	}
	return buf.Bytes(), nil
}

func addPageWithHeader(p *fpdf.Fpdf, doc app.InvoiceDocumentDTO) {
	p.AddPage()
	p.SetFillColor(242, 234, 220)
	p.Rect(0, 0, pageWidthIn, 1.15, "F")
	p.SetTextColor(50, 45, 40)
	p.SetFont("Helvetica", "B", 18)
	p.SetXY(marginIn, 0.32)
	p.CellFormat(3.1, 0.25, "INVOICE", "", 0, "L", false, 0, "")
	p.SetFont("Helvetica", "", 9)
	p.SetXY(marginIn, 0.66)
	p.CellFormat(3.1, 0.18, doc.Issuer.LegalName, "", 0, "L", false, 0, "")
	p.SetFont("Helvetica", "", 8)
	p.SetXY(3.65, 0.32)
	p.MultiCell(2.0, 0.16, invoiceHeaderMetadata(doc), "", "R", false)

	p.SetFont("Helvetica", "B", 9)
	p.SetXY(marginIn, 1.32)
	p.CellFormat(1.0, 0.18, "Bill To", "", 0, "L", false, 0, "")
	p.SetFont("Helvetica", "", 8)
	p.SetXY(marginIn, 1.55)
	p.MultiCell(2.55, 0.16, partyBlock(doc.Customer), "", "L", false)
	p.SetXY(3.05, 1.55)
	p.MultiCell(2.55, 0.16, partyBlock(doc.Issuer), "", "R", false)
}

func writeTableHeader(p *fpdf.Fpdf, y float64) {
	p.SetFillColor(67, 57, 49)
	p.SetTextColor(255, 255, 255)
	p.SetFont("Helvetica", "B", 7.5)
	p.SetXY(marginIn, y)
	p.CellFormat(2.45, rowHeightIn, "Description", "", 0, "L", true, 0, "")
	p.CellFormat(0.65, rowHeightIn, "Qty", "", 0, "R", true, 0, "")
	p.CellFormat(1.0, rowHeightIn, "Rate", "", 0, "R", true, 0, "")
	p.CellFormat(1.2, rowHeightIn, "Total", "", 0, "R", true, 0, "")
	p.SetTextColor(50, 45, 40)
}

func writeLineRow(p *fpdf.Fpdf, y float64, line app.InvoiceDocumentLineDTO) {
	p.SetFont("Helvetica", "", 7.5)
	p.SetXY(marginIn, y)
	p.CellFormat(2.45, rowHeightIn, truncate(line.Description, 34), "B", 0, "L", false, 0, "")
	p.CellFormat(0.65, rowHeightIn, fmt.Sprintf("%d", line.QuantityMin), "B", 0, "R", false, 0, "")
	p.CellFormat(1.0, rowHeightIn, money(line.UnitRateAmount, line.UnitRateCurrency), "B", 0, "R", false, 0, "")
	p.CellFormat(1.2, rowHeightIn, money(line.LineTotalAmount, line.LineTotalCurrency), "B", 0, "R", false, 0, "")
}

func writeTotals(p *fpdf.Fpdf, y float64, doc app.InvoiceDocumentDTO) {
	p.SetFillColor(242, 234, 220)
	p.Rect(3.25, y, 2.4, 0.72, "F")
	p.SetFont("Helvetica", "", 8)
	p.SetXY(3.4, y+0.12)
	p.CellFormat(1.0, lineHeightIn, "Subtotal", "", 0, "L", false, 0, "")
	p.CellFormat(1.1, lineHeightIn, money(doc.Subtotal, doc.Currency), "", 0, "R", false, 0, "")
	p.SetFont("Helvetica", "B", 9)
	p.SetXY(3.4, y+0.38)
	p.CellFormat(1.0, lineHeightIn, "Total", "", 0, "L", false, 0, "")
	p.CellFormat(1.1, lineHeightIn, money(doc.GrandTotal, doc.Currency), "", 0, "R", false, 0, "")
	if strings.TrimSpace(doc.Notes) != "" {
		p.SetFont("Helvetica", "", 7)
		p.SetXY(marginIn, y+0.12)
		p.MultiCell(2.55, 0.15, doc.Notes, "", "L", false)
	}
}

func partyBlock(p app.InvoiceDocumentPartyDTO) string {
	parts := []string{p.LegalName}
	if p.TaxID != "" {
		parts = append(parts, "Tax ID: "+p.TaxID)
	}
	if p.Email != "" {
		parts = append(parts, p.Email)
	}
	addr := strings.TrimSpace(strings.Join([]string{p.BillingAddress.Street, p.BillingAddress.City, p.BillingAddress.State, p.BillingAddress.Country}, ", "))
	if addr != "" {
		parts = append(parts, addr)
	}
	return strings.Join(parts, "\n")
}

func money(amount int64, currency string) string { return fmt.Sprintf("%s %d", currency, amount) }
func invoiceHeaderMetadata(doc app.InvoiceDocumentDTO) string {
	lines := []string{
		fmt.Sprintf("No: %s", firstNonEmpty(doc.InvoiceNumber, "—")),
		fmt.Sprintf("Status: %s", doc.Status),
		fmt.Sprintf("Created: %s", shortDate(doc.CreatedAt)),
		fmt.Sprintf("Issued: %s", shortDate(doc.IssuedAt)),
	}
	if doc.PeriodStart != "" || doc.PeriodEnd != "" {
		lines = append(lines, fmt.Sprintf("Period: %s - %s", shortDate(doc.PeriodStart), shortDate(doc.PeriodEnd)))
	}
	if doc.DueDate != "" {
		lines = append(lines, fmt.Sprintf("Due: %s", shortDate(doc.DueDate)))
	}
	return strings.Join(lines, "\n")
}
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
func shortDate(value string) string {
	if len(value) >= 10 {
		return value[:10]
	}
	return firstNonEmpty(value, "—")
}
func truncate(value string, max int) string {
	r := []rune(value)
	if len(r) <= max {
		return value
	}
	return string(r[:max-1]) + "…"
}
