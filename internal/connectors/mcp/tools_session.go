package mcp

import (
	"context"
	"fmt"

	"github.com/Carlos0934/billar/internal/app"
	"github.com/mark3labs/mcp-go/mcp"
	mcpsrv "github.com/mark3labs/mcp-go/server"
)

func registerSessionTools(server *mcpsrv.MCPServer, service app.SessionService, guard IngressGuard) []string {
	registered := make([]string, 0, 3)

	loginTool, loginHandler := sessionStartLoginTool(service)
	server.AddTool(loginTool, loginHandler)
	registered = append(registered, loginTool.Name)

	statusTool, statusHandler := sessionStatusTool(service, guard)
	server.AddTool(statusTool, statusHandler)
	registered = append(registered, statusTool.Name)

	logoutTool, logoutHandler := sessionLogoutTool(service, guard)
	server.AddTool(logoutTool, logoutHandler)
	registered = append(registered, logoutTool.Name)

	return registered
}

func startLoginTool(service app.SessionService) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	_, handler := sessionStartLoginTool(service)
	return handler
}

func statusTool(service app.SessionService, guard IngressGuard) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	_, handler := sessionStatusTool(service, guard)
	return handler
}

func logoutTool(service app.SessionService, guard IngressGuard) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	_, handler := sessionLogoutTool(service, guard)
	return handler
}

func sessionStartLoginTool(service app.SessionService) (mcp.Tool, func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
	tool := mcp.NewTool("session.start_login", mcp.WithDescription("Start the login flow and return the login URL"))
	return tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		_ = req
		if service == nil {
			return mcp.NewToolResultError("session service is required"), nil
		}

		result, err := service.StartLogin(ctx)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(loginIntentText(result)), nil
	}
}

func sessionStatusTool(service app.SessionService, guard IngressGuard) (mcp.Tool, func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
	tool := mcp.NewTool("session.status", mcp.WithDescription("Return the current session status and identity details"))
	return tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		_ = req
		if service == nil {
			return mcp.NewToolResultError("session service is required"), nil
		}
		if err := guard.authorize(req.Header); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		result, err := service.Status(ctx)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(sessionStatusText(result)), nil
	}
}

func sessionLogoutTool(service app.SessionService, guard IngressGuard) (mcp.Tool, func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
	tool := mcp.NewTool("session.logout", mcp.WithDescription("Terminate the current session"))
	return tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		_ = req
		if service == nil {
			return mcp.NewToolResultError("session service is required"), nil
		}
		if err := guard.authorize(req.Header); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		result, err := service.Logout(ctx)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(logoutText(result)), nil
	}
}

func formatToolRegistrationError(toolName string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("register %s tool: %w", toolName, err)
}
