package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/Carlos0934/billar/internal/app"
)

// TimeEntryServiceProvider is the seam that CLI commands use to call TimeEntryService operations.
type TimeEntryServiceProvider interface {
	Record(ctx context.Context, cmd app.RecordTimeEntryCommand) (app.TimeEntryDTO, error)
	Get(ctx context.Context, id string) (app.TimeEntryDTO, error)
	UpdateEntry(ctx context.Context, cmd app.UpdateTimeEntryCommand) (app.TimeEntryDTO, error)
	DeleteEntry(ctx context.Context, id string) error
	ListByCustomerProfile(ctx context.Context, customerID string) ([]app.TimeEntryDTO, error)
	ListUnbilled(ctx context.Context, customerID string) ([]app.TimeEntryDTO, error)
}

// -- flag parsers --

// recordTimeEntryInput is the JSON-bound struct for the record subcommand.
type recordTimeEntryInput struct {
	CustomerProfileID  string    `json:"customer_profile_id"`
	ServiceAgreementID string    `json:"service_agreement_id"`
	Description        string    `json:"description"`
	Hours              int64     `json:"hours"`
	Billable           bool      `json:"billable"`
	Date               time.Time `json:"date"`
}

func parseTimeEntryRecordFlags(args []string) (app.RecordTimeEntryCommand, Format, error) {
	flags := flag.NewFlagSet("time-entry record", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	var (
		jsonPayload string
		formatValue string
	)

	flags.StringVar(&jsonPayload, "json", "", "JSON payload for time entry recording")
	flags.StringVar(&formatValue, "format", string(FormatText), "output format")

	if err := flags.Parse(args); err != nil {
		return app.RecordTimeEntryCommand{}, "", fmt.Errorf("%s: %w", commandUsage, err)
	}

	if flags.NArg() != 0 {
		return app.RecordTimeEntryCommand{}, "", errors.New(commandUsage)
	}

	if jsonPayload == "" {
		return app.RecordTimeEntryCommand{}, "", errors.New("--json is required")
	}

	var input recordTimeEntryInput
	if err := json.Unmarshal([]byte(jsonPayload), &input); err != nil {
		return app.RecordTimeEntryCommand{}, "", formatJSONError(err)
	}

	format, err := ParseFormat(formatValue)
	if err != nil {
		return app.RecordTimeEntryCommand{}, "", err
	}

	return app.RecordTimeEntryCommand{
		CustomerProfileID:  input.CustomerProfileID,
		ServiceAgreementID: input.ServiceAgreementID,
		Description:        input.Description,
		Hours:              input.Hours,
		Billable:           input.Billable,
		Date:               input.Date,
	}, format, nil
}

func parseTimeEntryGetFlags(args []string) (string, Format, error) {
	flags := flag.NewFlagSet("time-entry get", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	var (
		id          string
		formatValue string
	)

	flags.StringVar(&id, "id", "", "time entry ID")
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

// updateTimeEntryInput is the JSON-bound struct for the update subcommand.
type updateTimeEntryInput struct {
	Description string `json:"description"`
	Hours       int64  `json:"hours"`
}

func parseTimeEntryUpdateFlags(args []string) (string, app.UpdateTimeEntryCommand, Format, error) {
	flags := flag.NewFlagSet("time-entry update", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	var (
		id          string
		jsonPayload string
		formatValue string
	)

	flags.StringVar(&id, "id", "", "time entry ID")
	flags.StringVar(&jsonPayload, "json", "", "JSON payload for time entry update")
	flags.StringVar(&formatValue, "format", string(FormatText), "output format")

	if err := flags.Parse(args); err != nil {
		return "", app.UpdateTimeEntryCommand{}, "", fmt.Errorf("%s: %w", commandUsage, err)
	}

	if flags.NArg() != 0 {
		return "", app.UpdateTimeEntryCommand{}, "", errors.New(commandUsage)
	}

	if id == "" {
		return "", app.UpdateTimeEntryCommand{}, "", errors.New("--id is required")
	}

	if jsonPayload == "" {
		return "", app.UpdateTimeEntryCommand{}, "", errors.New("--json is required")
	}

	var input updateTimeEntryInput
	if err := json.Unmarshal([]byte(jsonPayload), &input); err != nil {
		return "", app.UpdateTimeEntryCommand{}, "", formatJSONError(err)
	}

	format, err := ParseFormat(formatValue)
	if err != nil {
		return "", app.UpdateTimeEntryCommand{}, "", err
	}

	return id, app.UpdateTimeEntryCommand{
		ID:          id,
		Description: input.Description,
		Hours:       input.Hours,
	}, format, nil
}

func parseTimeEntryDeleteFlags(args []string) (string, Format, error) {
	flags := flag.NewFlagSet("time-entry delete", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	var (
		id          string
		formatValue string
	)

	flags.StringVar(&id, "id", "", "time entry ID")
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

func parseTimeEntryListFlags(args []string) (string, Format, error) {
	flags := flag.NewFlagSet("time-entry list", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	var (
		customerID  string
		formatValue string
	)

	flags.StringVar(&customerID, "customer-id", "", "customer profile ID")
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

// -- text writers --

func writeTimeEntryText(out io.Writer, e app.TimeEntryDTO, colorEnabled bool) error {
	view := newTextView(out, colorEnabled)
	view.Field("ID", e.ID)
	view.Field("Customer profile ID", e.CustomerProfileID)
	if e.ServiceAgreementID != "" {
		view.Field("Service agreement ID", e.ServiceAgreementID)
	}
	view.Field("Description", e.Description)
	view.Field("Hours", fmt.Sprintf("%d", e.Hours))
	view.Field("Billable", fmt.Sprintf("%v", e.Billable))
	if e.InvoiceID != "" {
		view.Field("Invoice ID", e.InvoiceID)
	}
	if e.Date != "" {
		view.Field("Date", e.Date)
	}
	_, err := io.WriteString(out, view.Build())
	return err
}

func writeTimeEntryRecordText(out io.Writer, e app.TimeEntryDTO, colorEnabled bool) error {
	view := newTextView(out, colorEnabled)
	view.Title("Time Entry Recorded").Divider("────────────")
	_, err := io.WriteString(out, view.Build())
	if err != nil {
		return err
	}
	return writeTimeEntryText(out, e, colorEnabled)
}

func writeTimeEntryGetText(out io.Writer, e app.TimeEntryDTO, colorEnabled bool) error {
	view := newTextView(out, colorEnabled)
	view.Title("Time Entry").Divider("────────────")
	_, err := io.WriteString(out, view.Build())
	if err != nil {
		return err
	}
	return writeTimeEntryText(out, e, colorEnabled)
}

func writeTimeEntryUpdateText(out io.Writer, e app.TimeEntryDTO, colorEnabled bool) error {
	view := newTextView(out, colorEnabled)
	view.Title("Time Entry Updated").Divider("────────────")
	_, err := io.WriteString(out, view.Build())
	if err != nil {
		return err
	}
	return writeTimeEntryText(out, e, colorEnabled)
}

func writeTimeEntryDeleteText(out io.Writer, id string, colorEnabled bool) error {
	view := newTextView(out, colorEnabled)
	view.Title("Time Entry Deleted").Divider("────────────")
	view.Field("ID", id)
	view.Field("Status", "deleted")
	_, err := io.WriteString(out, view.Build())
	return err
}

func writeTimeEntryListText(out io.Writer, results []app.TimeEntryDTO, colorEnabled bool) error {
	view := newTextView(out, colorEnabled)
	view.Title(fmt.Sprintf("Time Entries (%d)", len(results))).Divider("────────────")

	if len(results) == 0 {
		view.Line("No time entries found")
		_, err := io.WriteString(out, view.Build())
		return err
	}

	_, err := io.WriteString(out, view.Build())
	if err != nil {
		return err
	}

	for i, e := range results {
		if i > 0 {
			if _, err := fmt.Fprintln(out); err != nil {
				return err
			}
		}
		entryView := newTextView(out, colorEnabled)
		entryView.Line(fmt.Sprintf("%d. %s", i+1, e.ID))
		entryView.Line("   " + e.Description)
		entryView.Line(fmt.Sprintf("   Hours: %d | Billable: %v", e.Hours, e.Billable))
		_, err := io.WriteString(out, entryView.Build())
		if err != nil {
			return err
		}
	}

	return nil
}
