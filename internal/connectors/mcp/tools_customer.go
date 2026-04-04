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

func registerCustomerTools(server *mcpsrv.MCPServer, service CustomerServiceProvider, guard IngressGuard, logger *slog.Logger) []string {
	registered := make([]string, 0, 4)

	tool, handler := customerListTool(service, guard, logger)
	server.AddTool(tool, handler)
	registered = append(registered, tool.Name)

	tool, handler = customerCreateTool(service, guard, logger)
	server.AddTool(tool, handler)
	registered = append(registered, tool.Name)

	tool, handler = customerUpdateTool(service, guard, logger)
	server.AddTool(tool, handler)
	registered = append(registered, tool.Name)

	tool, handler = customerDeleteTool(service, guard, logger)
	server.AddTool(tool, handler)
	registered = append(registered, tool.Name)

	return registered
}

func customerListTool(service CustomerListProvider, guard IngressGuard, logger *slog.Logger) (mcp.Tool, func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
	tool := mcp.NewTool("customer.list", mcp.WithDescription("Return a paginated list of customers"))
	return tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if service == nil {
			return mcp.NewToolResultError("customer service is required"), nil
		}
		if err := guard.authorize(req.Header); err != nil {
			logging.Event(ctx, logger, slog.LevelWarn, "customer.list", "mcp", "denied", slog.String("reason", classifyMCPAuthReason(err)))
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

		return mcp.NewToolResultText(customerListText(result)), nil
	}
}

func parseSortValue(value string) (string, string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", ""
	}

	if strings.HasPrefix(value, "-") {
		return strings.TrimSpace(strings.TrimPrefix(value, "-")), "desc"
	}

	field, dir, found := strings.Cut(value, ":")
	if !found {
		return strings.TrimSpace(value), ""
	}

	return strings.TrimSpace(field), strings.TrimSpace(dir)
}

func customerCreateTool(service CustomerServiceProvider, guard IngressGuard, logger *slog.Logger) (mcp.Tool, func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
	tool := mcp.NewTool("customer.create",
		mcp.WithDescription(`Create a new customer.

FIELD NAMING (IMPORTANT):
- Use 'type', NOT 'customer_type' — the field is named 'type' (string: "company" or "individual")
- Use 'legal_name', NOT 'name' or 'company' — the field is named 'legal_name' (official/legal name)
- Use 'billing_address.country', NOT top-level 'country' — address fields are nested inside 'billing_address'

CUSTOMER TYPES:
- company: A business customer with legal name and optional trade name
- individual: A personal customer with legal name as full name

REQUIRED FIELDS:
- type: Must be exactly "company" or "individual" (novariants)
- legal_name: The official name (company legal name or individual full name)`),
		mcp.WithString("type",
			mcp.Required(),
			mcp.Description("Customer type: MUST be 'company' or 'individual'. Do NOT use 'customer_type' — the field name is 'type'."),
			mcp.Enum("company", "individual"),
		),
		mcp.WithString("legal_name",
			mcp.Required(),
			mcp.Description("Official/legal name of the customer. Do NOT use 'name' or 'company' — the field name is 'legal_name'. For companies: registered legal name. For individuals: full legal name."),
		),
		mcp.WithString("trade_name",
			mcp.Description("Optional commercial or trading name, different from legal name (e.g., 'Acme' for 'Acme Corporation S.A.')"),
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
			mcp.Description("Customer website URL"),
		),
		mcp.WithString("default_currency",
			mcp.Description("Default currency for billing (ISO 4217 code, e.g., 'USD', 'DOP', 'EUR')"),
		),
		mcp.WithString("notes",
			mcp.Description("Internal notes about this customer"),
		),
		mcp.WithObject("billing_address",
			mcp.Description("Billing address details. All address data (includingcountry) must be nested inside this object — do NOT use top-level 'country' or 'address' fields."),
			mcp.Properties(map[string]any{
				"street":      map[string]any{"type": "string", "description": "Street address line (e.g., '123 Main St')"},
				"city":        map[string]any{"type": "string", "description": "City or municipality (e.g., 'Santo Domingo')"},
				"state":       map[string]any{"type": "string", "description": "State, province, or region (e.g., 'Distrito Nacional')"},
				"postal_code": map[string]any{"type": "string", "description": "Postal or ZIP code (e.g., '10101')"},
				"country":     map[string]any{"type": "string", "description": "Country code or name (e.g., 'DO' or 'Dominican Republic'). Must be inside billing_address, not a top-level field."},
			}),
		),
	)
	return tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if service == nil {
			return mcp.NewToolResultError("customer service is required"), nil
		}
		if err := guard.authorize(req.Header); err != nil {
			logging.Event(ctx, logger, slog.LevelWarn, "customer.create", "mcp", "denied", slog.String("reason", classifyMCPAuthReason(err)))
			return mcp.NewToolResultError(err.Error()), nil
		}

		// Build command from explicit arguments
		cmd := app.CreateCustomerCommand{
			Type:            strings.TrimSpace(req.GetString("type", "")),
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

		return mcp.NewToolResultText(customerCreateText(result)), nil
	}
}

