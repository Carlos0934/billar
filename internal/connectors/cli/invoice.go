package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/Carlos0934/billar/internal/app"
)

// InvoiceServiceProvider is the seam that CLI commands use to call InvoiceService operations.
type InvoiceServiceProvider interface {
	CreateDraftFromUnbilled(ctx context.Context, cmd app.CreateDraftFromUnbilledCommand) (app.InvoiceDTO, error)
	IssueDraft(ctx context.Context, cmd app.IssueInvoiceCommand) (app.InvoiceDTO, error)
	Discard(ctx context.Context, id string) (app.DiscardResult, error)
	GetInvoice(ctx context.Context, id string) (app.InvoiceDTO, error)
	ListInvoices(ctx context.Context, customerID string, statusFilter string) ([]app.InvoiceSummaryDTO, error)
	RenderInvoicePDF(ctx context.Context, cmd app.RenderInvoicePDFCommand) (app.RenderedFileDTO, error)
	AddDraftLine(ctx context.Context, cmd app.AddDraftLineCommand) (app.InvoiceDTO, error)
	RemoveDraftLine(ctx context.Context, cmd app.RemoveDraftLineCommand) (app.InvoiceDTO, error)
}

func (c Command) runInvoice(ctx context.Context, args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: billar invoice <draft|issue|discard|show|list|pdf|line> [flags]")
	}

	subcommand := strings.ToLower(args[0])
	if c.invoice == nil {
		return errors.New("invoice service is required")
	}

	switch subcommand {
	case "draft":
		return c.runInvoiceDraft(ctx, args[1:], out)
	case "issue":
		return c.runInvoiceIssue(ctx, args[1:], out)
	case "discard":
		return c.runInvoiceDiscard(ctx, args[1:], out)
	case "show":
		return c.runInvoiceShow(ctx, args[1:], out)
	case "list":
		return c.runInvoiceList(ctx, args[1:], out)
	case "pdf":
		return c.runInvoicePDF(ctx, args[1:], out)
	case "line":
		return c.runInvoiceLine(ctx, args[1:], out)
	default:
		return fmt.Errorf("unknown command %q", strings.Join([]string{"invoice", args[0]}, " "))
	}
}

func (c Command) runInvoiceLine(ctx context.Context, args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: billar invoice line <add|remove> [flags]")
	}
	switch strings.ToLower(args[0]) {
	case "add":
		cmd, format, err := parseInvoiceLineAddFlags(args[1:])
		if err != nil {
			return err
		}
		result, err := c.invoice.AddDraftLine(ctx, cmd)
		if err != nil {
			return fmt.Errorf("run invoice line add command: %w", err)
		}
		return WriteOutput(out, format, OutputResult{Payload: result, TextWriter: func(w io.Writer) error { return writeInvoiceLineAddText(w, result, c.colorEnabled) }})
	case "remove":
		cmd, format, err := parseInvoiceLineRemoveFlags(args[1:])
		if err != nil {
			return err
		}
		result, err := c.invoice.RemoveDraftLine(ctx, cmd)
		if err != nil {
			return fmt.Errorf("run invoice line remove command: %w", err)
		}
		return WriteOutput(out, format, OutputResult{Payload: result, TextWriter: func(w io.Writer) error { return writeInvoiceLineRemoveText(w, result, c.colorEnabled) }})
	default:
		return fmt.Errorf("unknown command %q", strings.Join([]string{"invoice", "line", args[0]}, " "))
	}
}

func (c Command) runInvoicePDF(ctx context.Context, args []string, out io.Writer) error {
	cmd, format, err := parseInvoicePDFFlags(args)
	if err != nil {
		return err
	}
	result, err := c.invoice.RenderInvoicePDF(ctx, cmd)
	if err != nil {
		return fmt.Errorf("run invoice pdf command: %w", err)
	}
	output := OutputResult{Payload: result, TextWriter: func(w io.Writer) error { return writeRenderedFileText(w, result, c.colorEnabled) }}
	if err := WriteOutput(out, format, output); err != nil {
		return fmt.Errorf("write invoice pdf output: %w", err)
	}
	return nil
}

