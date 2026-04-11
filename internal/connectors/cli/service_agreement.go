package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/Carlos0934/billar/internal/app"
)

// AgreementServiceProvider is the seam that CLI commands use to call AgreementService operations.
type AgreementServiceProvider interface {
	Create(ctx context.Context, cmd app.CreateServiceAgreementCommand) (app.ServiceAgreementDTO, error)
	Get(ctx context.Context, id string) (app.ServiceAgreementDTO, error)
	ListByCustomerProfile(ctx context.Context, profileID string) ([]app.ServiceAgreementDTO, error)
	UpdateRate(ctx context.Context, id string, cmd app.UpdateServiceAgreementRateCommand) (app.ServiceAgreementDTO, error)
	Activate(ctx context.Context, id string) (app.ServiceAgreementDTO, error)
	Deactivate(ctx context.Context, id string) (app.ServiceAgreementDTO, error)
}

func parseAgreementCreateFlags(args []string) (app.CreateServiceAgreementCommand, Format, error) {
	flags := flag.NewFlagSet("agreement create", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	var (
		jsonPayload string
		formatValue string
	)

	flags.StringVar(&jsonPayload, "json", "", "JSON payload for service agreement creation")
	flags.StringVar(&formatValue, "format", string(FormatText), "output format")

	if err := flags.Parse(args); err != nil {
		return app.CreateServiceAgreementCommand{}, "", fmt.Errorf("%s: %w", commandUsage, err)
	}

	if flags.NArg() != 0 {
		return app.CreateServiceAgreementCommand{}, "", errors.New(commandUsage)
	}

	if jsonPayload == "" {
		return app.CreateServiceAgreementCommand{}, "", errors.New("--json is required")
	}

	var cmd app.CreateServiceAgreementCommand
	if err := json.Unmarshal([]byte(jsonPayload), &cmd); err != nil {
		return app.CreateServiceAgreementCommand{}, "", formatJSONError(err)
	}

	format, err := ParseFormat(formatValue)
	if err != nil {
		return app.CreateServiceAgreementCommand{}, "", err
	}

	return cmd, format, nil
}

func writeAgreementCreateText(out io.Writer, result app.ServiceAgreementDTO, colorEnabled bool) error {
	view := newTextView(out, colorEnabled)
	view.Title("Service Agreement Created").Divider("────────────")
	view.Field("ID", result.ID)
	view.Field("Customer profile ID", result.CustomerProfileID)
	view.Field("Name", result.Name)
	if result.Description != "" {
		view.Field("Description", result.Description)
	}
	view.Field("Billing mode", result.BillingMode)
	view.Field("Hourly rate", fmt.Sprintf("%d", result.HourlyRate))
	view.Field("Currency", result.Currency)
	view.Field("Active", fmt.Sprintf("%v", result.Active))
	_, err := io.WriteString(out, view.Build())
	return err
}

func parseAgreementGetFlags(args []string) (string, Format, error) {
	flags := flag.NewFlagSet("agreement get", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	var (
		id          string
		formatValue string
	)

	flags.StringVar(&id, "id", "", "service agreement ID")
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

func writeAgreementGetText(out io.Writer, result app.ServiceAgreementDTO, colorEnabled bool) error {
	view := newTextView(out, colorEnabled)
	view.Title("Service Agreement").Divider("────────────")
	view.Field("ID", result.ID)
	view.Field("Customer profile ID", result.CustomerProfileID)
	view.Field("Name", result.Name)
	if result.Description != "" {
		view.Field("Description", result.Description)
	}
	view.Field("Billing mode", result.BillingMode)
	view.Field("Hourly rate", fmt.Sprintf("%d", result.HourlyRate))
	view.Field("Currency", result.Currency)
	view.Field("Active", fmt.Sprintf("%v", result.Active))
	if result.ValidFrom != nil {
		view.Field("Valid from", *result.ValidFrom)
	}
	if result.ValidUntil != nil {
		view.Field("Valid until", *result.ValidUntil)
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

func parseAgreementListFlags(args []string) (string, Format, error) {
	flags := flag.NewFlagSet("agreement list", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	var (
		customerID  string
		formatValue string
	)

	flags.StringVar(&customerID, "customer-id", "", "customer profile ID to list agreements for")
	flags.StringVar(&formatValue, "format", string(FormatText), "output format")

	if err := flags.Parse(args); err != nil {
		return "", "", fmt.Errorf("%s: %w", commandUsage, err)
	}

	if flags.NArg() != 0 {
		return "", "", errors.New(commandUsage)
	}

	if customerID == "" {
		return "", "", errors.New("--customer-id is required")
	}

	format, err := ParseFormat(formatValue)
	if err != nil {
		return "", "", err
	}

	return customerID, format, nil
}

func writeAgreementListText(out io.Writer, results []app.ServiceAgreementDTO, colorEnabled bool) error {
	view := newTextView(out, colorEnabled)
	view.Title(fmt.Sprintf("Service Agreements (%d)", len(results))).Divider("────────────")

	if len(results) == 0 {
		view.Line("No service agreements found")
		_, err := io.WriteString(out, view.Build())
		return err
	}

	for i, sa := range results {
		if i > 0 {
			view.Line("")
		}
		view.Line(fmt.Sprintf("%d. %s", i+1, sa.ID))
		view.Line("   Name: " + sa.Name)
		view.Line(fmt.Sprintf("   Billing mode: %s | Rate: %d %s", sa.BillingMode, sa.HourlyRate, sa.Currency))
		view.Line(fmt.Sprintf("   Active: %v", sa.Active))
	}

	_, err := io.WriteString(out, view.Build())
	return err
}

func parseAgreementUpdateRateFlags(args []string) (string, app.UpdateServiceAgreementRateCommand, Format, error) {
	flags := flag.NewFlagSet("agreement update-rate", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	var (
		id          string
		jsonPayload string
		formatValue string
	)

	flags.StringVar(&id, "id", "", "service agreement ID")
	flags.StringVar(&jsonPayload, "json", "", "JSON payload for rate update")
	flags.StringVar(&formatValue, "format", string(FormatText), "output format")

	if err := flags.Parse(args); err != nil {
		return "", app.UpdateServiceAgreementRateCommand{}, "", fmt.Errorf("%s: %w", commandUsage, err)
	}

	if flags.NArg() != 0 {
		return "", app.UpdateServiceAgreementRateCommand{}, "", errors.New(commandUsage)
	}

	if id == "" {
		return "", app.UpdateServiceAgreementRateCommand{}, "", errors.New("--id is required")
	}

	if jsonPayload == "" {
		return "", app.UpdateServiceAgreementRateCommand{}, "", errors.New("--json is required")
	}

	var cmd app.UpdateServiceAgreementRateCommand
	if err := json.Unmarshal([]byte(jsonPayload), &cmd); err != nil {
		return "", app.UpdateServiceAgreementRateCommand{}, "", formatJSONError(err)
	}

	format, err := ParseFormat(formatValue)
	if err != nil {
		return "", app.UpdateServiceAgreementRateCommand{}, "", err
	}

	return id, cmd, format, nil
}

func writeAgreementUpdateRateText(out io.Writer, result app.ServiceAgreementDTO, colorEnabled bool) error {
	view := newTextView(out, colorEnabled)
	view.Title("Service Agreement Rate Updated").Divider("────────────")
	view.Field("ID", result.ID)
	view.Field("Hourly rate", fmt.Sprintf("%d", result.HourlyRate))
	view.Field("Currency", result.Currency)
	view.Field("Active", fmt.Sprintf("%v", result.Active))
	_, err := io.WriteString(out, view.Build())
	return err
}

func parseAgreementIDFlags(name string, args []string) (string, Format, error) {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	var (
		id          string
		formatValue string
	)

	flags.StringVar(&id, "id", "", "service agreement ID")
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

func writeAgreementActivateText(out io.Writer, result app.ServiceAgreementDTO, colorEnabled bool) error {
	view := newTextView(out, colorEnabled)
	view.Title("Service Agreement Activated").Divider("────────────")
	view.Field("ID", result.ID)
	view.Field("Active", fmt.Sprintf("%v", result.Active))
	_, err := io.WriteString(out, view.Build())
	return err
}

func writeAgreementDeactivateText(out io.Writer, result app.ServiceAgreementDTO, colorEnabled bool) error {
	view := newTextView(out, colorEnabled)
	view.Title("Service Agreement Deactivated").Divider("────────────")
	view.Field("ID", result.ID)
	view.Field("Active", fmt.Sprintf("%v", result.Active))
	_, err := io.WriteString(out, view.Build())
	return err
}
