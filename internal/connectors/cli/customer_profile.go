package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"

	"github.com/Carlos0934/billar/internal/app"
)

type CustomerProfileListProvider interface {
	List(ctx context.Context, query app.ListQuery) (app.ListResult[app.CustomerProfileDTO], error)
}

type CustomerProfileServiceProvider interface {
	CustomerProfileListProvider
	Create(ctx context.Context, cmd app.CreateCustomerProfileCommand) (app.CustomerProfileDTO, error)
	Get(ctx context.Context, id string) (app.CustomerProfileDTO, error)
	Update(ctx context.Context, id string, cmd app.PatchCustomerProfileCommand) (app.CustomerProfileDTO, error)
	Delete(ctx context.Context, id string) error
}

func parseCustomerProfileListFlags(args []string) (app.ListQuery, Format, error) {
	flags := flag.NewFlagSet("customer list", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	var (
		search     string
		sortValue  string
		page       int
		pageSize   int
		formatText string
	)

	flags.StringVar(&search, "search", "", "search by customer profile")
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

func writeCustomerProfileListText(out io.Writer, result app.ListResult[app.CustomerProfileDTO], colorEnabled bool) error {
	view := newTextView(out, colorEnabled)
	view.Title("Customer Profiles").Divider("───────────────").Field("Page", strconv.Itoa(result.Page)).Field("Page size", strconv.Itoa(result.PageSize)).Field("Total", strconv.Itoa(result.Total))

	if len(result.Items) == 0 {
		view.Line("No customer profiles found")
		_, err := io.WriteString(out, view.Build())
		return err
	}

	view.Line("")
	for i, profile := range result.Items {
		if i > 0 {
			view.Line("")
		}
		view.Line(fmt.Sprintf("%d. %s", i+1, profile.ID))
		view.Line("   Legal entity ID: " + profile.LegalEntityID)
		view.Line("   Status: " + profile.Status)
		if profile.DefaultCurrency != "" {
			view.Line("   Default currency: " + profile.DefaultCurrency)
		}
		if profile.Notes != "" {
			view.Line("   Notes: " + profile.Notes)
		}
		if profile.CreatedAt != "" {
			view.Line("   Created at: " + profile.CreatedAt)
		}
		if profile.UpdatedAt != "" {
			view.Line("   Updated at: " + profile.UpdatedAt)
		}
	}

	_, err := io.WriteString(out, view.Build())
	return err
}

func parseCustomerProfileCreateFlags(args []string) (app.CreateCustomerProfileCommand, Format, error) {
	flags := flag.NewFlagSet("customer create", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	var (
		jsonPayload string
		formatValue string
	)

	flags.StringVar(&jsonPayload, "json", "", "JSON payload for customer profile creation")
	flags.StringVar(&formatValue, "format", string(FormatText), "output format")

	if err := flags.Parse(args); err != nil {
		return app.CreateCustomerProfileCommand{}, "", fmt.Errorf("%s: %w", commandUsage, err)
	}

	if flags.NArg() != 0 {
		return app.CreateCustomerProfileCommand{}, "", errors.New(commandUsage)
	}

	if jsonPayload == "" {
		return app.CreateCustomerProfileCommand{}, "", errors.New("--json is required")
	}

	var cmd app.CreateCustomerProfileCommand
	if err := json.Unmarshal([]byte(jsonPayload), &cmd); err != nil {
		return app.CreateCustomerProfileCommand{}, "", formatJSONError(err)
	}

	format, err := ParseFormat(formatValue)
	if err != nil {
		return app.CreateCustomerProfileCommand{}, "", err
	}

	return cmd, format, nil
}

func writeCustomerProfileCreateText(out io.Writer, result app.CustomerProfileDTO, colorEnabled bool) error {
	view := newTextView(out, colorEnabled)
	view.Title("Customer Profile Created").Divider("────────────")
	view.Field("ID", result.ID)
	view.Field("Legal entity ID", result.LegalEntityID)
	if result.Status != "" {
		view.Field("Status", result.Status)
	}
	if result.DefaultCurrency != "" {
		view.Field("Default currency", result.DefaultCurrency)
	}
	_, err := io.WriteString(out, view.Build())
	return err
}

func parseCustomerProfileGetFlags(args []string) (string, Format, error) {
	flags := flag.NewFlagSet("customer get", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	var (
		id          string
		formatValue string
	)

	flags.StringVar(&id, "id", "", "customer profile ID")
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

func writeCustomerProfileGetText(out io.Writer, result app.CustomerProfileDTO, colorEnabled bool) error {
	view := newTextView(out, colorEnabled)
	view.Title("Customer Profile").Divider("────────────")
	view.Field("ID", result.ID)
	view.Field("Legal entity ID", result.LegalEntityID)
	if result.Status != "" {
		view.Field("Status", result.Status)
	}
	if result.DefaultCurrency != "" {
		view.Field("Default currency", result.DefaultCurrency)
	}
	if result.Notes != "" {
		view.Field("Notes", result.Notes)
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

func parseCustomerProfileUpdateFlags(args []string) (string, app.PatchCustomerProfileCommand, Format, error) {
	flags := flag.NewFlagSet("customer update", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	var (
		id          string
		jsonPayload string
		formatValue string
	)

	flags.StringVar(&id, "id", "", "customer profile ID")
	flags.StringVar(&jsonPayload, "json", "", "JSON payload for customer profile update")
	flags.StringVar(&formatValue, "format", string(FormatText), "output format")

	if err := flags.Parse(args); err != nil {
		return "", app.PatchCustomerProfileCommand{}, "", fmt.Errorf("%s: %w", commandUsage, err)
	}

	if flags.NArg() != 0 {
		return "", app.PatchCustomerProfileCommand{}, "", errors.New(commandUsage)
	}

	if id == "" {
		return "", app.PatchCustomerProfileCommand{}, "", errors.New("--id is required")
	}

	if jsonPayload == "" {
		return "", app.PatchCustomerProfileCommand{}, "", errors.New("--json is required")
	}

	var cmd app.PatchCustomerProfileCommand
	if err := json.Unmarshal([]byte(jsonPayload), &cmd); err != nil {
		return "", app.PatchCustomerProfileCommand{}, "", formatJSONError(err)
	}

	format, err := ParseFormat(formatValue)
	if err != nil {
		return "", app.PatchCustomerProfileCommand{}, "", err
	}

	return id, cmd, format, nil
}

func writeCustomerProfileUpdateText(out io.Writer, result app.CustomerProfileDTO, colorEnabled bool) error {
	view := newTextView(out, colorEnabled)
	view.Title("Customer Profile Updated").Divider("────────────")
	view.Field("ID", result.ID)
	view.Field("Legal entity ID", result.LegalEntityID)
	if result.Status != "" {
		view.Field("Status", result.Status)
	}
	if result.DefaultCurrency != "" {
		view.Field("Default currency", result.DefaultCurrency)
	}
	_, err := io.WriteString(out, view.Build())
	return err
}

func parseCustomerProfileDeleteFlags(args []string) (string, Format, error) {
	flags := flag.NewFlagSet("customer delete", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	var (
		id          string
		formatValue string
	)

	flags.StringVar(&id, "id", "", "customer profile ID")
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

func writeCustomerProfileDeleteText(out io.Writer, id string, colorEnabled bool) error {
	view := newTextView(out, colorEnabled)
	view.Title("Customer Profile Deleted").Divider("────────────")
	view.Field("ID", id)
	_, err := io.WriteString(out, view.Build())
	return err
}
