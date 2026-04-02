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

type HealthStatusProvider interface {
	Status(ctx context.Context) (app.HealthDTO, error)
}

type Command struct {
	health       HealthStatusProvider
	colorEnabled bool
}

const commandUsage = "usage: billar <health|status> [--format text|json|toon]"

func NewCommand(health HealthStatusProvider, colorEnabled bool) Command {
	return Command{health: health, colorEnabled: colorEnabled}
}

func (c Command) Run(ctx context.Context, args []string, out io.Writer) error {
	if c.health == nil {
		return errors.New("health service is required")
	}

	if len(args) == 0 {
		return errors.New(commandUsage)
	}

	subcommand := strings.ToLower(args[0])
	if subcommand != "health" && subcommand != "status" {
		return fmt.Errorf("unknown command %q", args[0])
	}

	format, err := parseFormatFlag(subcommand, args[1:])
	if err != nil {
		return err
	}

	status, err := c.health.Status(ctx)
	if err != nil {
		return fmt.Errorf("run %s command: %w", subcommand, err)
	}

	result := OutputResult{
		Payload: status,
		TextWriter: func(w io.Writer) error {
			return writeHealthText(w, status, c.colorEnabled)
		},
	}

	if err := WriteOutput(out, format, result); err != nil {
		return fmt.Errorf("write %s output: %w", subcommand, err)
	}

	return nil
}

func parseFormatFlag(name string, args []string) (Format, error) {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	var formatValue string
	flags.StringVar(&formatValue, "format", string(FormatText), "output format")

	if err := flags.Parse(args); err != nil {
		return "", fmt.Errorf("%s: %w", commandUsage, err)
	}

	if flags.NArg() != 0 {
		return "", errors.New(commandUsage)
	}

	format, err := ParseFormat(formatValue)
	if err != nil {
		return "", err
	}

	return format, nil
}

func writeHealthText(out io.Writer, status app.HealthDTO, colorEnabled bool) error {
	view := newTextView(out, colorEnabled)
	view.Title("Billar Health").Divider("─────────────").Status("Status", status.Status)
	_, err := io.WriteString(out, view.Build())
	return err
}
