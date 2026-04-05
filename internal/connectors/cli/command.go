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
	legalEntity  LegalEntityServiceProvider
	issuer       IssuerProfileServiceProvider
	customer     CustomerProfileServiceProvider
	colorEnabled bool
}

const commandUsage = "usage: billar <health|status|legal-entity <list|create|get|update|delete>|issuer <create|get|update>|customer <list|create|get|update|delete>> [flags]"

func NewCommand(health HealthStatusProvider, legalEntity LegalEntityServiceProvider, issuer IssuerProfileServiceProvider, customer CustomerProfileServiceProvider, colorEnabled bool) Command {
	return Command{
		health:       health,
		legalEntity:  legalEntity,
		issuer:       issuer,
		customer:     customer,
		colorEnabled: colorEnabled,
	}
}

func (c Command) Run(ctx context.Context, args []string, out io.Writer) error {
	if c.health == nil {
		return errors.New("health service is required")
	}

	if len(args) == 0 {
		return errors.New(commandUsage)
	}

	subcommand := strings.ToLower(args[0])
	switch subcommand {
	case "health", "status":
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

	case "legal-entity":
		return c.runLegalEntity(ctx, args[1:], out)

	case "issuer":
		return c.runIssuer(ctx, args[1:], out)

	case "customer":
		return c.runCustomer(ctx, args[1:], out)

	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func (c Command) runLegalEntity(ctx context.Context, args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New(commandUsage)
	}

	subcommand := strings.ToLower(args[0])
	if c.legalEntity == nil {
		return errors.New("legal entity service is required")
	}

	switch subcommand {
	case "list":
		query, format, err := parseLegalEntityListFlags(args[1:])
		if err != nil {
			return err
		}
		query = query.Normalize()

		result, err := c.legalEntity.List(ctx, query)
		if err != nil {
			return fmt.Errorf("run legal-entity list command: %w", err)
		}

		output := OutputResult{
			Payload: result,
			TextWriter: func(w io.Writer) error {
				return writeLegalEntityListText(w, result, c.colorEnabled)
			},
		}

		if err := WriteOutput(out, format, output); err != nil {
			return fmt.Errorf("write legal-entity list output: %w", err)
		}

		return nil

	case "create":
		cmd, format, err := parseLegalEntityCreateFlags(args[1:])
		if err != nil {
			return err
		}

		result, err := c.legalEntity.Create(ctx, cmd)
		if err != nil {
			return fmt.Errorf("run legal-entity create command: %w", err)
		}

		output := OutputResult{
			Payload: result,
			TextWriter: func(w io.Writer) error {
				return writeLegalEntityCreateText(w, result, c.colorEnabled)
			},
		}

		if err := WriteOutput(out, format, output); err != nil {
			return fmt.Errorf("write legal-entity create output: %w", err)
		}

		return nil

	case "get":
		id, format, err := parseLegalEntityGetFlags(args[1:])
		if err != nil {
			return err
		}

		result, err := c.legalEntity.Get(ctx, id)
		if err != nil {
			return fmt.Errorf("run legal-entity get command: %w", err)
		}

		output := OutputResult{
			Payload: result,
			TextWriter: func(w io.Writer) error {
				return writeLegalEntityGetText(w, result, c.colorEnabled)
			},
		}

		if err := WriteOutput(out, format, output); err != nil {
			return fmt.Errorf("write legal-entity get output: %w", err)
		}

		return nil

	case "update":
		id, cmd, format, err := parseLegalEntityUpdateFlags(args[1:])
		if err != nil {
			return err
		}

		result, err := c.legalEntity.Update(ctx, id, cmd)
		if err != nil {
			return fmt.Errorf("run legal-entity update command: %w", err)
		}

		output := OutputResult{
			Payload: result,
			TextWriter: func(w io.Writer) error {
				return writeLegalEntityUpdateText(w, result, c.colorEnabled)
			},
		}

		if err := WriteOutput(out, format, output); err != nil {
			return fmt.Errorf("write legal-entity update output: %w", err)
		}

		return nil

	case "delete":
		id, format, err := parseLegalEntityDeleteFlags(args[1:])
		if err != nil {
			return err
		}

		if err := c.legalEntity.Delete(ctx, id); err != nil {
			return fmt.Errorf("run legal-entity delete command: %w", err)
		}

		output := OutputResult{
			Payload: map[string]string{"id": id, "status": "deleted"},
			TextWriter: func(w io.Writer) error {
				return writeLegalEntityDeleteText(w, id, c.colorEnabled)
			},
		}

		if err := WriteOutput(out, format, output); err != nil {
			return fmt.Errorf("write legal-entity delete output: %w", err)
		}

		return nil

	default:
		return fmt.Errorf("unknown command %q", strings.Join([]string{"legal-entity", args[0]}, " "))
	}
}

func (c Command) runIssuer(ctx context.Context, args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New(commandUsage)
	}

	subcommand := strings.ToLower(args[0])
	if c.issuer == nil {
		return errors.New("issuer service is required")
	}

	switch subcommand {
	case "create":
		cmd, format, err := parseIssuerProfileCreateFlags(args[1:])
		if err != nil {
			return err
		}

		result, err := c.issuer.Create(ctx, cmd)
		if err != nil {
			return fmt.Errorf("run issuer create command: %w", err)
		}

		output := OutputResult{
			Payload: result,
			TextWriter: func(w io.Writer) error {
				return writeIssuerProfileCreateText(w, result, c.colorEnabled)
			},
		}

		if err := WriteOutput(out, format, output); err != nil {
			return fmt.Errorf("write issuer create output: %w", err)
		}

		return nil

	case "get":
		id, format, err := parseIssuerProfileGetFlags(args[1:])
		if err != nil {
			return err
		}

		result, err := c.issuer.Get(ctx, id)
		if err != nil {
			return fmt.Errorf("run issuer get command: %w", err)
		}

		output := OutputResult{
			Payload: result,
			TextWriter: func(w io.Writer) error {
				return writeIssuerProfileGetText(w, result, c.colorEnabled)
			},
		}

		if err := WriteOutput(out, format, output); err != nil {
			return fmt.Errorf("write issuer get output: %w", err)
		}

		return nil

	case "update":
		id, cmd, format, err := parseIssuerProfileUpdateFlags(args[1:])
		if err != nil {
			return err
		}

		result, err := c.issuer.Update(ctx, id, cmd)
		if err != nil {
			return fmt.Errorf("run issuer update command: %w", err)
		}

		output := OutputResult{
			Payload: result,
			TextWriter: func(w io.Writer) error {
				return writeIssuerProfileUpdateText(w, result, c.colorEnabled)
			},
		}

		if err := WriteOutput(out, format, output); err != nil {
			return fmt.Errorf("write issuer update output: %w", err)
		}

		return nil

	default:
		return fmt.Errorf("unknown command %q", strings.Join([]string{"issuer", args[0]}, " "))
	}
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
		query, format, err := parseCustomerProfileListFlags(args[1:])
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
				return writeCustomerProfileListText(w, result, c.colorEnabled)
			},
		}

		if err := WriteOutput(out, format, output); err != nil {
			return fmt.Errorf("write customer list output: %w", err)
		}

		return nil

	case "create":
		cmd, format, err := parseCustomerProfileCreateFlags(args[1:])
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
				return writeCustomerProfileCreateText(w, result, c.colorEnabled)
			},
		}

		if err := WriteOutput(out, format, output); err != nil {
			return fmt.Errorf("write customer create output: %w", err)
		}

		return nil

	case "get":
		id, format, err := parseCustomerProfileGetFlags(args[1:])
		if err != nil {
			return err
		}

		result, err := c.customer.Get(ctx, id)
		if err != nil {
			return fmt.Errorf("run customer get command: %w", err)
		}

		output := OutputResult{
			Payload: result,
			TextWriter: func(w io.Writer) error {
				return writeCustomerProfileGetText(w, result, c.colorEnabled)
			},
		}

		if err := WriteOutput(out, format, output); err != nil {
			return fmt.Errorf("write customer get output: %w", err)
		}

		return nil

	case "update":
		id, cmd, format, err := parseCustomerProfileUpdateFlags(args[1:])
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
				return writeCustomerProfileUpdateText(w, result, c.colorEnabled)
			},
		}

		if err := WriteOutput(out, format, output); err != nil {
			return fmt.Errorf("write customer update output: %w", err)
		}

		return nil

	case "delete":
		id, format, err := parseCustomerProfileDeleteFlags(args[1:])
		if err != nil {
			return err
		}

		if err := c.customer.Delete(ctx, id); err != nil {
			return fmt.Errorf("run customer delete command: %w", err)
		}

		output := OutputResult{
			Payload: map[string]string{"id": id, "status": "deleted"},
			TextWriter: func(w io.Writer) error {
				return writeCustomerProfileDeleteText(w, id, c.colorEnabled)
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
