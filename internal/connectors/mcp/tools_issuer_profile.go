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
	registered := make([]string, 0, 3)

	tool, handler := issuerProfileCreateTool(service, guard, logger)
	server.AddTool(tool, handler)
	registered = append(registered, tool.Name)

	tool, handler = issuerProfileGetTool(service, guard, logger)
	server.AddTool(tool, handler)
	registered = append(registered, tool.Name)

	tool, handler = issuerProfileUpdateTool(service, guard, logger)
	server.AddTool(tool, handler)
	registered = append(registered, tool.Name)

	return registered
}

func issuerProfileCreateTool(service IssuerProfileWriteProvider, guard IngressGuard, logger *slog.Logger) (mcp.Tool, func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
	tool := mcp.NewTool("issuer_profile.create",
		mcp.WithDescription(`Create a new issuer profile.

An issuer profile represents the billing operator (your own company) linked to a legal entity.

REQUIRED FIELDS:
- legal_entity_id: The ID of the legal entity this profile belongs to

OPTIONAL FIELDS:
- default_currency: Default currency for invoices (ISO 4217 code, e.g., 'USD', 'DOP')
- default_notes: Default notes included on invoices`),
		mcp.WithString("legal_entity_id",
			mcp.Required(),
			mcp.Description("Legal entity ID this profile belongs to (e.g., 'le_123')"),
		),
		mcp.WithString("default_currency",
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
			LegalEntityID:   strings.TrimSpace(req.GetString("legal_entity_id", "")),
			DefaultCurrency: strings.TrimSpace(req.GetString("default_currency", "")),
			DefaultNotes:    strings.TrimSpace(req.GetString("default_notes", "")),
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
Use empty string "" to clear an optional field.`),
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

		result, err := service.Update(ctx, id, cmd)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(issuerProfileUpdateText(result)), nil
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
