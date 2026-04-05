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

func registerIssuerProfileTools(server *mcpsrv.MCPServer, service IssuerProfileWriteProvider, guard IngressGuard, logger *slog.Logger) []string {
	registered := make([]string, 0, 4)

	tool, handler := issuerProfileCreateTool(service, guard, logger)
	server.AddTool(tool, handler)
	registered = append(registered, tool.Name)

	tool, handler = issuerProfileGetTool(service, guard, logger)
	server.AddTool(tool, handler)
	registered = append(registered, tool.Name)

	tool, handler = issuerProfileUpdateTool(service, guard, logger)
	server.AddTool(tool, handler)
	registered = append(registered, tool.Name)

	tool, handler = issuerProfileDeleteTool(service, guard, logger)
	server.AddTool(tool, handler)
	registered = append(registered, tool.Name)

	return registered
}

func issuerProfileCreateTool(service IssuerProfileWriteProvider, guard IngressGuard, logger *slog.Logger) (mcp.Tool, func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
	tool := mcp.NewTool("issuer_profile.create",
		mcp.WithDescription(`Create a new issuer profile.

An issuer profile represents the billing operator (your own company). The underlying legal entity is created automatically from the fields provided here.

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
- default_notes: Default notes included on invoices`),
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
		mcp.WithString("default_notes",
			mcp.Description("Default notes to include on invoices"),
		),
	)
	return tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if service == nil {
			return mcp.NewToolResultError("issuer profile service is required"), nil
		}
		if err := guard.authorize(req.Header); err != nil {
			logging.Event(ctx, logger, slog.LevelWarn, "issuer_profile.create", "mcp", "denied", slog.String("reason", classifyMCPAuthReason(err)))
			return mcp.NewToolResultError(err.Error()), nil
		}

		cmd := app.CreateIssuerProfileCommand{
			LegalEntityType: strings.TrimSpace(req.GetString("type", "")),
			LegalName:       strings.TrimSpace(req.GetString("legal_name", "")),
			TradeName:       strings.TrimSpace(req.GetString("trade_name", "")),
			TaxID:           strings.TrimSpace(req.GetString("tax_id", "")),
			Email:           strings.TrimSpace(req.GetString("email", "")),
			Phone:           strings.TrimSpace(req.GetString("phone", "")),
			Website:         strings.TrimSpace(req.GetString("website", "")),
			DefaultCurrency: strings.TrimSpace(req.GetString("default_currency", "")),
			DefaultNotes:    strings.TrimSpace(req.GetString("default_notes", "")),
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

		return mcp.NewToolResultText(issuerProfileCreateText(result)), nil
	}
}

func issuerProfileGetTool(service IssuerProfileWriteProvider, guard IngressGuard, logger *slog.Logger) (mcp.Tool, func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
	tool := mcp.NewTool("issuer_profile.get",
		mcp.WithDescription("Get an issuer profile by ID"),
		mcp.WithString("id",
			mcp.Required(),
			mcp.Description("Issuer profile ID (e.g., 'iss_123')"),
		),
	)
	return tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if service == nil {
			return mcp.NewToolResultError("issuer profile service is required"), nil
		}
		if err := guard.authorize(req.Header); err != nil {
			logging.Event(ctx, logger, slog.LevelWarn, "issuer_profile.get", "mcp", "denied", slog.String("reason", classifyMCPAuthReason(err)))
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

		return mcp.NewToolResultText(issuerProfileGetText(result)), nil
	}
}