func (c Command) runInvoiceDraft(ctx context.Context, args []string, out io.Writer) error {
	cmd, format, err := parseInvoiceDraftFlags(args)
	if err != nil {
		return err
	}

	result, err := c.invoice.CreateDraftFromUnbilled(ctx, cmd)
	if err != nil {
		return fmt.Errorf("run invoice draft command: %w", err)
	}

	output := OutputResult{
		Payload: result,
		TextWriter: func(w io.Writer) error {
			return writeInvoiceDraftText(w, result, c.colorEnabled)
		},
	}

	if err := WriteOutput(out, format, output); err != nil {
		return fmt.Errorf("write invoice draft output: %w", err)
	}
	return nil
}

func (c Command) runInvoiceIssue(ctx context.Context, args []string, out io.Writer) error {
	id, format, err := parseInvoiceIDFlags("invoice issue", args)
	if err != nil {
		return err
	}

	result, err := c.invoice.IssueDraft(ctx, app.IssueInvoiceCommand{InvoiceID: id})
	if err != nil {
		return fmt.Errorf("run invoice issue command: %w", err)
	}

	output := OutputResult{
		Payload: result,
		TextWriter: func(w io.Writer) error {
			return writeInvoiceIssueText(w, result, c.colorEnabled)
		},
	}

	if err := WriteOutput(out, format, output); err != nil {
		return fmt.Errorf("write invoice issue output: %w", err)
	}
	return nil
}

func (c Command) runInvoiceDiscard(ctx context.Context, args []string, out io.Writer) error {
	id, format, err := parseInvoiceIDFlags("invoice discard", args)
	if err != nil {
		return err
	}

	result, err := c.invoice.Discard(ctx, id)
	if err != nil {
		return fmt.Errorf("run invoice discard command: %w", err)
	}

	payload := map[string]any{"id": id, "was_soft_discard": result.WasSoftDiscard}
	if result.WasSoftDiscard {
		payload["invoice_number"] = result.InvoiceNumber
	}

	output := OutputResult{
		Payload: payload,
		TextWriter: func(w io.Writer) error {
			return writeInvoiceDiscardText(w, id, result, c.colorEnabled)
		},
	}

	if err := WriteOutput(out, format, output); err != nil {
		return fmt.Errorf("write invoice discard output: %w", err)
	}
	return nil
}

// -- flag parsers --

