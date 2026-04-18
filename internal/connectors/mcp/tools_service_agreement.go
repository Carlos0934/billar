package mcp

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/Carlos0934/billar/internal/app"
	"github.com/mark3labs/mcp-go/mcp"
	mcpsrv "github.com/mark3labs/mcp-go/server"
)

// AgreementServiceProvider is the seam that MCP tools use to call AgreementService operations.
type AgreementServiceProvider interface {
	Create(ctx context.Context, cmd app.CreateServiceAgreementCommand) (app.ServiceAgreementDTO, error)
	Get(ctx context.Context, id string) (app.ServiceAgreementDTO, error)
	ListByCustomerProfile(ctx context.Context, profileID string) ([]app.ServiceAgreementDTO, error)
	UpdateRate(ctx context.Context, id string, cmd app.UpdateServiceAgreementRateCommand) (app.ServiceAgreementDTO, error)
	Activate(ctx context.Context, id string) (app.ServiceAgreementDTO, error)
	Deactivate(ctx context.Context, id string) (app.ServiceAgreementDTO, error)
}

func registerServiceAgreementTools(server *mcpsrv.MCPServer, service AgreementServiceProvider, logger *slog.Logger) []string {
	registered := make([]string, 0, 6)

	tool, handler := serviceAgreementCreateTool(service, logger)
	server.AddTool(tool, handler)
	registered = append(registered, tool.Name)

	tool, handler = serviceAgreementGetTool(service, logger)
	server.AddTool(tool, handler)
	registered = append(registered, tool.Name)

	tool, handler = serviceAgreementListTool(service, logger)
	server.AddTool(tool, handler)
	registered = append(registered, tool.Name)

	tool, handler = serviceAgreementUpdateRateTool(service, logger)
	server.AddTool(tool, handler)
	registered = append(registered, tool.Name)

	tool, handler = serviceAgreementActivateTool(service, logger)
	server.AddTool(tool, handler)
	registered = append(registered, tool.Name)

	tool, handler = serviceAgreementDeactivateTool(service, logger)
	server.AddTool(tool, handler)
	registered = append(registered, tool.Name)

	return registered
}

func serviceAgreementCreateTool(service AgreementServiceProvider, logger *slog.Logger) (mcp.Tool, func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
	tool := mcp.NewTool("service_agreement.create",
		mcp.WithDescription(`Create a new service agreement for a customer profile.

A service agreement defines the billing terms (mode, hourly rate, currency) for a customer.

REQUIRED FIELDS:
- customer_profile_id: The customer profile this agreement belongs to
- name: A descriptive name for the agreement
- billing_mode: Billing mode (e.g., "hourly")
- hourly_rate: Rate in minor currency units (e.g., cents)
- currency: ISO 4217 currency code (e.g., "USD", "DOP")`),
		mcp.WithString("customer_profile_id",
			mcp.Required(),
			mcp.Description("Customer profile ID this agreement belongs to (e.g., 'cus_123')"),
		),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Descriptive name for the agreement"),
		),
		mcp.WithString("description",
			mcp.Description("Optional description of the agreement"),
		),
		mcp.WithString("billing_mode",
			mcp.Required(),
			mcp.Description("Billing mode (e.g., 'hourly')"),
		),
		mcp.WithNumber("hourly_rate",
			mcp.Required(),
			mcp.Description("Hourly rate in minor currency units (e.g., cents)"),
		),
		mcp.WithString("currency",
			mcp.Required(),
			mcp.Description("ISO 4217 currency code (e.g., 'USD', 'DOP')"),
		),
	)
	return tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if service == nil {
			return mcp.NewToolResultError("service agreement service is required"), nil
		}

		var input ServiceAgreementCreateInput
		if err := req.BindArguments(&input); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		if strings.TrimSpace(input.CustomerProfileID) == "" {
			return mcp.NewToolResultError("customer_profile_id argument is required"), nil
		}

		result, err := service.Create(ctx, input.toCommand())
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(serviceAgreementCreateText(result)), nil
	}
}

func serviceAgreementGetTool(service AgreementServiceProvider, logger *slog.Logger) (mcp.Tool, func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
	tool := mcp.NewTool("service_agreement.get",
		mcp.WithDescription("Get a service agreement by ID"),
		mcp.WithString("id",
			mcp.Required(),
			mcp.Description("Service agreement ID (e.g., 'sa_123')"),
		),
	)
	return tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if service == nil {
			return mcp.NewToolResultError("service agreement service is required"), nil
		}

		id := strings.TrimSpace(req.GetString("id", ""))
		if id == "" {
			return mcp.NewToolResultError("id argument is required"), nil
		}

		result, err := service.Get(ctx, id)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(serviceAgreementGetText(result)), nil
	}
}

func serviceAgreementListTool(service AgreementServiceProvider, logger *slog.Logger) (mcp.Tool, func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
	tool := mcp.NewTool("service_agreement.list_by_customer_profile",
		mcp.WithDescription("List all service agreements for a given customer profile"),
		mcp.WithString("customer_profile_id",
			mcp.Required(),
			mcp.Description("Customer profile ID to list agreements for (e.g., 'cus_123')"),
		),
	)
	return tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if service == nil {
			return mcp.NewToolResultError("service agreement service is required"), nil
		}

		customerProfileID := strings.TrimSpace(req.GetString("customer_profile_id", ""))
		if customerProfileID == "" {
			return mcp.NewToolResultError("customer_profile_id argument is required"), nil
		}

		results, err := service.ListByCustomerProfile(ctx, customerProfileID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(serviceAgreementListText(results)), nil
	}
}