func issuerProfileUpdateTool(service IssuerProfileWriteProvider, guard IngressGuard, logger *slog.Logger) (mcp.Tool, func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
	tool := mcp.NewTool("issuer_profile.update",
		mcp.WithDescription(`Update an existing issuer profile with partial patch.

Only provided fields will be updated; omitted fields remain unchanged.
Use empty string "" to clear an optional field.

Legal entity fields (type, legal_name, trade_name, tax_id, email, phone, website, billing_address)
are cascaded to the linked legal entity when provided.`),
		mcp.WithString("id",
			mcp.Required(),
			mcp.Description("Issuer profile ID to update (e.g., 'iss_123')"),
		),
		mcp.WithString("default_currency",
			mcp.Description("Update default billing currency (ISO 4217 code). Use empty string '' to clear."),
		),
		mcp.WithString("default_notes",
			mcp.Description("Update default invoice notes. Use empty string '' to clear."),
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
			return mcp.NewToolResultError("issuer profile service is required"), nil
		}
		if err := guard.authorize(req.Header); err != nil {
			logging.Event(ctx, logger, slog.LevelWarn, "issuer_profile.update", "mcp", "denied", slog.String("reason", classifyMCPAuthReason(err)))
			return mcp.NewToolResultError(err.Error()), nil
		}

		id := strings.TrimSpace(req.GetString("id", ""))
		if id == "" {
			return mcp.NewToolResultError("id argument is required"), nil
		}

		var cmd app.PatchIssuerProfileCommand

		args := req.GetArguments()
		if _, provided := args["default_currency"]; provided {
			cmd.DefaultCurrency = ptrTo(strings.TrimSpace(req.GetString("default_currency", "")))
		}
		if _, provided := args["default_notes"]; provided {
			cmd.DefaultNotes = ptrTo(strings.TrimSpace(req.GetString("default_notes", "")))
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

		return mcp.NewToolResultText(issuerProfileUpdateText(result)), nil
	}
}

func issuerProfileDeleteTool(service IssuerProfileWriteProvider, guard IngressGuard, logger *slog.Logger) (mcp.Tool, func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
	tool := mcp.NewTool("issuer_profile.delete",
		mcp.WithDescription("Delete an issuer profile"),
		mcp.WithString("id",
			mcp.Required(),
			mcp.Description("Issuer profile ID to delete (e.g., 'iss_123')"),
		),
	)
	return tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if service == nil {
			return mcp.NewToolResultError("issuer profile service is required"), nil
		}
		if err := guard.authorize(req.Header); err != nil {
			logging.Event(ctx, logger, slog.LevelWarn, "issuer_profile.delete", "mcp", "denied", slog.String("reason", classifyMCPAuthReason(err)))
			return mcp.NewToolResultError(err.Error()), nil
		}

		id := strings.TrimSpace(req.GetString("id", ""))
		if id == "" {
			return mcp.NewToolResultError("id argument is required"), nil
		}

		if err := service.Delete(ctx, id); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Issuer profile deleted: %s", id)), nil
	}
}

func issuerProfileCreateText(result app.IssuerProfileDTO) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Issuer profile created: %s\n", result.ID))
	b.WriteString(fmt.Sprintf("Legal entity ID: %s\n", result.LegalEntityID))
	if result.DefaultCurrency != "" {
		b.WriteString(fmt.Sprintf("Default currency: %s\n", result.DefaultCurrency))
	}
	return b.String()
}

func issuerProfileGetText(result app.IssuerProfileDTO) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Issuer profile: %s\n", result.ID))
	b.WriteString(fmt.Sprintf("Legal entity ID: %s\n", result.LegalEntityID))
	if result.DefaultCurrency != "" {
		b.WriteString(fmt.Sprintf("Default currency: %s\n", result.DefaultCurrency))
	}
	if result.DefaultNotes != "" {
		b.WriteString(fmt.Sprintf("Default notes: %s\n", result.DefaultNotes))
	}
	return b.String()
}

func issuerProfileUpdateText(result app.IssuerProfileDTO) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Issuer profile updated: %s\n", result.ID))
	b.WriteString(fmt.Sprintf("Legal entity ID: %s\n", result.LegalEntityID))
	if result.DefaultCurrency != "" {
		b.WriteString(fmt.Sprintf("Default currency: %s\n", result.DefaultCurrency))
	}
	return b.String()
}