func parseInvoiceDraftFlags(args []string) (app.CreateDraftFromUnbilledCommand, Format, error) {
	flags := flag.NewFlagSet("invoice draft", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	var (
		customerID, periodStart, periodEnd, dueDate, notes string
		formatValue                                        string
	)

	flags.StringVar(&customerID, "customer-id", "", "customer profile ID")
	flags.StringVar(&periodStart, "period-start", "", "billing period start date (YYYY-MM-DD or RFC3339)")
	flags.StringVar(&periodEnd, "period-end", "", "billing period end date (YYYY-MM-DD or RFC3339)")
	flags.StringVar(&dueDate, "due-date", "", "invoice due date (YYYY-MM-DD or RFC3339)")
	flags.StringVar(&notes, "notes", "", "invoice notes")
	flags.StringVar(&formatValue, "format", string(FormatText), "output format")

	if err := flags.Parse(args); err != nil {
		return app.CreateDraftFromUnbilledCommand{}, "", fmt.Errorf("invoice draft: %w", err)
	}
	if flags.NArg() != 0 {
		return app.CreateDraftFromUnbilledCommand{}, "", errors.New("usage: billar invoice draft --customer-id <id> [--period-start <date>] [--period-end <date>] [--due-date <date>] [--notes <notes>]")
	}
	if customerID == "" {
		return app.CreateDraftFromUnbilledCommand{}, "", errors.New("--customer-id is required")
	}

	format, err := ParseFormat(formatValue)
	if err != nil {
		return app.CreateDraftFromUnbilledCommand{}, "", err
	}

	return app.CreateDraftFromUnbilledCommand{CustomerProfileID: customerID, PeriodStart: periodStart, PeriodEnd: periodEnd, DueDate: dueDate, Notes: notes}, format, nil
}

func parseInvoiceIDFlags(name string, args []string) (string, Format, error) {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	var (
		id          string
		formatValue string
	)

	flags.StringVar(&id, "id", "", "invoice ID")
	flags.StringVar(&formatValue, "format", string(FormatText), "output format")

	if err := flags.Parse(args); err != nil {
		return "", "", fmt.Errorf("%s: %w", name, err)
	}
	if flags.NArg() != 0 {
		return "", "", fmt.Errorf("usage: billar %s --id <invoice-id>", name)
	}
	if id == "" {
		return "", "", errors.New("--id is required")
	}

	format, err := ParseFormat(formatValue)
	if err != nil {
		return "", "", err
	}

	return id, format, nil
}

func parseInvoicePDFFlags(args []string) (app.RenderInvoicePDFCommand, Format, error) {
	var invoiceID, outPath, formatValue string
	formatValue = string(FormatText)
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--out":
			if i+1 >= len(args) {
				return app.RenderInvoicePDFCommand{}, "", errors.New("--out is required")
			}
			outPath = args[i+1]
			i++
		case "--format":
			if i+1 >= len(args) {
				return app.RenderInvoicePDFCommand{}, "", errors.New("--format requires a value")
			}
			formatValue = args[i+1]
			i++
		default:
			if strings.HasPrefix(args[i], "-") {
				return app.RenderInvoicePDFCommand{}, "", fmt.Errorf("invoice pdf: unknown flag %s", args[i])
			}
			if invoiceID != "" {
				return app.RenderInvoicePDFCommand{}, "", errors.New("usage: billar invoice pdf <invoice-id> --out <path>")
			}
			invoiceID = args[i]
		}
	}
	if strings.TrimSpace(invoiceID) == "" {
		return app.RenderInvoicePDFCommand{}, "", errors.New("invoice id is required")
	}
	if strings.TrimSpace(outPath) == "" {
		return app.RenderInvoicePDFCommand{}, "", errors.New("--out is required")
	}
	format, err := ParseFormat(formatValue)
	if err != nil {
		return app.RenderInvoicePDFCommand{}, "", err
	}
	return app.RenderInvoicePDFCommand{InvoiceID: strings.TrimSpace(invoiceID), OutputPath: strings.TrimSpace(outPath)}, format, nil
}

func parseInvoiceLineAddFlags(args []string) (app.AddDraftLineCommand, Format, error) {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return app.AddDraftLineCommand{}, "", errors.New("usage: billar invoice line add <invoice-id> --description <text> --minutes <int> --rate <minor> --currency <ISO>")
	}
	invoiceID := strings.TrimSpace(args[0])
	flags := flag.NewFlagSet("invoice line add", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var description, currency, formatValue string
	var minutes int64
	var rate int64
	flags.StringVar(&description, "description", "", "line description")
	flags.Int64Var(&minutes, "minutes", 0, "line quantity in minutes")
	flags.Int64Var(&rate, "rate", 0, "unit rate in minor currency units")
	flags.StringVar(&currency, "currency", "", "line currency")
	flags.StringVar(&formatValue, "format", string(FormatText), "output format")
	if err := flags.Parse(args[1:]); err != nil {
		return app.AddDraftLineCommand{}, "", fmt.Errorf("invoice line add: %w", err)
	}
	if flags.NArg() != 0 {
		return app.AddDraftLineCommand{}, "", errors.New("usage: billar invoice line add <invoice-id> --description <text> --minutes <int> --rate <minor> --currency <ISO>")
	}
	if strings.TrimSpace(description) == "" {
		return app.AddDraftLineCommand{}, "", errors.New("--description is required")
	}
	if minutes <= 0 {
		return app.AddDraftLineCommand{}, "", errors.New("--minutes must be positive")
	}
	if rate <= 0 {
		return app.AddDraftLineCommand{}, "", errors.New("--rate must be positive")
	}
	if strings.TrimSpace(currency) == "" {
		return app.AddDraftLineCommand{}, "", errors.New("--currency is required")
	}
	format, err := ParseFormat(formatValue)
	if err != nil {
		return app.AddDraftLineCommand{}, "", err
	}
	return app.AddDraftLineCommand{InvoiceID: invoiceID, Description: strings.TrimSpace(description), QuantityMin: minutes, UnitRate: rate, Currency: strings.TrimSpace(currency)}, format, nil
}

