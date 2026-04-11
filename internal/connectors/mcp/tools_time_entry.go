package mcp

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Carlos0934/billar/internal/app"
	"github.com/Carlos0934/billar/internal/infra/logging"
	"github.com/mark3labs/mcp-go/mcp"
	mcpsrv "github.com/mark3labs/mcp-go/server"
)

// TimeEntryServiceProvider is the seam that MCP tools use to call TimeEntryService operations.
type TimeEntryServiceProvider interface {
	Record(ctx context.Context, cmd app.RecordTimeEntryCommand) (app.TimeEntryDTO, error)
	Get(ctx context.Context, id string) (app.TimeEntryDTO, error)
	UpdateEntry(ctx context.Context, cmd app.UpdateTimeEntryCommand) (app.TimeEntryDTO, error)
	DeleteEntry(ctx context.Context, id string) error
	ListByCustomerProfile(ctx context.Context, customerID string) ([]app.TimeEntryDTO, error)
	ListUnbilled(ctx context.Context, customerID string) ([]app.TimeEntryDTO, error)
}

func registerTimeEntryTools(server *mcpsrv.MCPServer, service TimeEntryServiceProvider, guard IngressGuard, logger *slog.Logger) []string {
	registered := make([]string, 0, 6)

	tool, handler := timeEntryRecordTool(service, guard, logger)
	server.AddTool(tool, handler)
	registered = append(registered, tool.Name)

	tool, handler = timeEntryGetTool(service, guard, logger)
	server.AddTool(tool, handler)
	registered = append(registered, tool.Name)

	tool, handler = timeEntryUpdateTool(service, guard, logger)
	server.AddTool(tool, handler)
	registered = append(registered, tool.Name)

	tool, handler = timeEntryDeleteTool(service, guard, logger)
	server.AddTool(tool, handler)
	registered = append(registered, tool.Name)

	tool, handler = timeEntryListTool(service, guard, logger)
	server.AddTool(tool, handler)
	registered = append(registered, tool.Name)

	tool, handler = timeEntryListUnbilledTool(service, guard, logger)
	server.AddTool(tool, handler)
	registered = append(registered, tool.Name)

	return registered
}

func timeEntryRecordTool(service TimeEntryServiceProvider, guard IngressGuard, logger *slog.Logger) (mcp.Tool, func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
	tool := mcp.NewTool("time_entry.record",
		mcp.WithDescription(`Record a new time entry for a customer.

REQUIRED FIELDS:
- customer_profile_id: The customer profile this entry belongs to
- service_agreement_id: The service agreement this entry is billed under
- description: Description of the work performed
- hours: Duration in minor time units (minutes)
- date: Date the work was performed (RFC3339)`),
		mcp.WithString("customer_profile_id",
			mcp.Required(),
			mcp.Description("Customer profile ID (e.g., 'cus_123')"),
		),
		mcp.WithString("service_agreement_id",
			mcp.Required(),
			mcp.Description("Service agreement ID (e.g., 'sa_123')"),
		),
		mcp.WithString("description",
			mcp.Required(),
			mcp.Description("Description of the work performed"),
		),
		mcp.WithNumber("hours",
			mcp.Required(),
			mcp.Description("Duration in minutes"),
		),
		mcp.WithBoolean("billable",
			mcp.Description("Whether this entry is billable (default: true)"),
		),
		mcp.WithString("date",
			mcp.Required(),
			mcp.Description("Date the work was performed (RFC3339, e.g., '2026-04-10T00:00:00Z')"),
		),
	)

	return tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if service == nil {
			return mcp.NewToolResultError("time entry service is required"), nil
		}
		if err := guard.authorize(req.Header); err != nil {
			logging.Event(ctx, logger, slog.LevelWarn, "time_entry.record", "mcp", "denied", slog.String("reason", classifyMCPAuthReason(err)))
			return mcp.NewToolResultError(err.Error()), nil
		}

		customerProfileID := strings.TrimSpace(req.GetString("customer_profile_id", ""))
		if customerProfileID == "" {
			return mcp.NewToolResultError("customer_profile_id argument is required"), nil
		}

		serviceAgreementID := strings.TrimSpace(req.GetString("service_agreement_id", ""))
		if serviceAgreementID == "" {
			return mcp.NewToolResultError("service_agreement_id argument is required"), nil
		}

		description := strings.TrimSpace(req.GetString("description", ""))
		if description == "" {
			return mcp.NewToolResultError("description argument is required"), nil
		}

		hours := int64(req.GetFloat("hours", 0))

		billable := req.GetBool("billable", true)

		dateStr := strings.TrimSpace(req.GetString("date", ""))
		var date time.Time
		if dateStr != "" {
			var err error
			date, err = time.Parse(time.RFC3339, dateStr)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid date format, expected RFC3339: %s", err.Error())), nil
			}
		}

		cmd := app.RecordTimeEntryCommand{
			CustomerProfileID:  customerProfileID,
			ServiceAgreementID: serviceAgreementID,
			Description:        description,
			Hours:              hours,
			Billable:           billable,
			Date:               date,
		}

		result, err := service.Record(ctx, cmd)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(timeEntryRecordText(result)), nil
	}
}

