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

func registerLegalEntityTools(server *mcpsrv.MCPServer, service LegalEntityWriteProvider, guard IngressGuard, logger *slog.Logger) []string {
	registered := make([]string, 0, 5)

	tool, handler := legalEntityListTool(service, guard, logger)
	server.AddTool(tool, handler)
	registered = append(registered, tool.Name)

	tool, handler = legalEntityCreateTool(service, guard, logger)
	server.AddTool(tool, handler)
	registered = append(registered, tool.Name)

	tool, handler = legalEntityGetTool(service, guard, logger)
	server.AddTool(tool, handler)
	registered = append(registered, tool.Name)

	tool, handler = legalEntityUpdateTool(service, guard, logger)
	server.AddTool(tool, handler)
	registered = append(registered, tool.Name)

	tool, handler = legalEntityDeleteTool(service, guard, logger)
	server.AddTool(tool, handler)
	registered = append(registered, tool.Name)

	return registered
}

func legalEntityListTool(service LegalEntityListProvider, guard IngressGuard, logger *slog.Logger) (mcp.Tool, func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
	tool := mcp.NewTool("legal_entity.list", mcp.WithDescription("Return a paginated list of legal entities"))
	return tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if service == nil {
			return mcp.NewToolResultError("legal entity service is required"), nil
		}
		if err := guard.authorize(req.Header); err != nil {
			logging.Event(ctx, logger, slog.LevelWarn, "legal_entity.list", "mcp", "denied", slog.String("reason", classifyMCPAuthReason(err)))
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

		return mcp.NewToolResultText(legalEntityListText(result)), nil
	}
}