func serviceAgreementUpdateRateTool(service AgreementServiceProvider, logger *slog.Logger) (mcp.Tool, func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
	tool := mcp.NewTool("service_agreement.update_rate",
		mcp.WithDescription("Update the hourly rate of an existing service agreement"),
		mcp.WithString("id",
			mcp.Required(),
			mcp.Description("Service agreement ID to update (e.g., 'sa_123')"),
		),
		mcp.WithNumber("hourly_rate",
			mcp.Required(),
			mcp.Description("New hourly rate in minor currency units (e.g., cents)"),
		),
	)
	return tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if service == nil {
			return mcp.NewToolResultError("service agreement service is required"), nil
		}

		var input ServiceAgreementUpdateRateInput
		if err := req.BindArguments(&input); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		id, cmd := input.toCommand()
		if id == "" {
			return mcp.NewToolResultError("id argument is required"), nil
		}

		result, err := service.UpdateRate(ctx, id, cmd)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(serviceAgreementUpdateRateText(result)), nil
	}
}

func serviceAgreementActivateTool(service AgreementServiceProvider, logger *slog.Logger) (mcp.Tool, func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
	tool := mcp.NewTool("service_agreement.activate",
		mcp.WithDescription("Activate a service agreement"),
		mcp.WithString("id",
			mcp.Required(),
			mcp.Description("Service agreement ID to activate (e.g., 'sa_123')"),
		),
	)
	return tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if service == nil {
			return mcp.NewToolResultError("service agreement service is required"), nil
		}

		id := strings.TrimSpace(req.GetString("id", ""))
		if id == "" {
			return mcp.NewToolResultError("id argument is required"), nil
		}

		result, err := service.Activate(ctx, id)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(serviceAgreementActivateText(result)), nil
	}
}

func serviceAgreementDeactivateTool(service AgreementServiceProvider, logger *slog.Logger) (mcp.Tool, func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
	tool := mcp.NewTool("service_agreement.deactivate",
		mcp.WithDescription("Deactivate a service agreement"),
		mcp.WithString("id",
			mcp.Required(),
			mcp.Description("Service agreement ID to deactivate (e.g., 'sa_123')"),
		),
	)
	return tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if service == nil {
			return mcp.NewToolResultError("service agreement service is required"), nil
		}

		id := strings.TrimSpace(req.GetString("id", ""))
		if id == "" {
			return mcp.NewToolResultError("id argument is required"), nil
		}

		result, err := service.Deactivate(ctx, id)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(serviceAgreementDeactivateText(result)), nil
	}
}

// -- text helpers --

func serviceAgreementText(sa app.ServiceAgreementDTO) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("ID: %s\n", sa.ID))
	b.WriteString(fmt.Sprintf("Customer profile ID: %s\n", sa.CustomerProfileID))
	b.WriteString(fmt.Sprintf("Name: %s\n", sa.Name))
	if sa.Description != "" {
		b.WriteString(fmt.Sprintf("Description: %s\n", sa.Description))
	}
	b.WriteString(fmt.Sprintf("Billing mode: %s\n", sa.BillingMode))
	b.WriteString(fmt.Sprintf("Hourly rate: %d\n", sa.HourlyRate))
	b.WriteString(fmt.Sprintf("Currency: %s\n", sa.Currency))
	b.WriteString(fmt.Sprintf("Active: %v\n", sa.Active))
	if sa.ValidFrom != nil {
		b.WriteString(fmt.Sprintf("Valid from: %s\n", *sa.ValidFrom))
	}
	if sa.ValidUntil != nil {
		b.WriteString(fmt.Sprintf("Valid until: %s\n", *sa.ValidUntil))
	}
	if sa.CreatedAt != "" {
		b.WriteString(fmt.Sprintf("Created at: %s\n", sa.CreatedAt))
	}
	if sa.UpdatedAt != "" {
		b.WriteString(fmt.Sprintf("Updated at: %s\n", sa.UpdatedAt))
	}
	return b.String()
}

func serviceAgreementCreateText(sa app.ServiceAgreementDTO) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Service agreement created: %s\n", sa.ID))
	b.WriteString(serviceAgreementText(sa))
	return b.String()
}

func serviceAgreementGetText(sa app.ServiceAgreementDTO) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Service agreement: %s\n", sa.ID))
	b.WriteString(serviceAgreementText(sa))
	return b.String()
}

func serviceAgreementListText(items []app.ServiceAgreementDTO) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Service agreements (%d)\n", len(items)))
	b.WriteString("───────────────\n")
	if len(items) == 0 {
		b.WriteString("No service agreements found\n")
		return b.String()
	}
	for i, sa := range items {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(fmt.Sprintf("%d. %s\n", i+1, sa.ID))
		b.WriteString(serviceAgreementText(sa))
	}
	return b.String()
}

func serviceAgreementUpdateRateText(sa app.ServiceAgreementDTO) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Service agreement rate updated: %s\n", sa.ID))
	b.WriteString(serviceAgreementText(sa))
	return b.String()
}

func serviceAgreementActivateText(sa app.ServiceAgreementDTO) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Service agreement activated: %s\n", sa.ID))
	b.WriteString(serviceAgreementText(sa))
	return b.String()
}

func serviceAgreementDeactivateText(sa app.ServiceAgreementDTO) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Service agreement deactivated: %s\n", sa.ID))
	b.WriteString(serviceAgreementText(sa))
	return b.String()
}
