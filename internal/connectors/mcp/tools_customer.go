package mcp

import (
	"context"
	"strings"

	"github.com/Carlos0934/billar/internal/app"
	"github.com/mark3labs/mcp-go/mcp"
	mcpsrv "github.com/mark3labs/mcp-go/server"
)

func registerCustomerTools(server *mcpsrv.MCPServer, service CustomerListProvider, guard IngressGuard) []string {
	registered := make([]string, 0, 1)

	tool, handler := customerListTool(service, guard)
	server.AddTool(tool, handler)
	registered = append(registered, tool.Name)

	return registered
}

func customerListTool(service CustomerListProvider, guard IngressGuard) (mcp.Tool, func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
	tool := mcp.NewTool("customer.list", mcp.WithDescription("Return a paginated list of customers"))
	return tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if service == nil {
			return mcp.NewToolResultError("customer service is required"), nil
		}
		if err := guard.authorize(req.Header); err != nil {
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
