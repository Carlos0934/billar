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

type IssuerProfileServiceProvider interface {
	Create(ctx context.Context, cmd app.CreateIssuerProfileCommand) (app.IssuerProfileDTO, error)
	Get(ctx context.Context, id string) (app.IssuerProfileDTO, error)
	Update(ctx context.Context, id string, cmd app.PatchIssuerProfileCommand) (app.IssuerProfileDTO, error)
}

func parseIssuerProfileCreateFlags(args []string) (app.CreateIssuerProfileCommand, Format, error) {
	flags := flag.NewFlagSet("issuer create", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	var (
		jsonPayload string
		formatValue string
	)

	flags.StringVar(&jsonPayload, "json", "", "JSON payload for issuer profile creation")
	flags.StringVar(&formatValue, "format", string(FormatText), "output format")

	if err := flags.Parse(args); err != nil {
		return app.CreateIssuerProfileCommand{}, "", fmt.Errorf("%s: %w", commandUsage, err)
	}

	if flags.NArg() != 0 {
		return app.CreateIssuerProfileCommand{}, "", errors.New(commandUsage)
	}

	if jsonPayload == "" {
		return app.CreateIssuerProfileCommand{}, "", errors.New("--json is required")
	}

	var cmd app.CreateIssuerProfileCommand
	if err := json.Unmarshal([]byte(jsonPayload), &cmd); err != nil {
		return app.CreateIssuerProfileCommand{}, "", formatJSONError(err)
	}

	format, err := ParseFormat(formatValue)
	if err != nil {
		return app.CreateIssuerProfileCommand{}, "", err
	}

	return cmd, format, nil
}

func writeIssuerProfileCreateText(out io.Writer, result app.IssuerProfileDTO, colorEnabled bool) error {
	view := newTextView(out, colorEnabled)
	view.Title("Issuer Profile Created").Divider("────────────")
	view.Field("ID", result.ID)
	view.Field("Legal entity ID", result.LegalEntityID)
	if result.DefaultCurrency != "" {
		view.Field("Default currency", result.DefaultCurrency)
	}
	if result.DefaultNotes != "" {
		view.Field("Default notes", result.DefaultNotes)
	}
	_, err := io.WriteString(out, view.Build())
	return err
}

func parseIssuerProfileGetFlags(args []string) (string, Format, error) {
	flags := flag.NewFlagSet("issuer get", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	var (
		id          string
		formatValue string
	)

	flags.StringVar(&id, "id", "", "issuer profile ID")
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

func writeIssuerProfileGetText(out io.Writer, result app.IssuerProfileDTO, colorEnabled bool) error {
	view := newTextView(out, colorEnabled)
	view.Title("Issuer Profile").Divider("────────────")
	view.Field("ID", result.ID)
	view.Field("Legal entity ID", result.LegalEntityID)
	if result.DefaultCurrency != "" {
		view.Field("Default currency", result.DefaultCurrency)
	}
	if result.DefaultNotes != "" {
		view.Field("Default notes", result.DefaultNotes)
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

func parseIssuerProfileUpdateFlags(args []string) (string, app.PatchIssuerProfileCommand, Format, error) {
	flags := flag.NewFlagSet("issuer update", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	var (
		id          string
		jsonPayload string
		formatValue string
	)

	flags.StringVar(&id, "id", "", "issuer profile ID")
	flags.StringVar(&jsonPayload, "json", "", "JSON payload for issuer profile update")
	flags.StringVar(&formatValue, "format", string(FormatText), "output format")

	if err := flags.Parse(args); err != nil {
		return "", app.PatchIssuerProfileCommand{}, "", fmt.Errorf("%s: %w", commandUsage, err)
	}

	if flags.NArg() != 0 {
		return "", app.PatchIssuerProfileCommand{}, "", errors.New(commandUsage)
	}

	if id == "" {
		return "", app.PatchIssuerProfileCommand{}, "", errors.New("--id is required")
	}

	if jsonPayload == "" {
		return "", app.PatchIssuerProfileCommand{}, "", errors.New("--json is required")
	}

	var cmd app.PatchIssuerProfileCommand
	if err := json.Unmarshal([]byte(jsonPayload), &cmd); err != nil {
		return "", app.PatchIssuerProfileCommand{}, "", formatJSONError(err)
	}

	format, err := ParseFormat(formatValue)
	if err != nil {
		return "", app.PatchIssuerProfileCommand{}, "", err
	}

	return id, cmd, format, nil
}

func writeIssuerProfileUpdateText(out io.Writer, result app.IssuerProfileDTO, colorEnabled bool) error {
	view := newTextView(out, colorEnabled)
	view.Title("Issuer Profile Updated").Divider("────────────")
	view.Field("ID", result.ID)
	view.Field("Legal entity ID", result.LegalEntityID)
	if result.DefaultCurrency != "" {
		view.Field("Default currency", result.DefaultCurrency)
	}
	if result.DefaultNotes != "" {
		view.Field("Default notes", result.DefaultNotes)
	}
	_, err := io.WriteString(out, view.Build())
	return err
}
