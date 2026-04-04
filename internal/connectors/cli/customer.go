package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/Carlos0934/billar/internal/app"
)

// formatJSONError converts a json.Unmarshal error into a structured field-path style error.
// For json.SyntaxError, it includes the position where the error occurred.
// For json.UnmarshalTypeError, it includes the field path and expected type.
func formatJSONError(err error) error {
	var syntaxErr *json.SyntaxError
	var typeErr *json.UnmarshalTypeError

	if errors.As(err, &syntaxErr) {
		return fmt.Errorf("json: syntax error at position %d: %s", syntaxErr.Offset, syntaxErr.Error())
	}

	if errors.As(err, &typeErr) {
		if typeErr.Field != "" {
			return fmt.Errorf("json: field %s: expected %s, got %s", typeErr.Field, typeErr.Type, typeErr.Value)
		}
		return fmt.Errorf("json: expected %s, got %s", typeErr.Type, typeErr.Value)
	}

	return fmt.Errorf("json: %s", err.Error())
}

func parseCustomerListFlags(args []string) (app.ListQuery, Format, error) {
	flags := flag.NewFlagSet("customer list", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	var (
		search     string
		sortValue  string
		page       int
		pageSize   int
		formatText string
	)

	flags.StringVar(&search, "search", "", "search by customer name")
	flags.StringVar(&sortValue, "sort", "", "sort field and direction")
	flags.IntVar(&page, "page", 0, "page number")
	flags.IntVar(&pageSize, "page-size", 0, "page size")
	flags.StringVar(&formatText, "format", string(FormatText), "output format")

	if err := flags.Parse(args); err != nil {
		return app.ListQuery{}, "", fmt.Errorf("%s: %w", commandUsage, err)
	}
	if flags.NArg() != 0 {
		return app.ListQuery{}, "", errors.New(commandUsage)
	}

	sortField, sortDir := parseSortValue(sortValue)
	format, err := ParseFormat(formatText)
	if err != nil {
		return app.ListQuery{}, "", err
	}

	return app.ListQuery{
		Page:      page,
		PageSize:  pageSize,
		Search:    search,
		SortField: sortField,
		SortDir:   sortDir,
	}, format, nil
}

func parseSortValue(value string) (string, string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", ""
	}

	if strings.HasPrefix(value, "-") {
		return strings.TrimSpace(strings.TrimPrefix(value, "-")), "desc"
	}

	field, dir, found := strings.Cut(value, ":")
	if !found {
		return strings.TrimSpace(value), ""
	}

	return strings.TrimSpace(field), strings.TrimSpace(dir)
}

func writeCustomerListText(out io.Writer, result app.ListResult[app.CustomerDTO], colorEnabled bool) error {
	view := newTextView(out, colorEnabled)
	view.Title("Billar Customers").Divider("───────────────").Field("Page", strconv.Itoa(result.Page)).Field("Page size", strconv.Itoa(result.PageSize)).Field("Total", strconv.Itoa(result.Total))

	if len(result.Items) == 0 {
		view.Line("No customers found")
		_, err := io.WriteString(out, view.Build())
		return err
	}

	view.Line("")
	for i, customer := range result.Items {
		if i > 0 {
			view.Line("")
		}
		view.Line(fmt.Sprintf("%d. %s", i+1, customer.LegalName))
		if customer.TradeName != "" && customer.TradeName != customer.LegalName {
			view.Line("   Trade name: " + customer.TradeName)
		}
		view.Line("   Type: " + customer.Type)
		view.Line("   Status: " + customer.Status)
		if customer.Email != "" {
			view.Line("   Email: " + customer.Email)
		}
		if customer.DefaultCurrency != "" {
			view.Line("   Default currency: " + customer.DefaultCurrency)
		}
		if customer.CreatedAt != "" {
			view.Line("   Created at: " + customer.CreatedAt)
		}
		if customer.UpdatedAt != "" {
			view.Line("   Updated at: " + customer.UpdatedAt)
		}
	}

	_, err := io.WriteString(out, view.Build())
	return err
}

