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

// InvoiceServiceProvider is the seam that MCP tools use to call InvoiceService operations.
type InvoiceServiceProvider interface {
	CreateDraftFromUnbilled(ctx context.Context, cmd app.CreateDraftFromUnbilledCommand) (app.InvoiceDTO, error)
	IssueDraft(ctx context.Context, cmd app.IssueInvoiceCommand) (app.InvoiceDTO, error)
	Discard(ctx context.Context, id string) (app.DiscardResult, error)
}

func registerInvoiceTools(server *mcpsrv.MCPServer, service InvoiceServiceProvider, guard IngressGuard, logger *slog.Logger) []string {
	registered := make([]string, 0, 3)

	tool, handler := invoiceDraftTool(service, guard, logger)
	server.AddTool(tool, handler)
	registered = append(registered, tool.Name)

	tool, handler = invoiceIssueTool(service, guard, logger)
	server.AddTool(tool, handler)
	registered = append(registered, tool.Name)

	tool, handler = invoiceDiscardTool(service, guard, logger)
	server.AddTool(tool, handler)
	registered = append(registered, tool.Name)

	return registered
}

func invoiceDraftTool(service InvoiceServiceProvider, guard IngressGuard, logger *slog.Logger) (mcp.Tool, func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
	tool := mcp.NewTool("invoice.draft",
		mcp.WithDescription("Create a draft invoice from unbilled time entries for a customer"),
		mcp.WithString("customer_profile_id",
			mcp.Required(),
			mcp.Description("Customer profile ID (e.g., 'cus_123')"),
		),
	)

	return tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if service == nil {
			return mcp.NewToolResultError("invoice service is required"), nil
		}
		if err := guard.authorize(req.Header); err != nil {
			logging.Event(ctx, logger, slog.LevelWarn, "invoice.draft", "mcp", "denied", slog.String("reason", classifyMCPAuthReason(err)))
			return mcp.NewToolResultError(err.Error()), nil
		}

		customerProfileID := strings.TrimSpace(req.GetString("customer_profile_id", ""))
		if customerProfileID == "" {
			return mcp.NewToolResultError("customer_profile_id argument is required"), nil
		}

		result, err := service.CreateDraftFromUnbilled(ctx, app.CreateDraftFromUnbilledCommand{CustomerProfileID: customerProfileID})
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(invoiceDraftText(result)), nil
	}
}

func invoiceIssueTool(service InvoiceServiceProvider, guard IngressGuard, logger *slog.Logger) (mcp.Tool, func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
	tool := mcp.NewTool("invoice.issue",
		mcp.WithDescription("Issue a draft invoice, assigning a permanent invoice number"),
		mcp.WithString("id",
			mcp.Required(),
			mcp.Description("Invoice ID to issue (e.g., 'inv_123')"),
		),
	)

	return tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if service == nil {
			return mcp.NewToolResultError("invoice service is required"), nil
		}
		if err := guard.authorize(req.Header); err != nil {
			logging.Event(ctx, logger, slog.LevelWarn, "invoice.issue", "mcp", "denied", slog.String("reason", classifyMCPAuthReason(err)))
			return mcp.NewToolResultError(err.Error()), nil
		}

		id := strings.TrimSpace(req.GetString("id", ""))
		if id == "" {
			return mcp.NewToolResultError("id argument is required"), nil
		}

		result, err := service.IssueDraft(ctx, app.IssueInvoiceCommand{InvoiceID: id})
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(invoiceIssueText(result)), nil
	}
}

func invoiceDiscardTool(service InvoiceServiceProvider, guard IngressGuard, logger *slog.Logger) (mcp.Tool, func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
	tool := mcp.NewTool("invoice.discard",
		mcp.WithDescription("Discard an invoice. Drafts are hard-deleted (entries unlocked); issued invoices are soft-discarded (number permanently consumed)."),
		mcp.WithString("id",
			mcp.Required(),
			mcp.Description("Invoice ID to discard (e.g., 'inv_123')"),
		),
	)

	return tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if service == nil {
			return mcp.NewToolResultError("invoice service is required"), nil
		}
		if err := guard.authorize(req.Header); err != nil {
			logging.Event(ctx, logger, slog.LevelWarn, "invoice.discard", "mcp", "denied", slog.String("reason", classifyMCPAuthReason(err)))
			return mcp.NewToolResultError(err.Error()), nil
		}

		id := strings.TrimSpace(req.GetString("id", ""))
		if id == "" {
			return mcp.NewToolResultError("id argument is required"), nil
		}

		result, err := service.Discard(ctx, id)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(invoiceDiscardText(id, result)), nil
	}
}

// -- text helpers --

func invoiceDraftText(inv app.InvoiceDTO) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Invoice draft created: %s\n", inv.ID))
	b.WriteString(invoiceTextFields(inv))
	return b.String()
}

func invoiceIssueText(inv app.InvoiceDTO) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Invoice issued: %s\n", inv.ID))
	b.WriteString(invoiceTextFields(inv))
	return b.String()
}

func invoiceDiscardText(id string, result app.DiscardResult) string {
	var b strings.Builder
	if result.WasSoftDiscard {
		b.WriteString(fmt.Sprintf("Invoice soft-discarded: %s\n", id))
		b.WriteString(fmt.Sprintf("Warning: Invoice %s was soft-discarded. Its number (%s) is permanently consumed.\n", id, result.InvoiceNumber))
	} else {
		b.WriteString(fmt.Sprintf("Invoice discarded: %s\n", id))
		b.WriteString("Draft invoice deleted; linked time entries unlocked.\n")
	}
	return b.String()
}

func invoiceTextFields(inv app.InvoiceDTO) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("ID: %s\n", inv.ID))
	if inv.InvoiceNumber != "" {
		b.WriteString(fmt.Sprintf("Number: %s\n", inv.InvoiceNumber))
	}
	b.WriteString(fmt.Sprintf("Customer: %s\n", inv.CustomerID))
	b.WriteString(fmt.Sprintf("Status: %s\n", inv.Status))
	b.WriteString(fmt.Sprintf("Currency: %s\n", inv.Currency))
	if len(inv.Lines) > 0 {
		b.WriteString(fmt.Sprintf("Lines: %d\n", len(inv.Lines)))
	}
	return b.String()
}