// extractAddressDTO builds an AddressDTO from a map[string]any argument
func extractAddressDTO(m map[string]any) app.AddressDTO {
	return app.AddressDTO{
		Street:     extractString(m, "street"),
		City:       extractString(m, "city"),
		State:      extractString(m, "state"),
		PostalCode: extractString(m, "postal_code"),
		Country:    extractString(m, "country"),
	}
}

// extractString safely extracts a string value from a map
func extractString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func customerUpdateTool(service CustomerServiceProvider, guard IngressGuard, logger *slog.Logger) (mcp.Tool, func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
	tool := mcp.NewTool("customer.update",
		mcp.WithDescription(`Update an existing customer with partial patch.

FIELD NAMING (IMPORTANT):
- Use 'type', NOT 'customer_type' — the field is named 'type' (string: "company" or "individual")
- Use 'legal_name', NOT 'name' or 'company' — the field is named 'legal_name' (official/legal name)
- Use 'billing_address.country', NOT top-level 'country' — address fields are nested inside 'billing_address'

Only provided fields will be updated; omitted fields remain unchanged.
Use empty string "" to clear an optional field.
To patch individual address fields, provide the entire billing_address object or omit it entirely.`),
		mcp.WithString("id",
			mcp.Required(),
			mcp.Description("Customer ID to update (e.g., 'cus_123')"),
		),
		mcp.WithString("type",
			mcp.Description("Update customer type: MUST be 'company' or 'individual'. Do NOT use 'customer_type' — the field name is 'type'."),
			mcp.Enum("company", "individual"),
		),
		mcp.WithString("legal_name",
			mcp.Description("Update official/legal name. Do NOT use 'name' or 'company' — the field name is 'legal_name'."),
		),
		mcp.WithString("trade_name",
			mcp.Description("Update commercial or trading name. Use empty string '' to clear."),
		),
		mcp.WithString("tax_id",
			mcp.Description("Update tax identification number. Use empty string '' to clear."),
		),
		mcp.WithString("email",
			mcp.Description("Update primary contact email. Use empty string '' to clear."),
		),
		mcp.WithString("phone",
			mcp.Description("Update primary contact phone. Use empty string '' to clear."),
		),
		mcp.WithString("website",
			mcp.Description("Update website URL. Use empty string '' to clear."),
		),
		mcp.WithString("default_currency",
			mcp.Description("Update default billing currency (ISO 4217 code). Use empty string '' to clear."),
		),
		mcp.WithString("notes",
			mcp.Description("Update internal notes. Use empty string '' to clear."),
		),
		mcp.WithObject("billing_address",
			mcp.Description("Update billing address. All address data (including country) must be nested inside this object — do NOT use top-level 'country' or 'address' fields. Providing this object replaces the entire address. Omit to keep current address."),
			mcp.Properties(map[string]any{
				"street":      map[string]any{"type": "string", "description": "Street address line (e.g., '123 Main St')"},
				"city":        map[string]any{"type": "string", "description": "City or municipality (e.g., 'Santo Domingo')"},
				"state":       map[string]any{"type": "string", "description": "State, province, or region (e.g., 'Distrito Nacional')"},
				"postal_code": map[string]any{"type": "string", "description": "Postal or ZIP code (e.g., '10101')"},
				"country":     map[string]any{"type": "string", "description": "Country code or name (e.g., 'DO' or 'Dominican Republic'). Must be inside billing_address, not a top-level field."},
			}),
		),
	)
	return tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if service == nil {
			return mcp.NewToolResultError("customer service is required"), nil
		}
		if err := guard.authorize(req.Header); err != nil {
			logging.Event(ctx, logger, slog.LevelWarn, "customer.update", "mcp", "denied", slog.String("reason", classifyMCPAuthReason(err)))
			return mcp.NewToolResultError(err.Error()), nil
		}

		id := strings.TrimSpace(req.GetString("id", ""))
		if id == "" {
			return mcp.NewToolResultError("id argument is required"), nil
		}

		// Build patch command from explicit arguments
		// Only set pointer fields if the argument was provided
		var cmd app.PatchCustomerCommand

		args := req.GetArguments()
		if _, provided := args["type"]; provided {
			cmd.Type = ptrTo(strings.TrimSpace(req.GetString("type", "")))
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
		if _, provided := args["default_currency"]; provided {
			cmd.DefaultCurrency = ptrTo(strings.TrimSpace(req.GetString("default_currency", "")))
		}
		if _, provided := args["notes"]; provided {
			cmd.Notes = ptrTo(strings.TrimSpace(req.GetString("notes", "")))
		}

		// Extract billing address if provided
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

		return mcp.NewToolResultText(customerUpdateText(result)), nil
	}
}