func timeEntryGetTool(service TimeEntryServiceProvider, guard IngressGuard, logger *slog.Logger) (mcp.Tool, func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
	tool := mcp.NewTool("time_entry.get",
		mcp.WithDescription("Get a time entry by ID"),
		mcp.WithString("id",
			mcp.Required(),
			mcp.Description("Time entry ID (e.g., 'te_123')"),
		),
	)

	return tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if service == nil {
			return mcp.NewToolResultError("time entry service is required"), nil
		}
		if err := guard.authorize(req.Header); err != nil {
			logging.Event(ctx, logger, slog.LevelWarn, "time_entry.get", "mcp", "denied", slog.String("reason", classifyMCPAuthReason(err)))
			return mcp.NewToolResultError(err.Error()), nil
		}

		id := strings.TrimSpace(req.GetString("id", ""))
		if id == "" {
			return mcp.NewToolResultError("id argument is required"), nil
		}

		result, err := service.Get(ctx, id)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(timeEntryGetText(result)), nil
	}
}

func timeEntryUpdateTool(service TimeEntryServiceProvider, guard IngressGuard, logger *slog.Logger) (mcp.Tool, func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
	tool := mcp.NewTool("time_entry.update",
		mcp.WithDescription("Update the description and/or hours of an existing time entry"),
		mcp.WithString("id",
			mcp.Required(),
			mcp.Description("Time entry ID to update (e.g., 'te_123')"),
		),
		mcp.WithString("description",
			mcp.Description("Updated description of the work performed"),
		),
		mcp.WithNumber("hours",
			mcp.Description("Updated duration in minutes"),
		),
	)

	return tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if service == nil {
			return mcp.NewToolResultError("time entry service is required"), nil
		}
		if err := guard.authorize(req.Header); err != nil {
			logging.Event(ctx, logger, slog.LevelWarn, "time_entry.update", "mcp", "denied", slog.String("reason", classifyMCPAuthReason(err)))
			return mcp.NewToolResultError(err.Error()), nil
		}

		var input UpdateTimeEntryInput
		if err := req.BindArguments(&input); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		cmd := input.toCommand()
		if cmd.ID == "" {
			return mcp.NewToolResultError("id argument is required"), nil
		}

		result, err := service.UpdateEntry(ctx, cmd)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(timeEntryUpdateText(result)), nil
	}
}

func timeEntryDeleteTool(service TimeEntryServiceProvider, guard IngressGuard, logger *slog.Logger) (mcp.Tool, func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
	tool := mcp.NewTool("time_entry.delete",
		mcp.WithDescription("Delete a time entry by ID"),
		mcp.WithString("id",
			mcp.Required(),
			mcp.Description("Time entry ID to delete (e.g., 'te_123')"),
		),
	)

	return tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if service == nil {
			return mcp.NewToolResultError("time entry service is required"), nil
		}
		if err := guard.authorize(req.Header); err != nil {
			logging.Event(ctx, logger, slog.LevelWarn, "time_entry.delete", "mcp", "denied", slog.String("reason", classifyMCPAuthReason(err)))
			return mcp.NewToolResultError(err.Error()), nil
		}

		id := strings.TrimSpace(req.GetString("id", ""))
		if id == "" {
			return mcp.NewToolResultError("id argument is required"), nil
		}

		if err := service.DeleteEntry(ctx, id); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Time entry deleted: %s\n", id)), nil
	}
}

