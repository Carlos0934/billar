package cli

import (
	"context"
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

// parseSortValue parses a sort value like "field" or "-field" or "field:asc" into field and direction.
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

type LegalEntityListProvider interface {
	List(ctx context.Context, query app.ListQuery) (app.ListResult[app.LegalEntityDTO], error)
}

type LegalEntityServiceProvider interface {
	LegalEntityListProvider
	Create(ctx context.Context, cmd app.CreateLegalEntityCommand) (app.LegalEntityDTO, error)
	Get(ctx context.Context, id string) (app.LegalEntityDTO, error)
	Update(ctx context.Context, id string, cmd app.PatchLegalEntityCommand) (app.LegalEntityDTO, error)
	Delete(ctx context.Context, id string) error
}

func parseLegalEntityListFlags(args []string) (app.ListQuery, Format, error) {
	flags := flag.NewFlagSet("legal-entity list", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	var (
		search     string
		sortValue  string
		page       int
		pageSize   int
		formatText string
	)

	flags.StringVar(&search, "search", "", "search by legal entity name")
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

func writeLegalEntityListText(out io.Writer, result app.ListResult[app.LegalEntityDTO], colorEnabled bool) error {
	view := newTextView(out, colorEnabled)
	view.Title("Legal Entities").Divider("───────────────").Field("Page", strconv.Itoa(result.Page)).Field("Page size", strconv.Itoa(result.PageSize)).Field("Total", strconv.Itoa(result.Total))

	if len(result.Items) == 0 {
		view.Line("No legal entities found")
		_, err := io.WriteString(out, view.Build())
		return err
	}

	view.Line("")
	for i, entity := range result.Items {
		if i > 0 {
			view.Line("")
		}
		view.Line(fmt.Sprintf("%d. %s", i+1, entity.LegalName))
		if entity.TradeName != "" && entity.TradeName != entity.LegalName {
			view.Line("   Trade name: " + entity.TradeName)
		}
		view.Line("   Type: " + entity.Type)
		if entity.TaxID != "" {
			view.Line("   Tax ID: " + entity.TaxID)
		}
		if entity.Email != "" {
			view.Line("   Email: " + entity.Email)
		}
		if entity.Phone != "" {
			view.Line("   Phone: " + entity.Phone)
		}
		if entity.Website != "" {
			view.Line("   Website: " + entity.Website)
		}
		if entity.CreatedAt != "" {
			view.Line("   Created at: " + entity.CreatedAt)
		}
		if entity.UpdatedAt != "" {
			view.Line("   Updated at: " + entity.UpdatedAt)
		}
	}

	_, err := io.WriteString(out, view.Build())
	return err
}

func parseLegalEntityCreateFlags(args []string) (app.CreateLegalEntityCommand, Format, error) {
	flags := flag.NewFlagSet("legal-entity create", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	var (
		jsonPayload string
		formatValue string
	)

	flags.StringVar(&jsonPayload, "json", "", "JSON payload for legal entity creation")
	flags.StringVar(&formatValue, "format", string(FormatText), "output format")

	if err := flags.Parse(args); err != nil {
		return app.CreateLegalEntityCommand{}, "", fmt.Errorf("%s: %w", commandUsage, err)
	}

	if flags.NArg() != 0 {
		return app.CreateLegalEntityCommand{}, "", errors.New(commandUsage)
	}

	if jsonPayload == "" {
		return app.CreateLegalEntityCommand{}, "", errors.New("--json is required")
	}

	var cmd app.CreateLegalEntityCommand
	if err := json.Unmarshal([]byte(jsonPayload), &cmd); err != nil {
		return app.CreateLegalEntityCommand{}, "", formatJSONError(err)
	}

	format, err := ParseFormat(formatValue)
	if err != nil {
		return app.CreateLegalEntityCommand{}, "", err
	}

	return cmd, format, nil
}

func writeLegalEntityCreateText(out io.Writer, result app.LegalEntityDTO, colorEnabled bool) error {
	view := newTextView(out, colorEnabled)
	view.Title("Legal Entity Created").Divider("────────────")
	view.Field("ID", result.ID)
	if result.Type != "" {
		view.Field("Type", result.Type)
	}
	if result.LegalName != "" {
		view.Field("Legal name", result.LegalName)
	}
	if result.TradeName != "" {
		view.Field("Trade name", result.TradeName)
	}
	if result.TaxID != "" {
		view.Field("Tax ID", result.TaxID)
	}
	if result.Email != "" {
		view.Field("Email", result.Email)
	}
	if result.Phone != "" {
		view.Field("Phone", result.Phone)
	}
	if result.Website != "" {
		view.Field("Website", result.Website)
	}
	_, err := io.WriteString(out, view.Build())
	return err
}

func parseLegalEntityGetFlags(args []string) (string, Format, error) {
	flags := flag.NewFlagSet("legal-entity get", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	var (
		id          string
		formatValue string
	)

	flags.StringVar(&id, "id", "", "legal entity ID")
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

func writeLegalEntityGetText(out io.Writer, result app.LegalEntityDTO, colorEnabled bool) error {
	view := newTextView(out, colorEnabled)
	view.Title("Legal Entity").Divider("────────────")
	view.Field("ID", result.ID)
	if result.Type != "" {
		view.Field("Type", result.Type)
	}
	if result.LegalName != "" {
		view.Field("Legal name", result.LegalName)
	}
	if result.TradeName != "" {
		view.Field("Trade name", result.TradeName)
	}
	if result.TaxID != "" {
		view.Field("Tax ID", result.TaxID)
	}
	if result.Email != "" {
		view.Field("Email", result.Email)
	}
	if result.Phone != "" {
		view.Field("Phone", result.Phone)
	}
	if result.Website != "" {
		view.Field("Website", result.Website)
	}
	if result.BillingAddress.Street != "" || result.BillingAddress.City != "" {
		view.Field("Billing address", formatAddress(result.BillingAddress))
	}
	if result.CreatedAt != "" {
		view.Field("Created at", result.CreatedAt)
	}
	if result.UpdatedAt != "" {
		view.Field("Updated at", result.UpdatedAt)
	}
	_, err := io.WriteString(out, view.Build())
	return err
}

func parseLegalEntityUpdateFlags(args []string) (string, app.PatchLegalEntityCommand, Format, error) {
	flags := flag.NewFlagSet("legal-entity update", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	var (
		id          string
		jsonPayload string
		formatValue string
	)

	flags.StringVar(&id, "id", "", "legal entity ID")
	flags.StringVar(&jsonPayload, "json", "", "JSON payload for legal entity update")
	flags.StringVar(&formatValue, "format", string(FormatText), "output format")

	if err := flags.Parse(args); err != nil {
		return "", app.PatchLegalEntityCommand{}, "", fmt.Errorf("%s: %w", commandUsage, err)
	}

	if flags.NArg() != 0 {
		return "", app.PatchLegalEntityCommand{}, "", errors.New(commandUsage)
	}

	if id == "" {
		return "", app.PatchLegalEntityCommand{}, "", errors.New("--id is required")
	}

	if jsonPayload == "" {
		return "", app.PatchLegalEntityCommand{}, "", errors.New("--json is required")
	}

	var cmd app.PatchLegalEntityCommand
	if err := json.Unmarshal([]byte(jsonPayload), &cmd); err != nil {
		return "", app.PatchLegalEntityCommand{}, "", formatJSONError(err)
	}

	format, err := ParseFormat(formatValue)
	if err != nil {
		return "", app.PatchLegalEntityCommand{}, "", err
	}

	return id, cmd, format, nil
}

func writeLegalEntityUpdateText(out io.Writer, result app.LegalEntityDTO, colorEnabled bool) error {
	view := newTextView(out, colorEnabled)
	view.Title("Legal Entity Updated").Divider("────────────")
	view.Field("ID", result.ID)
	if result.Type != "" {
		view.Field("Type", result.Type)
	}
	if result.LegalName != "" {
		view.Field("Legal name", result.LegalName)
	}
	if result.TradeName != "" {
		view.Field("Trade name", result.TradeName)
	}
	if result.Email != "" {
		view.Field("Email", result.Email)
	}
	_, err := io.WriteString(out, view.Build())
	return err
}

func parseLegalEntityDeleteFlags(args []string) (string, Format, error) {
	flags := flag.NewFlagSet("legal-entity delete", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	var (
		id          string
		formatValue string
	)

	flags.StringVar(&id, "id", "", "legal entity ID")
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

func writeLegalEntityDeleteText(out io.Writer, id string, colorEnabled bool) error {
	view := newTextView(out, colorEnabled)
	view.Title("Legal Entity Deleted").Divider("────────────")
	view.Field("ID", id)
	_, err := io.WriteString(out, view.Build())
	return err
}

func formatAddress(addr app.AddressDTO) string {
	parts := []string{}
	if addr.Street != "" {
		parts = append(parts, addr.Street)
	}
	if addr.City != "" {
		parts = append(parts, addr.City)
	}
	if addr.State != "" {
		parts = append(parts, addr.State)
	}
	if addr.PostalCode != "" {
		parts = append(parts, addr.PostalCode)
	}
	if addr.Country != "" {
		parts = append(parts, addr.Country)
	}
	return strings.Join(parts, ", ")
}