func parseCustomerCreateFlags(args []string) (app.CreateCustomerCommand, Format, error) {
	flags := flag.NewFlagSet("customer create", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	var (
		jsonPayload string
		formatValue string
	)

	flags.StringVar(&jsonPayload, "json", "", "JSON payload for customer creation")
	flags.StringVar(&formatValue, "format", string(FormatText), "output format")

	if err := flags.Parse(args); err != nil {
		return app.CreateCustomerCommand{}, "", fmt.Errorf("%s: %w", commandUsage, err)
	}

	if flags.NArg() != 0 {
		return app.CreateCustomerCommand{}, "", errors.New(commandUsage)
	}

	if jsonPayload == "" {
		return app.CreateCustomerCommand{}, "", errors.New("--json is required")
	}

	var cmd app.CreateCustomerCommand
	if err := json.Unmarshal([]byte(jsonPayload), &cmd); err != nil {
		return app.CreateCustomerCommand{}, "", formatJSONError(err)
	}

	format, err := ParseFormat(formatValue)
	if err != nil {
		return app.CreateCustomerCommand{}, "", err
	}

	return cmd, format, nil
}

func writeCustomerCreateText(out io.Writer, result app.CustomerDTO, colorEnabled bool) error {
	view := newTextView(out, colorEnabled)
	view.Title("Customer Created").Divider("────────────")
	view.Field("ID", result.ID)
	if result.Type != "" {
		view.Field("Type", result.Type)
	}
	if result.LegalName != "" {
		view.Field("Legal name", result.LegalName)
	}
	if result.Email != "" {
		view.Field("Email", result.Email)
	}
	if result.Status != "" {
		view.Field("Status", result.Status)
	}
	_, err := io.WriteString(out, view.Build())
	return err
}

func parseCustomerUpdateFlags(args []string) (string, app.PatchCustomerCommand, Format, error) {
	flags := flag.NewFlagSet("customer update", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	var (
		id          string
		jsonPayload string
		formatValue string
	)

	flags.StringVar(&id, "id", "", "customer ID")
	flags.StringVar(&jsonPayload, "json", "", "JSON payload for customer update")
	flags.StringVar(&formatValue, "format", string(FormatText), "output format")

	if err := flags.Parse(args); err != nil {
		return "", app.PatchCustomerCommand{}, "", fmt.Errorf("%s: %w", commandUsage, err)
	}

	if flags.NArg() != 0 {
		return "", app.PatchCustomerCommand{}, "", errors.New(commandUsage)
	}

	if id == "" {
		return "", app.PatchCustomerCommand{}, "", errors.New("--id is required")
	}

	if jsonPayload == "" {
		return "", app.PatchCustomerCommand{}, "", errors.New("--json is required")
	}

	var cmd app.PatchCustomerCommand
	if err := json.Unmarshal([]byte(jsonPayload), &cmd); err != nil {
		return "", app.PatchCustomerCommand{}, "", formatJSONError(err)
	}

	format, err := ParseFormat(formatValue)
	if err != nil {
		return "", app.PatchCustomerCommand{}, "", err
	}

	return id, cmd, format, nil
}

func writeCustomerUpdateText(out io.Writer, result app.CustomerDTO, colorEnabled bool) error {
	view := newTextView(out, colorEnabled)
	view.Title("Customer Updated").Divider("────────────")
	view.Field("ID", result.ID)
	if result.Type != "" {
		view.Field("Type", result.Type)
	}
	if result.LegalName != "" {
		view.Field("Legal name", result.LegalName)
	}
	if result.Email != "" {
		view.Field("Email", result.Email)
	}
	if result.Status != "" {
		view.Field("Status", result.Status)
	}
	_, err := io.WriteString(out, view.Build())
	return err
}

func parseCustomerDeleteFlags(args []string) (string, Format, error) {
	flags := flag.NewFlagSet("customer delete", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	var (
		id          string
		formatValue string
	)

	flags.StringVar(&id, "id", "", "customer ID")
	flags.StringVar(&formatValue, "format", string(FormatText), "output format")

	if err := flags.Parse(args); err != nil {
		return "", "", fmt.Errorf("%s: %w", commandUsage, err)
	}

	if flags.NArg() != 0 {
		return "", "", errors.New(commandUsage)
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

func writeCustomerDeleteText(out io.Writer, id string, colorEnabled bool) error {
	view := newTextView(out, colorEnabled)
	view.Title("Customer Deleted").Divider("────────────")
	view.Field("ID", id)
	_, err := io.WriteString(out, view.Build())
	return err
}
