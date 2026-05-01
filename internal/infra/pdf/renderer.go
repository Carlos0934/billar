package pdf

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"codeberg.org/go-pdf/fpdf"
	"github.com/Carlos0934/billar/internal/app"
)

const (
	pageWidthIn  = 6.0
	pageHeightIn = 8.0
	marginIn     = 0.35
	lineHeightIn = 0.18
	contentWidth = pageWidthIn - 2*marginIn
	ruleColor    = 208
)

type pdfText func(string) string

type Renderer struct{}

func (Renderer) Render(ctx context.Context, doc app.InvoiceDocumentDTO) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	p := fpdf.NewCustom(&fpdf.InitType{UnitStr: "in", Size: fpdf.SizeType{Wd: pageWidthIn, Ht: pageHeightIn}})
	p.SetCompression(false)
	p.SetTitle("Invoice "+firstNonEmpty(doc.InvoiceNumber, doc.InvoiceID), true)
	p.SetMargins(marginIn, marginIn, marginIn)
	p.SetAutoPageBreak(false, marginIn)
	p.AliasNbPages("{nb}")
	tr := pdfText(p.UnicodeTranslatorFromDescriptor(""))

	addPageWithHeader(p, tr, doc)
	y := 3.48
	for _, line := range doc.Lines {
		rowH := lineBlockHeight(p, line)
		if y+rowH > 6.72 {
			writeFooter(p, tr, doc)
			addPageWithHeader(p, tr, doc)
			y = 3.48
		}
		writeLineBlock(p, tr, y, line)
		y += rowH + 0.08
	}
	if y > 6.55 {
		writeFooter(p, tr, doc)
		addPageWithHeader(p, tr, doc)
		y = 3.48
	}
	writeTotals(p, tr, y+0.04, doc)
	writeFooter(p, tr, doc)

	var buf bytes.Buffer
	if err := p.Output(&buf); err != nil {
		return nil, fmt.Errorf("output pdf: %w", err)
	}
	return buf.Bytes(), nil
}

func addPageWithHeader(p *fpdf.Fpdf, tr pdfText, doc app.InvoiceDocumentDTO) {
	p.AddPage()
	p.SetFillColor(255, 255, 255)
	p.Rect(0, 0, pageWidthIn, pageHeightIn, "F")
	p.SetTextColor(32, 32, 32)
	p.SetFont("Helvetica", "B", 28)
	p.SetXY(marginIn, 0.38)
	p.CellFormat(contentWidth, 0.32, tr("INVOICE"), "", 0, "L", false, 0, "")
	p.SetFont("Helvetica", "", 10)
	p.SetXY(marginIn, 0.78)
	p.CellFormat(contentWidth, 0.16, tr("Invoice "+firstNonEmpty(doc.InvoiceNumber, doc.InvoiceID)), "", 0, "L", false, 0, "")
	drawRule(p, 1.06)

	colW := contentWidth / 3
	writeMetaCell(p, tr, marginIn, 1.23, colW, "INVOICE DATE", displayDate(firstNonEmpty(doc.InvoiceDate, doc.IssuedAt, doc.CreatedAt)))
	writeMetaCell(p, tr, marginIn+colW, 1.23, colW, "DUE DATE", displayDate(doc.DueDate))
	writeMetaCell(p, tr, marginIn+2*colW, 1.23, colW, "PAYMENT COMMUNICATION", firstNonEmpty(doc.PaymentComm, doc.InvoiceNumber, doc.InvoiceID))
	drawRule(p, 1.74)

	writePartyColumn(p, tr, marginIn, 1.94, "BILL TO", doc.Customer, "L")
	writePartyColumn(p, tr, marginIn+contentWidth/2+0.14, 1.94, "FROM", doc.Issuer, "L")
	drawRule(p, 3.30)
}

func writeMetaCell(p *fpdf.Fpdf, tr pdfText, x, y, w float64, label, value string) {
	p.SetTextColor(90, 90, 90)
	p.SetFont("Helvetica", "B", 7)
	p.SetXY(x, y)
	p.CellFormat(w-0.08, 0.11, tr(label), "", 0, "L", false, 0, "")
	p.SetTextColor(32, 32, 32)
	p.SetFont("Helvetica", "", 9)
	p.SetXY(x, y+0.14)
	p.MultiCell(w-0.08, 0.13, tr(safePDFText(value)), "", "L", false)
}

