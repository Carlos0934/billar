package mcp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/Carlos0934/billar/internal/app"
	"github.com/Carlos0934/billar/internal/infra/logging"
	"github.com/mark3labs/mcp-go/mcp"
	mcpsrv "github.com/mark3labs/mcp-go/server"
)

func registerTools(server *mcpsrv.MCPServer, sessionService app.SessionService, issuer IssuerProfileWriteProvider, customer CustomerProfileWriteProvider, agreement AgreementServiceProvider, timeEntry TimeEntryServiceProvider, invoice InvoiceServiceProvider, guard IngressGuard, logger *slog.Logger) []string {
	registered := make([]string, 0, 13)

	statusTool, statusHandler := sessionStatusTool(sessionService, guard, logger)
	server.AddTool(statusTool, statusHandler)
	registered = append(registered, statusTool.Name)

	registered = append(registered, registerIssuerProfileTools(server, issuer, guard, logger)...)
	registered = append(registered, registerCustomerProfileTools(server, customer, guard, logger)...)
	registered = append(registered, registerServiceAgreementTools(server, agreement, guard, logger)...)
	registered = append(registered, registerTimeEntryTools(server, timeEntry, guard, logger)...)
	registered = append(registered, registerInvoiceTools(server, invoice, guard, logger)...)

	return registered
}

func statusTool(service app.SessionService, guard IngressGuard, logger *slog.Logger) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	_, handler := sessionStatusTool(service, guard, logger)
	return handler
}

func sessionStatusTool(service app.SessionService, guard IngressGuard, logger *slog.Logger) (mcp.Tool, func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
	tool := mcp.NewTool("session.status", mcp.WithDescription("Return the current session status and identity details"))
	return tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		_ = req
		if service == nil {
			logging.Event(ctx, logger, slog.LevelError, "session.status", "mcp", "error", slog.String("reason", "missing_session_service"))
			return mcp.NewToolResultError("session service is required"), nil
		}
		if err := guard.authorize(req.Header); err != nil {
			logging.Event(ctx, logger, slog.LevelWarn, "session.status", "mcp", "denied", slog.String("reason", classifyMCPAuthReason(err)))
			return mcp.NewToolResultError(err.Error()), nil
		}

		result, err := service.Status(ctx)
		if err != nil {
			logging.Event(ctx, logger, slog.LevelError, "session.status", "mcp", "error", slog.String("reason", classifyMCPAuthReason(err)))
			return mcp.NewToolResultError(err.Error()), nil
		}
		logging.Event(ctx, logger, slog.LevelInfo, "session.status", "mcp", "success", slog.String("status", result.Status))

		return mcp.NewToolResultText(sessionStatusText(result)), nil
	}
}

func classifyMCPAuthReason(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, ErrIPNotAllowed) {
		return "ip_not_allowed"
	}
	return "internal_error"
}

func formatToolRegistrationError(toolName string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("register %s tool: %w", toolName, err)
}