func parseInvoiceLineRemoveFlags(args []string) (app.RemoveDraftLineCommand, Format, error) {
	if len(args) < 2 || strings.HasPrefix(args[0], "-") || strings.HasPrefix(args[1], "-") {
		return app.RemoveDraftLineCommand{}, "", errors.New("usage: billar invoice line remove <invoice-id> <line-id>")
	}
	invoiceID := strings.TrimSpace(args[0])
	lineID := strings.TrimSpace(args[1])
	flags := flag.NewFlagSet("invoice line remove", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var formatValue string
	flags.StringVar(&formatValue, "format", string(FormatText), "output format")
	if err := flags.Parse(args[2:]); err != nil {
		return app.RemoveDraftLineCommand{}, "", fmt.Errorf("invoice line remove: %w", err)
	}
	if flags.NArg() != 0 {
		return app.RemoveDraftLineCommand{}, "", errors.New("usage: billar invoice line remove <invoice-id> <line-id>")
	}
	format, err := ParseFormat(formatValue)
	if err != nil {
		return app.RemoveDraftLineCommand{}, "", err
	}
	return app.RemoveDraftLineCommand{InvoiceID: invoiceID, InvoiceLineID: lineID}, format, nil
}

// -- text writers --

func writeInvoiceDraftText(out io.Writer, inv app.InvoiceDTO, colorEnabled bool) error {
	view := newTextView(out, colorEnabled)
	view.Title("Invoice Draft Created").Divider("─────────────────────")
	_, err := io.WriteString(out, view.Build())
	if err != nil {
		return err
	}
	return writeInvoiceText(out, inv, colorEnabled)
}

func writeInvoiceIssueText(out io.Writer, inv app.InvoiceDTO, colorEnabled bool) error {
	view := newTextView(out, colorEnabled)
	view.Title("Invoice Issued").Divider("───────────────")
	_, err := io.WriteString(out, view.Build())
	if err != nil {
		return err
	}
	return writeInvoiceText(out, inv, colorEnabled)
}

func writeInvoiceLineAddText(out io.Writer, inv app.InvoiceDTO, colorEnabled bool) error {
	view := newTextView(out, colorEnabled)
	view.Title("Invoice Line Added").Divider("──────────────────")
	if _, err := io.WriteString(out, view.Build()); err != nil {
		return err
	}
	return writeInvoiceText(out, inv, colorEnabled)
}

func writeInvoiceLineRemoveText(out io.Writer, inv app.InvoiceDTO, colorEnabled bool) error {
	view := newTextView(out, colorEnabled)
	view.Title("Invoice Line Removed").Divider("────────────────────")
	if _, err := io.WriteString(out, view.Build()); err != nil {
		return err
	}
	return writeInvoiceText(out, inv, colorEnabled)
}

func writeInvoiceDiscardText(out io.Writer, id string, result app.DiscardResult, colorEnabled bool) error {
	view := newTextView(out, colorEnabled)
	if result.WasSoftDiscard {
		view.Title("Invoice Soft-Discarded").Divider("──────────────────────")
		view.Field("ID", id)
		view.Field("Status", "discarded")
		view.Line("")
		view.Line(fmt.Sprintf("Warning: Invoice %s was soft-discarded. Its number (%s) is permanently consumed.", id, result.InvoiceNumber))
	} else {
		view.Title("Invoice Discarded").Divider("──────────────────")
		view.Field("ID", id)
		view.Field("Status", "deleted")
		view.Line("Draft invoice deleted; linked time entries unlocked.")
	}
	_, err := io.WriteString(out, view.Build())
	return err
}

func writeRenderedFileText(out io.Writer, file app.RenderedFileDTO, colorEnabled bool) error {
	view := newTextView(out, colorEnabled)
	view.Title("Invoice PDF Exported").Divider("────────────────────")
	view.Field("Invoice", file.InvoiceID)
	view.Field("Filename", file.Filename)
	view.Field("Path", file.Path)
	view.Field("MIME Type", file.MimeType)
	view.Field("Size", fmt.Sprintf("%d bytes", file.SizeBytes))
	_, err := io.WriteString(out, view.Build())
	return err
}

func writeInvoiceText(out io.Writer, inv app.InvoiceDTO, colorEnabled bool) error {
	view := newTextView(out, colorEnabled)
	view.Field("ID", inv.ID)
	if inv.InvoiceNumber != "" {
		view.Field("Number", inv.InvoiceNumber)
	}
	view.Field("Customer", inv.CustomerID)
	view.Field("Status", inv.Status)
	view.Field("Currency", inv.Currency)
	if inv.PeriodStart != "" || inv.PeriodEnd != "" {
		view.Field("Period", formatInvoicePeriod(inv.PeriodStart, inv.PeriodEnd))
	}
	if inv.DueDate != "" {
		view.Field("Due Date", inv.DueDate)
	}
	if inv.Notes != "" {
		view.Field("Notes", inv.Notes)
	}
	if len(inv.Lines) > 0 {
		view.Field("Lines", fmt.Sprintf("%d", len(inv.Lines)))
	}
	_, err := io.WriteString(out, view.Build())
	return err
}

func (c Command) runInvoiceShow(ctx context.Context, args []string, out io.Writer) error {
	id, format, err := parseInvoiceIDFlags("invoice show", args)
	if err != nil {
		return err
	}

	result, err := c.invoice.GetInvoice(ctx, id)
	if err != nil {
		return fmt.Errorf("run invoice show command: %w", err)
	}

	output := OutputResult{
		Payload: result,
		TextWriter: func(w io.Writer) error {
			return buildInvoiceShowText(w, result, c.colorEnabled)
		},
	}

	if err := WriteOutput(out, format, output); err != nil {
		return fmt.Errorf("write invoice show output: %w", err)
	}
	return nil
}

func (c Command) runInvoiceList(ctx context.Context, args []string, out io.Writer) error {
	customerID, statusFilter, format, err := parseInvoiceListFlags(args)
	if err != nil {
		return err
	}

	results, err := c.invoice.ListInvoices(ctx, customerID, statusFilter)
	if err != nil {
		return fmt.Errorf("run invoice list command: %w", err)
	}

	output := OutputResult{
		Payload: results,
		TextWriter: func(w io.Writer) error {
			return buildInvoiceListText(w, results, customerID, statusFilter, c.colorEnabled)
		},
	}

	if err := WriteOutput(out, format, output); err != nil {
		return fmt.Errorf("write invoice list output: %w", err)
	}
	return nil
}

func parseInvoiceListFlags(args []string) (string, string, Format, error) {
	flags := flag.NewFlagSet("invoice list", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	var (
		customerID  string
		status      string
		formatValue string
	)

	flags.StringVar(&customerID, "customer-id", "", "customer profile ID")
	flags.StringVar(&status, "status", "", "optional status filter (draft|issued|discarded)")
	flags.StringVar(&formatValue, "format", string(FormatText), "output format")

	if err := flags.Parse(args); err != nil {
		return "", "", "", fmt.Errorf("invoice list: %w", err)
	}
	if customerID == "" {
		return "", "", "", errors.New("--customer-id is required")
	}

	format, err := ParseFormat(formatValue)
	if err != nil {
		return "", "", "", err
	}

	return customerID, status, format, nil
}

func buildInvoiceShowText(out io.Writer, inv app.InvoiceDTO, colorEnabled bool) error {
	var b strings.Builder

	b.WriteString("Invoice\n")
	b.WriteString("───────\n")
	b.WriteString(fmt.Sprintf("ID: %s\n", inv.ID))
	b.WriteString(fmt.Sprintf("Number: %s\n", inv.InvoiceNumber))
	b.WriteString(fmt.Sprintf("Customer: %s\n", inv.CustomerID))
	b.WriteString(fmt.Sprintf("Status: %s\n", inv.Status))
	b.WriteString(fmt.Sprintf("Currency: %s\n", inv.Currency))
	if inv.PeriodStart != "" || inv.PeriodEnd != "" {
		b.WriteString(fmt.Sprintf("Period: %s\n", formatInvoicePeriod(inv.PeriodStart, inv.PeriodEnd)))
	}
	if inv.DueDate != "" {
		b.WriteString(fmt.Sprintf("Due Date: %s\n", inv.DueDate))
	}
	if inv.Notes != "" {
		b.WriteString(fmt.Sprintf("Notes: %s\n", inv.Notes))
	}
	b.WriteString("\n")
	b.WriteString("Lines\n")
	b.WriteString("─────\n")
	b.WriteString(fmt.Sprintf("%-26s %-10s %-12s %s\n", "Description", "Qty(min)", "Rate", "Total"))
	for _, line := range inv.Lines {
		b.WriteString(fmt.Sprintf("%-26s %-10d %-7d %s  %-7d %s\n",
			line.Description,
			line.QuantityMin,
			line.UnitRateAmount, line.UnitRateCurrency,
			line.LineTotalAmount, line.LineTotalCurrency,
		))
	}
	b.WriteString("\n")
	b.WriteString("Totals\n")
	b.WriteString("──────\n")
	b.WriteString(fmt.Sprintf("Subtotal: %d %s\n", inv.Subtotal, inv.Currency))
	b.WriteString(fmt.Sprintf("Grand Total: %d %s\n", inv.GrandTotal, inv.Currency))

	_, err := io.WriteString(out, b.String())
	return err
}

func buildInvoiceListText(out io.Writer, summaries []app.InvoiceSummaryDTO, customerID, statusFilter string, colorEnabled bool) error {
	if len(summaries) == 0 {
		_, err := io.WriteString(out, "No invoices found.\n")
		return err
	}

	var b strings.Builder
	b.WriteString("Invoices\n")
	b.WriteString("────────\n")
	b.WriteString(fmt.Sprintf("Customer: %s\n", customerID))
	if statusFilter != "" {
		b.WriteString(fmt.Sprintf("Status: %s\n", statusFilter))
	}
	b.WriteString(fmt.Sprintf("Count: %d\n", len(summaries)))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("%-16s %-12s %-28s %-21s %-10s %s\n", "Number", "Status", "Period", "Due Date", "Currency", "Total"))
	for _, s := range summaries {
		num := s.InvoiceNumber
		if num == "" {
			num = "—"
		}
		period := formatInvoicePeriodCompact(s.PeriodStart, s.PeriodEnd)
		if period == "" {
			period = "–"
		}
		due := s.DueDate
		if due == "" {
			due = "–"
		}
		b.WriteString(fmt.Sprintf("%-16s %-12s %-28s %-21s %-10s %d\n", num, s.Status, period, due, s.Currency, s.GrandTotal))
	}

	_, err := io.WriteString(out, b.String())
	return err
}

func formatInvoicePeriod(start, end string) string {
	if start == "" && end == "" {
		return ""
	}
	if start == "" {
		start = "—"
	}
	if end == "" {
		end = "—"
	}
	return start + " — " + end
}

func formatInvoicePeriodCompact(start, end string) string {
	if start == "" && end == "" {
		return ""
	}
	if start == "" {
		start = "–"
	}
	if end == "" {
		end = "–"
	}
	return start + "–" + end
}
