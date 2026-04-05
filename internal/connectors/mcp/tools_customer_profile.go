package mcp

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/Carlos0934/billar/internal/app"
	"github.com/Carlos0934/billar/internal/infra/logging"
	"github.com/mark3labs/mcp-go/mcp"
	mcpsrv "github.com/mark3labs/mcp-go/server"
)

func registerCustomerProfileTools(server *mcpsrv.MCPServer, service CustomerProfileWriteProvider, guard IngressGuard, logger *slog.Logger) []string {
	registered := make([]string, 0, 5)

	tool, handler := customerProfileListTool(service, guard, logger)
	server.AddTool(tool, handler)
	registered = append(registered, tool.Name)

	tool, handler = customerProfileCreateTool(service, guard, logger)
	server.AddTool(tool, handler)
	registered = append(registered, tool.Name)

	tool, handler = customerProfileGetTool(service, guard, logger)
	server.AddTool(tool, handler)
	registered = append(registered, tool.Name)

	tool, handler = customerProfileUpdateTool(service, guard, logger)
	server.AddTool(tool, handler)
	registered = append(registered, tool.Name)

	tool, handler = customerProfileDeleteTool(service, guard, logger)
	server.AddTool(tool, handler)
	registered = append(registered, tool.Name)

	return registered
}

func customerProfileListTool(service CustomerProfileListProvider, guard IngressGuard, logger *slog.Logger) (mcp.Tool, func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
	tool := mcp.NewTool("customer_profile.list", mcp.WithDescription("Return a paginated list of customer profiles"))
	return tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if service == nil {
			return mcp.NewToolResultError("customer profile service is required"), nil
		}
		if err := guard.authorize(req.Header); err != nil {
			logging.Event(ctx, logger, slog.LevelWarn, "customer_profile.list", "mcp", "denied", slog.String("reason", classifyMCPAuthReason(err)))
			return mcp.NewToolResultError(err.Error()), nil
		}

		query := app.ListQuery{
			Search:    strings.TrimSpace(req.GetString("search", "")),
			SortField: strings.TrimSpace(req.GetString("sort", "")),
			Page:      req.GetInt("page", 0),
			PageSize:  req.GetInt("page_size", 0),
		}
		if query.SortField == "" {
			query.SortField = strings.TrimSpace(req.GetString("sortField", ""))
		}
		query.SortField, query.SortDir = parseSortValue(query.SortField)
		query = query.Normalize()

		result, err := service.List(ctx, query)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(customerProfileListText(result)), nil
	}
}

func customerProfileCreateTool(service CustomerProfileWriteProvider, guard IngressGuard, logger *slog.Logger) (mcp.Tool, func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
	tool := mcp.NewTool("customer_profile.create",
		mcp.WithDescription(`Create a new customer profile.

A customer profile represents a client to be billed, linked to a legal entity.

REQUIRED FIELDS:
- legal_entity_id: The ID of the legal entity this profile belongs to

OPTIONAL FIELDS:
- default_currency: Default currency for invoices (ISO 4217 code, e.g., 'USD', 'DOP')
- notes: Internal notes about this customer`),
		mcp.WithString("legal_entity_id",
			mcp.Required(),
			mcp.Description("Legal entity ID this profile belongs to (e.g., 'le_123')"),
		),
		mcp.WithString("default_currency",
			mcp.Description("Default currency for billing (ISO 4217 code, e.g., 'USD', 'DOP', 'EUR')"),
		),
		mcp.WithString("notes",
			mcp.Description("Internal notes about this customer"),
		),
	)
	return tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if service == nil {
			return mcp.NewToolResultError("customer profile service is required"), nil
		}
		if err := guard.authorize(req.Header); err != nil {
			logging.Event(ctx, logger, slog.LevelWarn, "customer_profile.create", "mcp", "denied", slog.String("reason", classifyMCPAuthReason(err)))
			return mcp.NewToolResultError(err.Error()), nil
		}

		cmd := app.CreateCustomerProfileCommand{
			LegalEntityID:   strings.TrimSpace(req.GetString("legal_entity_id", "")),
			DefaultCurrency: strings.TrimSpace(req.GetString("default_currency", "")),
			Notes:           strings.TrimSpace(req.GetString("notes", "")),
		}

		result, err := service.Create(ctx, cmd)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(customerProfileCreateText(result)), nil
	}
}

func customerProfileGetTool(service CustomerProfileWriteProvider, guard IngressGuard, logger *slog.Logger) (mcp.Tool, func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
	tool := mcp.NewTool("customer_profile.get",
		mcp.WithDescription("Get a customer profile by ID"),
		mcp.WithString("id",
			mcp.Required(),
			mcp.Description("Customer profile ID (e.g., 'cus_123')"),
		),
	)
	return tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if service == nil {
			return mcp.NewToolResultError("customer profile service is required"), nil
		}
		if err := guard.authorize(req.Header); err != nil {
			logging.Event(ctx, logger, slog.LevelWarn, "customer_profile.get", "mcp", "denied", slog.String("reason", classifyMCPAuthReason(err)))
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

		return mcp.NewToolResultText(customerProfileGetText(result)), nil
	}
}