func legalEntityCreateTool(service LegalEntityWriteProvider, guard IngressGuard, logger *slog.Logger) (mcp.Tool, func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
	tool := mcp.NewTool("legal_entity.create",
		mcp.WithDescription(`Create a new legal entity.

FIELD NAMING (IMPORTANT):
- Use 'type', NOT 'entity_type' — the field is named 'type' (string: "company" or "individual")
- Use 'legal_name', NOT 'name' — the field is named 'legal_name' (official/legal name)
- Use 'billing_address.country', NOT top-level 'country' — address fields are nested inside 'billing_address'

ENTITY TYPES:
- company: A business entity with legal name and optional trade name
- individual: A person with legal name as full name

REQUIRED FIELDS:
- type: Must be exactly "company" or "individual"
- legal_name: The official name (company legal name or individual full name)`),
		mcp.WithString("type",
			mcp.Required(),
			mcp.Description("Entity type: MUST be 'company' or 'individual'. Do NOT use 'entity_type' — the field name is 'type'."),
			mcp.Enum("company", "individual"),
		),
		mcp.WithString("legal_name",
			mcp.Required(),
			mcp.Description("Official/legal name of the entity. Do NOT use 'name' — the field name is 'legal_name'. For companies: registered legal name. For individuals: full legal name."),
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
			mcp.Description("Entity website URL"),
		),
		mcp.WithObject("billing_address",
			mcp.Description("Billing address details. All address data (including country) must be nested inside this object — do NOT use top-level 'country' or 'address' fields."),
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
			return mcp.NewToolResultError("legal entity service is required"), nil
		}
		if err := guard.authorize(req.Header); err != nil {
			logging.Event(ctx, logger, slog.LevelWarn, "legal_entity.create", "mcp", "denied", slog.String("reason", classifyMCPAuthReason(err)))
			return mcp.NewToolResultError(err.Error()), nil
		}

		// Build command from explicit arguments
		cmd := app.CreateLegalEntityCommand{
			Type:      strings.TrimSpace(req.GetString("type", "")),
			LegalName: strings.TrimSpace(req.GetString("legal_name", "")),
			TradeName: strings.TrimSpace(req.GetString("trade_name", "")),
			TaxID:     strings.TrimSpace(req.GetString("tax_id", "")),
			Email:     strings.TrimSpace(req.GetString("email", "")),
			Phone:     strings.TrimSpace(req.GetString("phone", "")),
			Website:   strings.TrimSpace(req.GetString("website", "")),
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

		return mcp.NewToolResultText(legalEntityCreateText(result)), nil
	}
}

func legalEntityGetTool(service LegalEntityWriteProvider, guard IngressGuard, logger *slog.Logger) (mcp.Tool, func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
	tool := mcp.NewTool("legal_entity.get",
		mcp.WithDescription("Get a legal entity by ID"),
		mcp.WithString("id",
			mcp.Required(),
			mcp.Description("Legal entity ID (e.g., 'le_123')"),
		),
	)
	return tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if service == nil {
			return mcp.NewToolResultError("legal entity service is required"), nil
		}
		if err := guard.authorize(req.Header); err != nil {
			logging.Event(ctx, logger, slog.LevelWarn, "legal_entity.get", "mcp", "denied", slog.String("reason", classifyMCPAuthReason(err)))
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

		return mcp.NewToolResultText(legalEntityGetText(result)), nil
	}
}

func legalEntityUpdateTool(service LegalEntityWriteProvider, guard IngressGuard, logger *slog.Logger) (mcp.Tool, func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
	tool := mcp.NewTool("legal_entity.update",
		mcp.WithDescription(`Update an existing legal entity with partial patch.

FIELD NAMING (IMPORTANT):
- Use 'type', NOT 'entity_type' — the field is named 'type' (string: "company" or "individual")
- Use 'legal_name', NOT 'name' — the field is named 'legal_name' (official/legal name)
- Use 'billing_address.country', NOT top-level 'country' — address fields are nested inside 'billing_address'

Only provided fields will be updated; omitted fields remain unchanged.
Use empty string "" to clear an optional field.
To patch individual address fields, provide the entire billing_address object or omit it entirely.`),
		mcp.WithString("id",
			mcp.Required(),
			mcp.Description("Legal entity ID to update (e.g., 'le_123')"),
		),
		mcp.WithString("type",
			mcp.Description("Update entity type: MUST be 'company' or 'individual'. Do NOT use 'entity_type' — the field name is 'type'."),
			mcp.Enum("company", "individual"),
		),
		mcp.WithString("legal_name",
			mcp.Description("Update official/legal name. Do NOT use 'name' — the field name is 'legal_name'."),
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
			return mcp.NewToolResultError("legal entity service is required"), nil
		}
		if err := guard.authorize(req.Header); err != nil {
			logging.Event(ctx, logger, slog.LevelWarn, "legal_entity.update", "mcp", "denied", slog.String("reason", classifyMCPAuthReason(err)))
			return mcp.NewToolResultError(err.Error()), nil
		}

		id := strings.TrimSpace(req.GetString("id", ""))
		if id == "" {
			return mcp.NewToolResultError("id argument is required"), nil
		}

		// Build patch command from explicit arguments
		// Only set pointer fields if the argument was provided
		var cmd app.PatchLegalEntityCommand

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

		return mcp.NewToolResultText(legalEntityUpdateText(result)), nil
	}
}

func legalEntityDeleteTool(service LegalEntityWriteProvider, guard IngressGuard, logger *slog.Logger) (mcp.Tool, func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
	tool := mcp.NewTool("legal_entity.delete", mcp.WithDescription("Delete a legal entity"))
	return tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if service == nil {
			return mcp.NewToolResultError("legal entity service is required"), nil
		}
		if err := guard.authorize(req.Header); err != nil {
			logging.Event(ctx, logger, slog.LevelWarn, "legal_entity.delete", "mcp", "denied", slog.String("reason", classifyMCPAuthReason(err)))
			return mcp.NewToolResultError(err.Error()), nil
		}

		id := strings.TrimSpace(req.GetString("id", ""))
		if id == "" {
			return mcp.NewToolResultError("id argument is required"), nil
		}

		if err := service.Delete(ctx, id); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Legal entity deleted: %s", id)), nil
	}
}

