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

type QuoteServiceProvider interface {
	Create(ctx context.Context, cmd app.CreateQuoteCommand) (app.QuoteDTO, error)
	List(ctx context.Context, cmd app.ListQuotesCommand) ([]app.QuoteSummaryDTO, error)
	Get(ctx context.Context, id string) (app.QuoteDTO, error)
	AddLine(ctx context.Context, cmd app.AddQuoteLineCommand) (app.QuoteDTO, error)
	Send(ctx context.Context, id string) (app.QuoteDTO, error)
	Accept(ctx context.Context, id string) (app.QuoteDTO, error)
	Reject(ctx context.Context, id string) (app.QuoteDTO, error)
	Expire(ctx context.Context, id string) (app.QuoteDTO, error)
	Delete(ctx context.Context, id string) error
}

func (c Command) runQuote(ctx context.Context, args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: billar quote <create|list|show|add-line|send|accept|reject|expire|delete> [flags]")
	}
	if c.quote == nil {
		return errors.New("quote service is required")
	}

	switch strings.ToLower(args[0]) {
	case "create":
		cmd, format, err := parseQuoteCreateFlags(args[1:])
		if err != nil {
			return err
		}
		result, err := c.quote.Create(ctx, cmd)
		if err != nil {
			return fmt.Errorf("run quote create command: %w", err)
		}
		return WriteOutput(out, format, OutputResult{Payload: result, TextWriter: func(w io.Writer) error { return writeQuoteText(w, "Quote Created", result, c.colorEnabled) }})
	case "list":
		cmd, format, err := parseQuoteListFlags(args[1:])
		if err != nil {
			return err
		}
		result, err := c.quote.List(ctx, cmd)
		if err != nil {
			return fmt.Errorf("run quote list command: %w", err)
		}
		return WriteOutput(out, format, OutputResult{Payload: result, TextWriter: func(w io.Writer) error { return writeQuoteListText(w, result, c.colorEnabled) }})
	case "show":
		id, format, err := parseQuoteIDFlags("quote show", args[1:])
		if err != nil {
			return err
		}
		result, err := c.quote.Get(ctx, id)
		if err != nil {
			return fmt.Errorf("run quote show command: %w", err)
		}
		return WriteOutput(out, format, OutputResult{Payload: result, TextWriter: func(w io.Writer) error { return writeQuoteText(w, "Quote", result, c.colorEnabled) }})
	case "add-line":
		cmd, format, err := parseQuoteAddLineFlags(args[1:])
		if err != nil {
			return err
		}
		result, err := c.quote.AddLine(ctx, cmd)
		if err != nil {
			return fmt.Errorf("run quote add-line command: %w", err)
		}
		return WriteOutput(out, format, OutputResult{Payload: result, TextWriter: func(w io.Writer) error { return writeQuoteText(w, "Quote Line Added", result, c.colorEnabled) }})
	case "send", "accept", "reject", "expire":
		return c.runQuoteLifecycle(ctx, strings.ToLower(args[0]), args[1:], out)
	case "delete":
		id, format, err := parseQuoteIDFlags("quote delete", args[1:])
		if err != nil {
			return err
		}
		if err := c.quote.Delete(ctx, id); err != nil {
			return fmt.Errorf("run quote delete command: %w", err)
		}
		payload := map[string]string{"id": id, "status": "deleted"}
		return WriteOutput(out, format, OutputResult{Payload: payload, TextWriter: func(w io.Writer) error { _, err := fmt.Fprintf(w, "Quote deleted: %s\n", id); return err }})
	default:
		return fmt.Errorf("unknown command %q", strings.Join([]string{"quote", args[0]}, " "))
	}
}

func (c Command) runQuoteLifecycle(ctx context.Context, action string, args []string, out io.Writer) error {
	id, format, err := parseQuoteIDFlags("quote "+action, args)
	if err != nil {
		return err
	}
	var result app.QuoteDTO
	switch action {
	case "send":
		result, err = c.quote.Send(ctx, id)
	case "accept":
		result, err = c.quote.Accept(ctx, id)
	case "reject":
		result, err = c.quote.Reject(ctx, id)
	case "expire":
		result, err = c.quote.Expire(ctx, id)
	}
	if err != nil {
		return fmt.Errorf("run quote %s command: %w", action, err)
	}
	title := "Quote " + strings.Title(action)
	return WriteOutput(out, format, OutputResult{Payload: result, TextWriter: func(w io.Writer) error { return writeQuoteText(w, title, result, c.colorEnabled) }})
}

