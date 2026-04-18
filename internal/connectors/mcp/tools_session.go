package mcp

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Carlos0934/billar/internal/app"
	"github.com/Carlos0934/billar/internal/infra/logging"
	"github.com/mark3labs/mcp-go/mcp"
	mcpsrv "github.com/mark3labs/mcp-go/server"
)

func registerTools(server *mcpsrv.MCPServer, sessionService app.SessionService, issuer IssuerProfileWriteProvider, customer CustomerProfileWriteProvider, agreement AgreementServiceProvider, timeEntry TimeEntryServiceProvider, invoice InvoiceServiceProvider, logger *slog.Logger) []string {
	registered := make([]string, 0, 13)

	statusTool, statusHandler := sessionStatusTool(sessionService, logger)
	server.AddTool(statusTool, statusHandler)
	registered = append(registered, statusTool.Name)

	registered = append(registered, registerIssuerProfileTools(server, issuer, logger)...)
	registered = append(registered, registerCustomerProfileTools(server, customer, logger)...)
	registered = append(registered, registerServiceAgreementTools(server, agreement, logger)...)
	registered = append(registered, registerTimeEntryTools(server, timeEntry, logger)...)
	registered = append(registered, registerInvoiceTools(server, invoice, logger)...)

	return registered
}

func statusTool(service app.SessionService, logger *slog.Logger) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	_, handler := sessionStatusTool(service, logger)
	return handler
}

func sessionStatusTool(service app.SessionService, logger *slog.Logger) (mcp.Tool, func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
	tool := mcp.NewTool("session.status", mcp.WithDescription("Return the current session status and identity details"))
	return tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		_ = req
		if service == nil {
			logging.Event(ctx, logger, slog.LevelError, "session.status", "mcp", "error", slog.String("reason", "missing_session_service"))
			return mcp.NewToolResultError("session service is required"), nil
		}

		result, err := service.Status(ctx)
		if err != nil {
			logging.Event(ctx, logger, slog.LevelError, "session.status", "mcp", "error", slog.String("reason", err.Error()))
			return mcp.NewToolResultError(err.Error()), nil
		}
		logging.Event(ctx, logger, slog.LevelInfo, "session.status", "mcp", "success", slog.String("status", result.Status))

		return mcp.NewToolResultText(sessionStatusText(result)), nil
	}
}

func formatToolRegistrationError(toolName string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("register %s tool: %w", toolName, err)
}
