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

A customer profile represents a client to be billed. The underlying legal entity is created automatically from the fields provided here.

FIELD NAMING (IMPORTANT):
- Use 'type', NOT 'entity_type' — the field is named 'type' (string: "company" or "individual")
- Use 'legal_name', NOT 'name' — the field is named 'legal_name' (official/legal name)
- Use 'billing_address.country', NOT top-level 'country' — address fields are nested inside 'billing_address'

REQUIRED FIELDS:
- type: Entity type — must be exactly "company" or "individual"
- legal_name: Official/legal name of the entity
- default_currency: Default currency for invoices (ISO 4217 code, e.g., 'USD', 'DOP')

OPTIONAL FIELDS:
- trade_name, tax_id, email, phone, website, billing_address
- notes: Internal notes about this customer`),
		mcp.WithString("type",
			mcp.Required(),
			mcp.Description("Entity type: MUST be 'company' or 'individual'. Do NOT use 'entity_type' — the field name is 'type'."),
			mcp.Enum("company", "individual"),
		),
		mcp.WithString("legal_name",
			mcp.Required(),
			mcp.Description("Official/legal name of the entity. Do NOT use 'name' — the field name is 'legal_name'."),
		),
		mcp.WithString("trade_name",
			mcp.Description("Optional commercial or trading name"),
		),
		mcp.WithString("tax_id",
			mcp.Description("Tax identification number (e.g., RNC, NIT, or similar)"),
		),
		mcp.WithString("email",
			mcp.Description("Primary contact email address"),
		),
		mcp.WithString("phone",
			mcp.Description("Primary contact phone number"),
		),
		mcp.WithString("website",
			mcp.Description("Entity website URL"),
		),
		mcp.WithObject("billing_address",
			mcp.Description("Billing address details. All address data (including country) must be nested inside this object."),
			mcp.Properties(map[string]any{
				"street":      map[string]any{"type": "string", "description": "Street address line"},
				"city":        map[string]any{"type": "string", "description": "City or municipality"},
				"state":       map[string]any{"type": "string", "description": "State, province, or region"},
				"postal_code": map[string]any{"type": "string", "description": "Postal or ZIP code"},
				"country":     map[string]any{"type": "string", "description": "Country code or name (e.g., 'DO'). Must be inside billing_address."},
			}),
		),
		mcp.WithString("default_currency",
			mcp.Required(),
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
			LegalEntityType: strings.TrimSpace(req.GetString("type", "")),
			LegalName:       strings.TrimSpace(req.GetString("legal_name", "")),
			TradeName:       strings.TrimSpace(req.GetString("trade_name", "")),
			TaxID:           strings.TrimSpace(req.GetString("tax_id", "")),
			Email:           strings.TrimSpace(req.GetString("email", "")),
			Phone:           strings.TrimSpace(req.GetString("phone", "")),
			Website:         strings.TrimSpace(req.GetString("website", "")),
			DefaultCurrency: strings.TrimSpace(req.GetString("default_currency", "")),
			Notes:           strings.TrimSpace(req.GetString("notes", "")),
		}

		// Extract billing address if provided
		args := req.GetArguments()
		if addr, ok := args["billing_address"]; ok && addr != nil {
			if addrMap, ok := addr.(map[string]any); ok {
				cmd.BillingAddress = extractAddressDTO(addrMap)
			}
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
Use empty string "" to clear an optional field.

Legal entity fields (type, legal_name, trade_name, tax_id, email, phone, website, billing_address)
are cascaded to the linked legal entity when provided.`),
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
		mcp.WithString("type",
			mcp.Description("Update entity type: 'company' or 'individual'. Cascaded to linked legal entity."),
			mcp.Enum("company", "individual"),
		),
		mcp.WithString("legal_name",
			mcp.Description("Update official/legal name. Cascaded to linked legal entity."),
		),
		mcp.WithString("trade_name",
			mcp.Description("Update commercial or trading name. Cascaded to linked legal entity. Use empty string '' to clear."),
		),
		mcp.WithString("tax_id",
			mcp.Description("Update tax identification number. Cascaded to linked legal entity. Use empty string '' to clear."),
		),
		mcp.WithString("email",
			mcp.Description("Update primary contact email. Cascaded to linked legal entity. Use empty string '' to clear."),
		),
		mcp.WithString("phone",
			mcp.Description("Update primary contact phone. Cascaded to linked legal entity. Use empty string '' to clear."),
		),
		mcp.WithString("website",
			mcp.Description("Update website URL. Cascaded to linked legal entity. Use empty string '' to clear."),
		),
		mcp.WithObject("billing_address",
			mcp.Description("Update billing address. Cascaded to linked legal entity. All address fields must be nested inside this object."),
			mcp.Properties(map[string]any{
				"street":      map[string]any{"type": "string", "description": "Street address line"},
				"city":        map[string]any{"type": "string", "description": "City or municipality"},
				"state":       map[string]any{"type": "string", "description": "State, province, or region"},
				"postal_code": map[string]any{"type": "string", "description": "Postal or ZIP code"},
				"country":     map[string]any{"type": "string", "description": "Country code or name (e.g., 'DO'). Must be inside billing_address."},
			}),
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
		// Legal entity fields — cascaded to the linked entity.
		if _, provided := args["type"]; provided {
			cmd.LegalEntityType = ptrTo(strings.TrimSpace(req.GetString("type", "")))
		}
		if _, provided := args["legal_name"]; provided {
			cmd.LegalName = ptrTo(strings.TrimSpace(req.GetString("legal_name", "")))
		}
		if _, provided := args["trade_name"]; provided {
			cmd.TradeName = ptrTo(strings.TrimSpace(req.GetString("trade_name", "")))
		}
		if _, provided := args["tax_id"]; provided {
			cmd.TaxID = ptrTo(strings.TrimSpace(req.GetString("tax_id", "")))
		}
		if _, provided := args["email"]; provided {
			cmd.Email = ptrTo(strings.TrimSpace(req.GetString("email", "")))
		}
		if _, provided := args["phone"]; provided {
			cmd.Phone = ptrTo(strings.TrimSpace(req.GetString("phone", "")))
		}
		if _, provided := args["website"]; provided {
			cmd.Website = ptrTo(strings.TrimSpace(req.GetString("website", "")))
		}
		if addr, ok := args["billing_address"]; ok && addr != nil {
			if addrMap, ok := addr.(map[string]any); ok {
				addrDTO := extractAddressDTO(addrMap)
				cmd.BillingAddress = &addrDTO
			}
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