func writePartyColumn(p *fpdf.Fpdf, tr pdfText, x, y float64, title string, party app.InvoiceDocumentPartyDTO, align string) {
	p.SetTextColor(90, 90, 90)
	p.SetFont("Helvetica", "B", 8)
	p.SetXY(x, y)
	p.CellFormat(2.35, 0.16, tr(title), "", 0, align, false, 0, "")
	p.SetTextColor(32, 32, 32)
	p.SetFont("Helvetica", "", 7.3)
	p.SetXY(x, y+0.25)
	p.MultiCell(2.35, 0.12, tr(safePDFText(partyBlock(party))), "", align, false)
}

func writeLineBlock(p *fpdf.Fpdf, tr pdfText, y float64, line app.InvoiceDocumentLineDTO) {
	description := safePDFText(line.Description)
	p.SetTextColor(90, 90, 90)
	p.SetFont("Helvetica", "B", 7)
	p.SetXY(marginIn, y)
	p.CellFormat(contentWidth, 0.12, tr("WORK DESCRIPTION"), "", 0, "L", false, 0, "")

	p.SetTextColor(32, 32, 32)
	p.SetFont("Helvetica", "", 8)
	p.SetXY(marginIn, y+0.23)
	p.MultiCell(contentWidth, 0.15, tr(description), "", "L", false)

	pricingY := y + workDescriptionHeight(p, description) + 0.20
	drawRule(p, pricingY)
	pricingY += 0.19
	colW := contentWidth / 4
	writePricingColumn(p, tr, marginIn, pricingY, colW, "HOURS", quantity(line), "L")
	writePricingColumn(p, tr, marginIn+colW, pricingY, colW, "UNIT PRICE", unitPrice(line), "L")
	writePricingColumn(p, tr, marginIn+2*colW, pricingY, colW, "TAXES", money(line.TaxMinor, firstNonEmpty(line.LineTotalCurrency, line.UnitRateCurrency)), "L")
	writePricingColumn(p, tr, marginIn+3*colW, pricingY, colW, "AMOUNT", money(line.LineTotalAmount+line.TaxMinor, firstNonEmpty(line.LineTotalCurrency, line.UnitRateCurrency)), "R")
	drawRule(p, pricingY+0.46)
}

func quantity(line app.InvoiceDocumentLineDTO) string {
	displayQuantity := strings.TrimSpace(line.QuantityDisplay)
	if displayQuantity != "" {
		return displayQuantity
	}
	return formatQuantityMinutes(line.QuantityMin)
}

func unitPrice(line app.InvoiceDocumentLineDTO) string {
	displayRate := strings.TrimSpace(line.UnitPriceDisplay)
	if displayRate == "" {
		return money(line.UnitRateAmount, line.UnitRateCurrency)
	}
	currency := firstNonEmpty(line.UnitRateCurrency, line.LineTotalCurrency)
	if currency == "" {
		return displayRate
	}
	return fmt.Sprintf("%s%s %s", strings.TrimSpace(currency), currencySymbol(currency), displayRate)
}

func writePricingColumn(p *fpdf.Fpdf, tr pdfText, x, y, w float64, label, value, align string) {
	p.SetTextColor(90, 90, 90)
	p.SetFont("Helvetica", "B", 7)
	p.SetXY(x, y)
	p.CellFormat(w-0.06, 0.12, tr(label), "", 0, align, false, 0, "")
	p.SetTextColor(32, 32, 32)
	p.SetFont("Helvetica", "", 9)
	p.SetXY(x, y+0.18)
	p.CellFormat(w-0.06, 0.14, tr(safePDFText(value)), "", 0, align, false, 0, "")
}

func lineBlockHeight(p *fpdf.Fpdf, line app.InvoiceDocumentLineDTO) float64 {
	return workDescriptionHeight(p, safePDFText(line.Description)) + 0.86

}

func workDescriptionHeight(p *fpdf.Fpdf, description string) float64 {
	lines := p.SplitLines([]byte(description), contentWidth)
	return math.Max(0.36, float64(len(lines))*0.15+0.25)
}

func writeTotals(p *fpdf.Fpdf, tr pdfText, y float64, doc app.InvoiceDocumentDTO) {
	p.SetTextColor(32, 32, 32)
	p.SetFont("Helvetica", "B", 10)
	colW := contentWidth / 4
	totalY := y + 0.06
	p.SetXY(marginIn+2*colW, totalY)
	p.CellFormat(colW-0.06, lineHeightIn, tr("TOTAL"), "", 0, "R", false, 0, "")
	p.CellFormat(colW-0.06, lineHeightIn, tr(money(doc.GrandTotal, doc.Currency)), "", 0, "R", false, 0, "")
	if strings.TrimSpace(doc.Notes) != "" {
		p.SetTextColor(90, 90, 90)
		p.SetFont("Helvetica", "", 7)
		p.SetXY(marginIn, totalY)
		p.MultiCell(2.35, 0.15, tr(safePDFText(doc.Notes)), "", "L", false)
	}
}

