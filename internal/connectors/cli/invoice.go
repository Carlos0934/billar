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
}

func (c Command) runInvoice(ctx context.Context, args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: billar invoice <draft|issue|discard> [flags]")
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
	default:
		return fmt.Errorf("unknown command %q", strings.Join([]string{"invoice", args[0]}, " "))
	}
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
		customerID  string
		formatValue string
	)

	flags.StringVar(&customerID, "customer-id", "", "customer profile ID")
	flags.StringVar(&formatValue, "format", string(FormatText), "output format")

	if err := flags.Parse(args); err != nil {
		return app.CreateDraftFromUnbilledCommand{}, "", fmt.Errorf("invoice draft: %w", err)
	}
	if flags.NArg() != 0 {
		return app.CreateDraftFromUnbilledCommand{}, "", errors.New("usage: billar invoice draft --customer-id <id>")
	}
	if customerID == "" {
		return app.CreateDraftFromUnbilledCommand{}, "", errors.New("--customer-id is required")
	}

	format, err := ParseFormat(formatValue)
	if err != nil {
		return app.CreateDraftFromUnbilledCommand{}, "", err
	}

	return app.CreateDraftFromUnbilledCommand{CustomerProfileID: customerID}, format, nil
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

func writeInvoiceText(out io.Writer, inv app.InvoiceDTO, colorEnabled bool) error {
	view := newTextView(out, colorEnabled)
	view.Field("ID", inv.ID)
	if inv.InvoiceNumber != "" {
		view.Field("Number", inv.InvoiceNumber)
	}
	view.Field("Customer", inv.CustomerID)
	view.Field("Status", inv.Status)
	view.Field("Currency", inv.Currency)
	if len(inv.Lines) > 0 {
		view.Field("Lines", fmt.Sprintf("%d", len(inv.Lines)))
	}
	_, err := io.WriteString(out, view.Build())
	return err
}
