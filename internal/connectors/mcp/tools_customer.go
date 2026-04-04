package mcp

import (
	"context"
	"encoding/json"
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
	tool := mcp.NewTool("customer.create", mcp.WithDescription("Create a new customer"))
	return tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if service == nil {
			return mcp.NewToolResultError("customer service is required"), nil
		}
		if err := guard.authorize(req.Header); err != nil {
			logging.Event(ctx, logger, slog.LevelWarn, "customer.create", "mcp", "denied", slog.String("reason", classifyMCPAuthReason(err)))
			return mcp.NewToolResultError(err.Error()), nil
		}

		jsonInput := strings.TrimSpace(req.GetString("json", ""))
		if jsonInput == "" {
			return mcp.NewToolResultError("json argument is required"), nil
		}

		var cmd app.CreateCustomerCommand
		if err := json.Unmarshal([]byte(jsonInput), &cmd); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("parse JSON: %v", err)), nil
		}

		result, err := service.Create(ctx, cmd)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(customerCreateText(result)), nil
	}
}

func customerUpdateTool(service CustomerServiceProvider, guard IngressGuard, logger *slog.Logger) (mcp.Tool, func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
	tool := mcp.NewTool("customer.update", mcp.WithDescription("Update an existing customer with partial patch"))
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

		jsonInput := strings.TrimSpace(req.GetString("json", ""))
		if jsonInput == "" {
			return mcp.NewToolResultError("json argument is required"), nil
		}

		var cmd app.PatchCustomerCommand
		if err := json.Unmarshal([]byte(jsonInput), &cmd); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("parse JSON: %v", err)), nil
		}

		result, err := service.Update(ctx, id, cmd)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(customerUpdateText(result)), nil
	}
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