func legalEntityListText(result app.ListResult[app.LegalEntityDTO]) string {
	var builder strings.Builder
	builder.WriteString("Billar Legal Entities\n")
	builder.WriteString("───────────────\n")
	builder.WriteString(fmt.Sprintf("Page: %d\n", result.Page))
	builder.WriteString(fmt.Sprintf("Page size: %d\n", result.PageSize))
	builder.WriteString(fmt.Sprintf("Total: %d\n", result.Total))

	if len(result.Items) == 0 {
		builder.WriteString("No legal entities found\n")
		return builder.String()
	}

	builder.WriteString("\n")
	for i, entity := range result.Items {
		if i > 0 {
			builder.WriteString("\n")
		}
		builder.WriteString(fmt.Sprintf("%d. %s\n", i+1, entity.LegalName))
		if entity.TradeName != "" && entity.TradeName != entity.LegalName {
			builder.WriteString(fmt.Sprintf("   Trade name: %s\n", entity.TradeName))
		}
		builder.WriteString(fmt.Sprintf("   Type: %s\n", entity.Type))
		if entity.TaxID != "" {
			builder.WriteString(fmt.Sprintf("   Tax ID: %s\n", entity.TaxID))
		}
		if entity.Email != "" {
			builder.WriteString(fmt.Sprintf("   Email: %s\n", entity.Email))
		}
		if entity.Phone != "" {
			builder.WriteString(fmt.Sprintf("   Phone: %s\n", entity.Phone))
		}
		if entity.Website != "" {
			builder.WriteString(fmt.Sprintf("   Website: %s\n", entity.Website))
		}
		if entity.CreatedAt != "" {
			builder.WriteString(fmt.Sprintf("   Created at: %s\n", entity.CreatedAt))
		}
		if entity.UpdatedAt != "" {
			builder.WriteString(fmt.Sprintf("   Updated at: %s\n", entity.UpdatedAt))
		}
	}

	return builder.String()
}

func legalEntityCreateText(result app.LegalEntityDTO) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Legal entity created: %s\n", result.ID))
	if result.Type != "" {
		b.WriteString(fmt.Sprintf("Type: %s\n", result.Type))
	}
	if result.LegalName != "" {
		b.WriteString(fmt.Sprintf("Legal name: %s\n", result.LegalName))
	}
	if result.Email != "" {
		b.WriteString(fmt.Sprintf("Email: %s\n", result.Email))
	}
	if result.Phone != "" {
		b.WriteString(fmt.Sprintf("Phone: %s\n", result.Phone))
	}
	return b.String()
}

func legalEntityGetText(result app.LegalEntityDTO) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Legal entity: %s\n", result.ID))
	if result.Type != "" {
		b.WriteString(fmt.Sprintf("Type: %s\n", result.Type))
	}
	if result.LegalName != "" {
		b.WriteString(fmt.Sprintf("Legal name: %s\n", result.LegalName))
	}
	if result.TradeName != "" {
		b.WriteString(fmt.Sprintf("Trade name: %s\n", result.TradeName))
	}
	if result.Email != "" {
		b.WriteString(fmt.Sprintf("Email: %s\n", result.Email))
	}
	if result.Phone != "" {
		b.WriteString(fmt.Sprintf("Phone: %s\n", result.Phone))
	}
	return b.String()
}

func legalEntityUpdateText(result app.LegalEntityDTO) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Legal entity updated: %s\n", result.ID))
	if result.Type != "" {
		b.WriteString(fmt.Sprintf("Type: %s\n", result.Type))
	}
	if result.LegalName != "" {
		b.WriteString(fmt.Sprintf("Legal name: %s\n", result.LegalName))
	}
	if result.Email != "" {
		b.WriteString(fmt.Sprintf("Email: %s\n", result.Email))
	}
	if result.Phone != "" {
		b.WriteString(fmt.Sprintf("Phone: %s\n", result.Phone))
	}
	return b.String()
}

// extractAddressDTO builds an app.AddressDTO from a map[string]any argument
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