// ptrTo returns a pointer to the given string
func ptrTo(s string) *string {
	return &s
}

func customerDeleteTool(service CustomerServiceProvider, guard IngressGuard, logger *slog.Logger) (mcp.Tool, func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
	tool := mcp.NewTool("customer.delete", mcp.WithDescription("Delete a customer"))
	return tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if service == nil {
			return mcp.NewToolResultError("customer service is required"), nil
		}
		if err := guard.authorize(req.Header); err != nil {
			logging.Event(ctx, logger, slog.LevelWarn, "customer.delete", "mcp", "denied", slog.String("reason", classifyMCPAuthReason(err)))
			return mcp.NewToolResultError(err.Error()), nil
		}

		id := strings.TrimSpace(req.GetString("id", ""))
		if id == "" {
			return mcp.NewToolResultError("id argument is required"), nil
		}

		if err := service.Delete(ctx, id); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Customer deleted: %s", id)), nil
	}
}

func customerCreateText(result app.CustomerDTO) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Customer created: %s\n", result.ID))
	if result.Type != "" {
		b.WriteString(fmt.Sprintf("Type: %s\n", result.Type))
	}
	if result.LegalName != "" {
		b.WriteString(fmt.Sprintf("Legal name: %s\n", result.LegalName))
	}
	if result.Email != "" {
		b.WriteString(fmt.Sprintf("Email: %s\n", result.Email))
	}
	if result.Status != "" {
		b.WriteString(fmt.Sprintf("Status: %s\n", result.Status))
	}
	return b.String()
}

func customerUpdateText(result app.CustomerDTO) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Customer updated: %s\n", result.ID))
	if result.Type != "" {
		b.WriteString(fmt.Sprintf("Type: %s\n", result.Type))
	}
	if result.LegalName != "" {
		b.WriteString(fmt.Sprintf("Legal name: %s\n", result.LegalName))
	}
	if result.Email != "" {
		b.WriteString(fmt.Sprintf("Email: %s\n", result.Email))
	}
	if result.Status != "" {
		b.WriteString(fmt.Sprintf("Status: %s\n", result.Status))
	}
	return b.String()
}