func timeEntryListTool(service TimeEntryServiceProvider, guard IngressGuard, logger *slog.Logger) (mcp.Tool, func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
	tool := mcp.NewTool("time_entry.list_by_customer_profile",
		mcp.WithDescription("List all time entries for a given customer profile"),
		mcp.WithString("customer_profile_id",
			mcp.Required(),
			mcp.Description("Customer profile ID to list entries for (e.g., 'cus_123')"),
		),
	)

	return tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if service == nil {
			return mcp.NewToolResultError("time entry service is required"), nil
		}
		if err := guard.authorize(req.Header); err != nil {
			logging.Event(ctx, logger, slog.LevelWarn, "time_entry.list_by_customer_profile", "mcp", "denied", slog.String("reason", classifyMCPAuthReason(err)))
			return mcp.NewToolResultError(err.Error()), nil
		}

		customerProfileID := strings.TrimSpace(req.GetString("customer_profile_id", ""))
		if customerProfileID == "" {
			return mcp.NewToolResultError("customer_profile_id argument is required"), nil
		}

		results, err := service.ListByCustomerProfile(ctx, customerProfileID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(timeEntryListText(results)), nil
	}
}

func timeEntryListUnbilledTool(service TimeEntryServiceProvider, guard IngressGuard, logger *slog.Logger) (mcp.Tool, func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
	tool := mcp.NewTool("time_entry.list_unbilled",
		mcp.WithDescription("List all unbilled time entries for a given customer profile"),
		mcp.WithString("customer_profile_id",
			mcp.Required(),
			mcp.Description("Customer profile ID to list unbilled entries for (e.g., 'cus_123')"),
		),
	)

	return tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if service == nil {
			return mcp.NewToolResultError("time entry service is required"), nil
		}
		if err := guard.authorize(req.Header); err != nil {
			logging.Event(ctx, logger, slog.LevelWarn, "time_entry.list_unbilled", "mcp", "denied", slog.String("reason", classifyMCPAuthReason(err)))
			return mcp.NewToolResultError(err.Error()), nil
		}

		customerProfileID := strings.TrimSpace(req.GetString("customer_profile_id", ""))
		if customerProfileID == "" {
			return mcp.NewToolResultError("customer_profile_id argument is required"), nil
		}

		results, err := service.ListUnbilled(ctx, customerProfileID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(timeEntryListUnbilledText(results)), nil
	}
}

// -- text helpers --

func timeEntryText(te app.TimeEntryDTO) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("ID: %s\n", te.ID))
	b.WriteString(fmt.Sprintf("Customer profile ID: %s\n", te.CustomerProfileID))
	if te.ServiceAgreementID != "" {
		b.WriteString(fmt.Sprintf("Service agreement ID: %s\n", te.ServiceAgreementID))
	}
	b.WriteString(fmt.Sprintf("Description: %s\n", te.Description))
	b.WriteString(fmt.Sprintf("Hours: %d\n", te.Hours))
	b.WriteString(fmt.Sprintf("Billable: %v\n", te.Billable))
	if te.InvoiceID != "" {
		b.WriteString(fmt.Sprintf("Invoice ID: %s\n", te.InvoiceID))
	}
	if te.Date != "" {
		b.WriteString(fmt.Sprintf("Date: %s\n", te.Date))
	}
	if te.CreatedAt != "" {
		b.WriteString(fmt.Sprintf("Created at: %s\n", te.CreatedAt))
	}
	if te.UpdatedAt != "" {
		b.WriteString(fmt.Sprintf("Updated at: %s\n", te.UpdatedAt))
	}
	return b.String()
}

func timeEntryRecordText(te app.TimeEntryDTO) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Time entry recorded: %s\n", te.ID))
	b.WriteString(timeEntryText(te))
	return b.String()
}

func timeEntryGetText(te app.TimeEntryDTO) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Time entry: %s\n", te.ID))
	b.WriteString(timeEntryText(te))
	return b.String()
}

func timeEntryUpdateText(te app.TimeEntryDTO) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Time entry updated: %s\n", te.ID))
	b.WriteString(timeEntryText(te))
	return b.String()
}

func timeEntryListText(items []app.TimeEntryDTO) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Time entries (%d)\n", len(items)))
	b.WriteString("───────────────\n")
	if len(items) == 0 {
		b.WriteString("No time entries found\n")
		return b.String()
	}
	for i, te := range items {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(fmt.Sprintf("%d. %s\n", i+1, te.ID))
		b.WriteString(timeEntryText(te))
	}
	return b.String()
}

func timeEntryListUnbilledText(items []app.TimeEntryDTO) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Unbilled time entries (%d)\n", len(items)))
	b.WriteString("───────────────\n")
	if len(items) == 0 {
		b.WriteString("No unbilled time entries found\n")
		return b.String()
	}
	for i, te := range items {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(fmt.Sprintf("%d. %s\n", i+1, te.ID))
		b.WriteString(timeEntryText(te))
	}
	return b.String()
}