func customerProfileUpdateTool(service CustomerProfileWriteProvider, guard IngressGuard, logger *slog.Logger) (mcp.Tool, func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
	tool := mcp.NewTool("customer_profile.update",
		mcp.WithDescription(`Update an existing customer profile with partial patch.

Only provided fields will be updated; omitted fields remain unchanged.
Use empty string "" to clear an optional field.`),
		mcp.WithString("id",
			mcp.Required(),
			mcp.Description("Customer profile ID to update (e.g., 'cus_123')"),
		),
		mcp.WithString("status",
			mcp.Description("Update customer status (e.g., 'active', 'inactive')"),
		),
		mcp.WithString("default_currency",
			mcp.Description("Update default billing currency (ISO 4217 code). Use empty string '' to clear."),
		),
		mcp.WithString("notes",
			mcp.Description("Update internal notes. Use empty string '' to clear."),
		),
	)
	return tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if service == nil {
			return mcp.NewToolResultError("customer profile service is required"), nil
		}
		if err := guard.authorize(req.Header); err != nil {
			logging.Event(ctx, logger, slog.LevelWarn, "customer_profile.update", "mcp", "denied", slog.String("reason", classifyMCPAuthReason(err)))
			return mcp.NewToolResultError(err.Error()), nil
		}

		id := strings.TrimSpace(req.GetString("id", ""))
		if id == "" {
			return mcp.NewToolResultError("id argument is required"), nil
		}

		var cmd app.PatchCustomerProfileCommand

		args := req.GetArguments()
		if _, provided := args["status"]; provided {
			cmd.Status = ptrTo(strings.TrimSpace(req.GetString("status", "")))
		}
		if _, provided := args["default_currency"]; provided {
			cmd.DefaultCurrency = ptrTo(strings.TrimSpace(req.GetString("default_currency", "")))
		}
		if _, provided := args["notes"]; provided {
			cmd.Notes = ptrTo(strings.TrimSpace(req.GetString("notes", "")))
		}

		result, err := service.Update(ctx, id, cmd)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(customerProfileUpdateText(result)), nil
	}
}

func customerProfileDeleteTool(service CustomerProfileWriteProvider, guard IngressGuard, logger *slog.Logger) (mcp.Tool, func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
	tool := mcp.NewTool("customer_profile.delete", mcp.WithDescription("Delete a customer profile"))
	return tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if service == nil {
			return mcp.NewToolResultError("customer profile service is required"), nil
		}
		if err := guard.authorize(req.Header); err != nil {
			logging.Event(ctx, logger, slog.LevelWarn, "customer_profile.delete", "mcp", "denied", slog.String("reason", classifyMCPAuthReason(err)))
			return mcp.NewToolResultError(err.Error()), nil
		}

		id := strings.TrimSpace(req.GetString("id", ""))
		if id == "" {
			return mcp.NewToolResultError("id argument is required"), nil
		}

		if err := service.Delete(ctx, id); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Customer profile deleted: %s", id)), nil
	}
}

func customerProfileListText(result app.ListResult[app.CustomerProfileDTO]) string {
	var builder strings.Builder
	builder.WriteString("Billar Customer Profiles\n")
	builder.WriteString("───────────────\n")
	builder.WriteString(fmt.Sprintf("Page: %d\n", result.Page))
	builder.WriteString(fmt.Sprintf("Page size: %d\n", result.PageSize))
	builder.WriteString(fmt.Sprintf("Total: %d\n", result.Total))

	if len(result.Items) == 0 {
		builder.WriteString("No customer profiles found\n")
		return builder.String()
	}

	builder.WriteString("\n")
	for i, profile := range result.Items {
		if i > 0 {
			builder.WriteString("\n")
		}
		builder.WriteString(fmt.Sprintf("%d. Customer Profile %s\n", i+1, profile.ID))
		builder.WriteString(fmt.Sprintf("   Legal entity ID: %s\n", profile.LegalEntityID))
		builder.WriteString(fmt.Sprintf("   Status: %s\n", profile.Status))
		if profile.DefaultCurrency != "" {
			builder.WriteString(fmt.Sprintf("   Default currency: %s\n", profile.DefaultCurrency))
		}
		if profile.CreatedAt != "" {
			builder.WriteString(fmt.Sprintf("   Created at: %s\n", profile.CreatedAt))
		}
		if profile.UpdatedAt != "" {
			builder.WriteString(fmt.Sprintf("   Updated at: %s\n", profile.UpdatedAt))
		}
	}

	return builder.String()
}

func customerProfileCreateText(result app.CustomerProfileDTO) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Customer profile created: %s\n", result.ID))
	b.WriteString(fmt.Sprintf("Legal entity ID: %s\n", result.LegalEntityID))
	b.WriteString(fmt.Sprintf("Status: %s\n", result.Status))
	if result.DefaultCurrency != "" {
		b.WriteString(fmt.Sprintf("Default currency: %s\n", result.DefaultCurrency))
	}
	return b.String()
}

func customerProfileGetText(result app.CustomerProfileDTO) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Customer profile: %s\n", result.ID))
	b.WriteString(fmt.Sprintf("Legal entity ID: %s\n", result.LegalEntityID))
	b.WriteString(fmt.Sprintf("Status: %s\n", result.Status))
	if result.DefaultCurrency != "" {
		b.WriteString(fmt.Sprintf("Default currency: %s\n", result.DefaultCurrency))
	}
	if result.Notes != "" {
		b.WriteString(fmt.Sprintf("Notes: %s\n", result.Notes))
	}
	return b.String()
}

func customerProfileUpdateText(result app.CustomerProfileDTO) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Customer profile updated: %s\n", result.ID))
	b.WriteString(fmt.Sprintf("Legal entity ID: %s\n", result.LegalEntityID))
	b.WriteString(fmt.Sprintf("Status: %s\n", result.Status))
	if result.DefaultCurrency != "" {
		b.WriteString(fmt.Sprintf("Default currency: %s\n", result.DefaultCurrency))
	}
	return b.String()
}
