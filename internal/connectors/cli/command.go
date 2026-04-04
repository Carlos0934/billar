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

type CustomerListProvider interface {
	List(ctx context.Context, query app.ListQuery) (app.ListResult[app.CustomerDTO], error)
}

type CustomerServiceProvider interface {
	CustomerListProvider
	Create(ctx context.Context, cmd app.CreateCustomerCommand) (app.CustomerDTO, error)
	Update(ctx context.Context, id string, cmd app.PatchCustomerCommand) (app.CustomerDTO, error)
	Delete(ctx context.Context, id string) error
}

type Command struct {
	health       HealthStatusProvider
	customer     CustomerServiceProvider
	colorEnabled bool
}

const commandUsage = "usage: billar <health|status|customer list|customer create|customer update|customer delete> [flags]"

func NewCommand(health HealthStatusProvider, customer CustomerServiceProvider, colorEnabled bool) Command {
	return Command{health: health, customer: customer, colorEnabled: colorEnabled}
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
		if subcommand != "customer" {
			return fmt.Errorf("unknown command %q", args[0])
		}
		return c.runCustomer(ctx, args[1:], out)
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

func (c Command) runCustomer(ctx context.Context, args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New(commandUsage)
	}

	subcommand := strings.ToLower(args[0])
	if c.customer == nil {
		return errors.New("customer service is required")
	}

	switch subcommand {
	case "list":
		query, format, err := parseCustomerListFlags(args[1:])
		if err != nil {
			return err
		}
		query = query.Normalize()

		result, err := c.customer.List(ctx, query)
		if err != nil {
			return fmt.Errorf("run customer list command: %w", err)
		}

		output := OutputResult{
			Payload: result,
			TextWriter: func(w io.Writer) error {
				return writeCustomerListText(w, result, c.colorEnabled)
			},
		}

		if err := WriteOutput(out, format, output); err != nil {
			return fmt.Errorf("write customer list output: %w", err)
		}

		return nil

	case "create":
		cmd, format, err := parseCustomerCreateFlags(args[1:])
		if err != nil {
			return err
		}

		result, err := c.customer.Create(ctx, cmd)
		if err != nil {
			return fmt.Errorf("run customer create command: %w", err)
		}

		output := OutputResult{
			Payload: result,
			TextWriter: func(w io.Writer) error {
				return writeCustomerCreateText(w, result, c.colorEnabled)
			},
		}

		if err := WriteOutput(out, format, output); err != nil {
			return fmt.Errorf("write customer create output: %w", err)
		}

		return nil

	case "update":
		id, cmd, format, err := parseCustomerUpdateFlags(args[1:])
		if err != nil {
			return err
		}

		result, err := c.customer.Update(ctx, id, cmd)
		if err != nil {
			return fmt.Errorf("run customer update command: %w", err)
		}

		output := OutputResult{
			Payload: result,
			TextWriter: func(w io.Writer) error {
				return writeCustomerUpdateText(w, result, c.colorEnabled)
			},
		}

		if err := WriteOutput(out, format, output); err != nil {
			return fmt.Errorf("write customer update output: %w", err)
		}

		return nil

	case "delete":
		id, format, err := parseCustomerDeleteFlags(args[1:])
		if err != nil {
			return err
		}

		if err := c.customer.Delete(ctx, id); err != nil {
			return fmt.Errorf("run customer delete command: %w", err)
		}

		output := OutputResult{
			Payload: map[string]string{"id": id, "status": "deleted"},
			TextWriter: func(w io.Writer) error {
				return writeCustomerDeleteText(w, id, c.colorEnabled)
			},
		}

		if err := WriteOutput(out, format, output); err != nil {
			return fmt.Errorf("write customer delete output: %w", err)
		}

		return nil

	default:
		return fmt.Errorf("unknown command %q", strings.Join([]string{"customer", args[0]}, " "))
	}
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
