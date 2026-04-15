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
	agreement    AgreementServiceProvider
	timeEntry    TimeEntryServiceProvider
	invoice      InvoiceServiceProvider
	colorEnabled bool
}

const commandUsage = "usage: billar <health|status|legal-entity <list|create|get|update|delete>|issuer <create|get|update>|customer <list|create|get|update|delete>|agreement <create|get|list|update-rate|activate|deactivate>|time-entry <record|get|update|delete|list|list-unbilled>|invoice <draft|issue|discard>> [flags]"

func NewCommand(health HealthStatusProvider, legalEntity LegalEntityServiceProvider, issuer IssuerProfileServiceProvider, customer CustomerProfileServiceProvider, agreement AgreementServiceProvider, timeEntry TimeEntryServiceProvider, invoice InvoiceServiceProvider, colorEnabled bool) Command {
	return Command{
		health:       health,
		legalEntity:  legalEntity,
		issuer:       issuer,
		customer:     customer,
		agreement:    agreement,
		timeEntry:    timeEntry,
		invoice:      invoice,
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

	case "agreement":
		return c.runAgreement(ctx, args[1:], out)

	case "time-entry":
		return c.runTimeEntry(ctx, args[1:], out)

	case "invoice":
		return c.runInvoice(ctx, args[1:], out)

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

func (c Command) runAgreement(ctx context.Context, args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New(commandUsage)
	}

	subcommand := strings.ToLower(args[0])
	if c.agreement == nil {
		return errors.New("agreement service is required")
	}

	switch subcommand {
	case "create":
		cmd, format, err := parseAgreementCreateFlags(args[1:])
		if err != nil {
			return err
		}

		result, err := c.agreement.Create(ctx, cmd)
		if err != nil {
			return fmt.Errorf("run agreement create command: %w", err)
		}

		output := OutputResult{
			Payload: result,
			TextWriter: func(w io.Writer) error {
				return writeAgreementCreateText(w, result, c.colorEnabled)
			},
		}

		if err := WriteOutput(out, format, output); err != nil {
			return fmt.Errorf("write agreement create output: %w", err)
		}

		return nil

	case "get":
		id, format, err := parseAgreementGetFlags(args[1:])
		if err != nil {
			return err
		}

		result, err := c.agreement.Get(ctx, id)
		if err != nil {
			return fmt.Errorf("run agreement get command: %w", err)
		}

		output := OutputResult{
			Payload: result,
			TextWriter: func(w io.Writer) error {
				return writeAgreementGetText(w, result, c.colorEnabled)
			},
		}

		if err := WriteOutput(out, format, output); err != nil {
			return fmt.Errorf("write agreement get output: %w", err)
		}

		return nil

	case "list":
		customerID, format, err := parseAgreementListFlags(args[1:])
		if err != nil {
			return err
		}

		results, err := c.agreement.ListByCustomerProfile(ctx, customerID)
		if err != nil {
			return fmt.Errorf("run agreement list command: %w", err)
		}

		output := OutputResult{
			Payload: results,
			TextWriter: func(w io.Writer) error {
				return writeAgreementListText(w, results, c.colorEnabled)
			},
		}

		if err := WriteOutput(out, format, output); err != nil {
			return fmt.Errorf("write agreement list output: %w", err)
		}

		return nil

	case "update-rate":
		id, cmd, format, err := parseAgreementUpdateRateFlags(args[1:])
		if err != nil {
			return err
		}

		result, err := c.agreement.UpdateRate(ctx, id, cmd)
		if err != nil {
			return fmt.Errorf("run agreement update-rate command: %w", err)
		}

		output := OutputResult{
			Payload: result,
			TextWriter: func(w io.Writer) error {
				return writeAgreementUpdateRateText(w, result, c.colorEnabled)
			},
		}

		if err := WriteOutput(out, format, output); err != nil {
			return fmt.Errorf("write agreement update-rate output: %w", err)
		}

		return nil

	case "activate":
		id, format, err := parseAgreementIDFlags("agreement activate", args[1:])
		if err != nil {
			return err
		}

		result, err := c.agreement.Activate(ctx, id)
		if err != nil {
			return fmt.Errorf("run agreement activate command: %w", err)
		}

		output := OutputResult{
			Payload: result,
			TextWriter: func(w io.Writer) error {
				return writeAgreementActivateText(w, result, c.colorEnabled)
			},
		}

		if err := WriteOutput(out, format, output); err != nil {
			return fmt.Errorf("write agreement activate output: %w", err)
		}

		return nil

	case "deactivate":
		id, format, err := parseAgreementIDFlags("agreement deactivate", args[1:])
		if err != nil {
			return err
		}

		result, err := c.agreement.Deactivate(ctx, id)
		if err != nil {
			return fmt.Errorf("run agreement deactivate command: %w", err)
		}

		output := OutputResult{
			Payload: result,
			TextWriter: func(w io.Writer) error {
				return writeAgreementDeactivateText(w, result, c.colorEnabled)
			},
		}

		if err := WriteOutput(out, format, output); err != nil {
			return fmt.Errorf("write agreement deactivate output: %w", err)
		}

		return nil

	default:
		return fmt.Errorf("unknown command %q", strings.Join([]string{"agreement", args[0]}, " "))
	}
}

func (c Command) runTimeEntry(ctx context.Context, args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New(commandUsage)
	}

	subcommand := strings.ToLower(args[0])
	if c.timeEntry == nil {
		return errors.New("time entry service is required")
	}

	switch subcommand {
	case "record":
		cmd, format, err := parseTimeEntryRecordFlags(args[1:])
		if err != nil {
			return err
		}

		result, err := c.timeEntry.Record(ctx, cmd)
		if err != nil {
			return fmt.Errorf("run time-entry record command: %w", err)
		}

		output := OutputResult{
			Payload: result,
			TextWriter: func(w io.Writer) error {
				return writeTimeEntryRecordText(w, result, c.colorEnabled)
			},
		}

		if err := WriteOutput(out, format, output); err != nil {
			return fmt.Errorf("write time-entry record output: %w", err)
		}

		return nil

	case "get":
		id, format, err := parseTimeEntryGetFlags(args[1:])
		if err != nil {
			return err
		}

		result, err := c.timeEntry.Get(ctx, id)
		if err != nil {
			return fmt.Errorf("run time-entry get command: %w", err)
		}

		output := OutputResult{
			Payload: result,
			TextWriter: func(w io.Writer) error {
				return writeTimeEntryGetText(w, result, c.colorEnabled)
			},
		}

		if err := WriteOutput(out, format, output); err != nil {
			return fmt.Errorf("write time-entry get output: %w", err)
		}

		return nil

	case "update":
		id, cmd, format, err := parseTimeEntryUpdateFlags(args[1:])
		if err != nil {
			return err
		}
		_ = id

		result, err := c.timeEntry.UpdateEntry(ctx, cmd)
		if err != nil {
			return fmt.Errorf("run time-entry update command: %w", err)
		}

		output := OutputResult{
			Payload: result,
			TextWriter: func(w io.Writer) error {
				return writeTimeEntryUpdateText(w, result, c.colorEnabled)
			},
		}

		if err := WriteOutput(out, format, output); err != nil {
			return fmt.Errorf("write time-entry update output: %w", err)
		}

		return nil

	case "delete":
		id, format, err := parseTimeEntryDeleteFlags(args[1:])
		if err != nil {
			return err
		}

		if err := c.timeEntry.DeleteEntry(ctx, id); err != nil {
			return fmt.Errorf("run time-entry delete command: %w", err)
		}

		output := OutputResult{
			Payload: map[string]string{"id": id, "status": "deleted"},
			TextWriter: func(w io.Writer) error {
				return writeTimeEntryDeleteText(w, id, c.colorEnabled)
			},
		}

		if err := WriteOutput(out, format, output); err != nil {
			return fmt.Errorf("write time-entry delete output: %w", err)
		}

		return nil

	case "list":
		customerID, format, err := parseTimeEntryListFlags(args[1:])
		if err != nil {
			return err
		}

		results, err := c.timeEntry.ListByCustomerProfile(ctx, customerID)
		if err != nil {
			return fmt.Errorf("run time-entry list command: %w", err)
		}

		output := OutputResult{
			Payload: results,
			TextWriter: func(w io.Writer) error {
				return writeTimeEntryListText(w, results, c.colorEnabled)
			},
		}

		if err := WriteOutput(out, format, output); err != nil {
			return fmt.Errorf("write time-entry list output: %w", err)
		}

		return nil

	case "list-unbilled":
		customerID, format, err := parseTimeEntryListFlags(args[1:])
		if err != nil {
			return err
		}

		results, err := c.timeEntry.ListUnbilled(ctx, customerID)
		if err != nil {
			return fmt.Errorf("run time-entry list-unbilled command: %w", err)
		}

		output := OutputResult{
			Payload: results,
			TextWriter: func(w io.Writer) error {
				return writeTimeEntryListText(w, results, c.colorEnabled)
			},
		}

		if err := WriteOutput(out, format, output); err != nil {
			return fmt.Errorf("write time-entry list-unbilled output: %w", err)
		}

		return nil

	default:
		return fmt.Errorf("unknown command %q", strings.Join([]string{"time-entry", args[0]}, " "))
	}
}