func parseQuoteCreateFlags(args []string) (app.CreateQuoteCommand, Format, error) {
	flags := flag.NewFlagSet("quote create", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var customerID, currency, notes, formatValue string
	flags.StringVar(&customerID, "customer-id", "", "customer profile ID")
	flags.StringVar(&currency, "currency", "", "quote currency")
	flags.StringVar(&notes, "notes", "", "quote notes")
	flags.StringVar(&formatValue, "format", string(FormatText), "output format")
	if err := flags.Parse(args); err != nil {
		return app.CreateQuoteCommand{}, "", err
	}
	if flags.NArg() != 0 {
		return app.CreateQuoteCommand{}, "", errors.New("usage: billar quote create [flags]")
	}
	if strings.TrimSpace(customerID) == "" {
		return app.CreateQuoteCommand{}, "", errors.New("--customer-id is required")
	}
	if strings.TrimSpace(currency) == "" {
		return app.CreateQuoteCommand{}, "", errors.New("--currency is required")
	}
	format, err := ParseFormat(formatValue)
	return app.CreateQuoteCommand{CustomerID: customerID, Currency: currency, Notes: notes}, format, err
}

func parseQuoteListFlags(args []string) (app.ListQuotesCommand, Format, error) {
	flags := flag.NewFlagSet("quote list", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var customerID, status, formatValue string
	flags.StringVar(&customerID, "customer-id", "", "customer profile ID")
	flags.StringVar(&status, "status", "", "quote status filter")
	flags.StringVar(&formatValue, "format", string(FormatText), "output format")
	if err := flags.Parse(args); err != nil {
		return app.ListQuotesCommand{}, "", err
	}
	if flags.NArg() != 0 {
		return app.ListQuotesCommand{}, "", errors.New("usage: billar quote list [flags]")
	}
	if strings.TrimSpace(customerID) == "" {
		return app.ListQuotesCommand{}, "", errors.New("--customer-id is required")
	}
	format, err := ParseFormat(formatValue)
	return app.ListQuotesCommand{CustomerID: customerID, Status: status}, format, err
}

func parseQuoteIDFlags(name string, args []string) (string, Format, error) {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var id, formatValue string
	flags.StringVar(&id, "id", "", "quote ID")
	flags.StringVar(&formatValue, "format", string(FormatText), "output format")
	if err := flags.Parse(args); err != nil {
		return "", "", err
	}
	if flags.NArg() != 0 {
		return "", "", fmt.Errorf("usage: billar %s [flags]", name)
	}
	if strings.TrimSpace(id) == "" {
		return "", "", errors.New("--id is required")
	}
	format, err := ParseFormat(formatValue)
	return id, format, err
}

func parseQuoteAddLineFlags(args []string) (app.AddQuoteLineCommand, Format, error) {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return app.AddQuoteLineCommand{}, "", errors.New("usage: billar quote add-line <quote-id> [flags]")
	}
	quoteID := args[0]
	flags := flag.NewFlagSet("quote add-line", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var agreementID, description, formatValue string
	var minutes int64
	flags.StringVar(&agreementID, "agreement-id", "", "service agreement ID")
	flags.StringVar(&description, "description", "", "line description")
	flags.Int64Var(&minutes, "minutes", 0, "line quantity in minutes")
	flags.StringVar(&formatValue, "format", string(FormatText), "output format")
	if err := flags.Parse(args[1:]); err != nil {
		return app.AddQuoteLineCommand{}, "", err
	}
	if flags.NArg() != 0 {
		return app.AddQuoteLineCommand{}, "", errors.New("usage: billar quote add-line <quote-id> [flags]")
	}
	if strings.TrimSpace(agreementID) == "" {
		return app.AddQuoteLineCommand{}, "", errors.New("--agreement-id is required")
	}
	if strings.TrimSpace(description) == "" {
		return app.AddQuoteLineCommand{}, "", errors.New("--description is required")
	}
	if minutes <= 0 {
		return app.AddQuoteLineCommand{}, "", errors.New("--minutes must be greater than 0")
	}
	format, err := ParseFormat(formatValue)
	return app.AddQuoteLineCommand{QuoteID: quoteID, ServiceAgreementID: agreementID, Description: description, QuantityMin: minutes}, format, err
}

func writeQuoteText(out io.Writer, title string, result app.QuoteDTO, colorEnabled bool) error {
	view := newTextView(out, colorEnabled)
	view.Title(title).Divider("────────────")
	view.Field("ID", result.ID).Field("Customer ID", result.CustomerID).Field("Status", result.Status).Field("Currency", result.Currency).Field("Total", fmt.Sprintf("%d", result.Total))
	if result.Notes != "" {
		view.Field("Notes", result.Notes)
	}
	view.Field("Can delete", fmt.Sprintf("%v", result.CanDelete)).Field("Can convert to invoice", fmt.Sprintf("%v", result.CanConvertToInvoice))
	if len(result.Lines) > 0 {
		view.Divider("Lines")
		for _, line := range result.Lines {
			view.Line(fmt.Sprintf("- %s %s (%d min) total=%d", line.ID, line.Description, line.QuantityMin, line.LineTotalAmount))
		}
	}
	_, err := io.WriteString(out, view.Build())
	return err
}

func writeQuoteListText(out io.Writer, results []app.QuoteSummaryDTO, colorEnabled bool) error {
	view := newTextView(out, colorEnabled)
	view.Title(fmt.Sprintf("Quotes (%d)", len(results))).Divider("────────────")
	if len(results) == 0 {
		view.Line("No quotes found")
	} else {
		for _, quote := range results {
			view.Line(fmt.Sprintf("- %s customer=%s status=%s total=%d %s", quote.ID, quote.CustomerID, quote.Status, quote.Total, quote.Currency))
		}
	}
	_, err := io.WriteString(out, view.Build())
	return err
}