func writeFooter(p *fpdf.Fpdf, tr pdfText, doc app.InvoiceDocumentDTO) {
	drawRule(p, 7.18)
	footer := strings.Join(nonEmpty(doc.Issuer.Email, doc.Issuer.Website), " | ")
	p.SetTextColor(90, 90, 90)
	p.SetFont("Helvetica", "", 7)
	p.SetXY(marginIn, 7.32)
	p.CellFormat(contentWidth, 0.15, tr(safePDFText(footer)), "", 0, "C", false, 0, "")
	p.SetXY(marginIn, 7.50)
	p.CellFormat(contentWidth, 0.15, tr(fmt.Sprintf("Page: %d of {nb}", p.PageNo())), "", 0, "C", false, 0, "")
}

func partyBlock(p app.InvoiceDocumentPartyDTO) string {
	parts := []string{p.LegalName}
	addr := nonEmpty(p.BillingAddress.Street, p.BillingAddress.City, p.BillingAddress.State, p.BillingAddress.PostalCode, p.BillingAddress.Country)
	parts = append(parts, addr...)
	if p.TaxID != "" {
		parts = append(parts, "Tax ID: "+p.TaxID)
	}
	return strings.Join(nonEmpty(parts...), "\n")
}

func drawRule(p *fpdf.Fpdf, y float64) {
	p.SetDrawColor(ruleColor, ruleColor, ruleColor)
	p.Line(marginIn, y, pageWidthIn-marginIn, y)
}

func displayDate(value string) string {
	trimmed := firstNonEmpty(value, "—")
	if trimmed == "—" {
		return trimmed
	}
	if len(trimmed) >= 10 {
		if parsed, err := time.Parse("2006-01-02", trimmed[:10]); err == nil {
			return parsed.Format("01/02/2006")
		}
	}
	return trimmed
}

func safePDFText(value string) string {
	replaced := strings.NewReplacer(
		"á", "a", "à", "a", "â", "a", "ä", "a", "ã", "a", "å", "a",
		"Á", "A", "À", "A", "Â", "A", "Ä", "A", "Ã", "A", "Å", "A",
		"é", "e", "è", "e", "ê", "e", "ë", "e",
		"É", "E", "È", "E", "Ê", "E", "Ë", "E",
		"í", "i", "ì", "i", "î", "i", "ï", "i",
		"Í", "I", "Ì", "I", "Î", "I", "Ï", "I",
		"ó", "o", "ò", "o", "ô", "o", "ö", "o", "õ", "o",
		"Ó", "O", "Ò", "O", "Ô", "O", "Ö", "O", "Õ", "O",
		"ú", "u", "ù", "u", "û", "u", "ü", "u",
		"Ú", "U", "Ù", "U", "Û", "U", "Ü", "U",
		"ñ", "n", "Ñ", "N", "ç", "c", "Ç", "C",
		"€", "EUR", "£", "GBP", "•", "|",
	).Replace(value)
	var b strings.Builder
	for _, r := range replaced {
		if r == '\n' || r == '\t' || (r >= 32 && r <= 126) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func money(amount int64, currency string) string {
	sign := ""
	if amount < 0 {
		sign = "-"
		amount = -amount
	}
	major, minor := amount/100, amount%100
	return fmt.Sprintf("%s%s %s%s.%02d", strings.TrimSpace(currency), currencySymbol(currency), sign, majorWithCommas(major), minor)
}

func currencySymbol(currency string) string {
	switch strings.ToUpper(strings.TrimSpace(currency)) {
	case "USD":
		return "$"
	case "EUR":
		return "€"
	case "GBP":
		return "£"
	case "DOP":
		return "$"
	default:
		return ""
	}
}

func majorWithCommas(n int64) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var b strings.Builder
	pre := len(s) % 3
	if pre == 0 {
		pre = 3
	}
	b.WriteString(s[:pre])
	for i := pre; i < len(s); i += 3 {
		b.WriteString(",")
		b.WriteString(s[i : i+3])
	}
	return b.String()
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func nonEmpty(values ...string) []string {
	items := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			items = append(items, trimmed)
		}
	}
	return items
}

func formatQuantityMinutes(minutes int64) string {
	if minutes == 0 {
		return "—"
	}
	if minutes%60 == 0 {
		return fmt.Sprintf("%d", minutes/60)
	}
	return fmt.Sprintf("%.2f", float64(minutes)/60)
}
