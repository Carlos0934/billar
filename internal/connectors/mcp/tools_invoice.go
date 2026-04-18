package mcp

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/Carlos0934/billar/internal/app"
	"github.com/mark3labs/mcp-go/mcp"
	mcpsrv "github.com/mark3labs/mcp-go/server"
)

// InvoiceServiceProvider is the seam that MCP tools use to call InvoiceService operations.
type InvoiceServiceProvider interface {
	CreateDraftFromUnbilled(ctx context.Context, cmd app.CreateDraftFromUnbilledCommand) (app.InvoiceDTO, error)
	IssueDraft(ctx context.Context, cmd app.IssueInvoiceCommand) (app.InvoiceDTO, error)
	Discard(ctx context.Context, id string) (app.DiscardResult, error)
	GetInvoice(ctx context.Context, id string) (app.InvoiceDTO, error)
	ListInvoices(ctx context.Context, customerID string, statusFilter string) ([]app.InvoiceSummaryDTO, error)
}

func registerInvoiceTools(server *mcpsrv.MCPServer, service InvoiceServiceProvider, logger *slog.Logger) []string {
	registered := make([]string, 0, 5)

	tool, handler := invoiceDraftTool(service, logger)
	server.AddTool(tool, handler)
	registered = append(registered, tool.Name)

	tool, handler = invoiceIssueTool(service, logger)
	server.AddTool(tool, handler)
	registered = append(registered, tool.Name)

	tool, handler = invoiceDiscardTool(service, logger)
	server.AddTool(tool, handler)
	registered = append(registered, tool.Name)

	tool, handler = invoiceGetTool(service, logger)
	server.AddTool(tool, handler)
	registered = append(registered, tool.Name)

	tool, handler = invoiceListTool(service, logger)
	server.AddTool(tool, handler)
	registered = append(registered, tool.Name)

	return registered
}

func invoiceDraftTool(service InvoiceServiceProvider, logger *slog.Logger) (mcp.Tool, func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
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

func invoiceIssueTool(service InvoiceServiceProvider, logger *slog.Logger) (mcp.Tool, func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
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

func invoiceDiscardTool(service InvoiceServiceProvider, logger *slog.Logger) (mcp.Tool, func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
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

func invoiceGetTool(service InvoiceServiceProvider, logger *slog.Logger) (mcp.Tool, func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
	tool := mcp.NewTool("invoice.get",
		mcp.WithDescription("Retrieve a single invoice by ID, including all line items"),
		mcp.WithString("id",
			mcp.Required(),
			mcp.Description("Invoice ID (e.g., 'inv_123')"),
		),
	)

	return tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if service == nil {
			return mcp.NewToolResultError("invoice service is required"), nil
		}

		id := strings.TrimSpace(req.GetString("id", ""))
		if id == "" {
			return mcp.NewToolResultError("id argument is required"), nil
		}

		result, err := service.GetInvoice(ctx, id)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultStructured(result, invoiceShowText(result)), nil
	}
}

func invoiceListTool(service InvoiceServiceProvider, logger *slog.Logger) (mcp.Tool, func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
	tool := mcp.NewTool("invoice.list",
		mcp.WithDescription("List invoices for a customer profile (summary view, no line items)"),
		mcp.WithString("customer_profile_id",
			mcp.Required(),
			mcp.Description("Customer profile ID (e.g., 'cus_123')"),
		),
		mcp.WithString("status",
			mcp.Description("Optional status filter: draft, issued, discarded"),
		),
	)

	return tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if service == nil {
			return mcp.NewToolResultError("invoice service is required"), nil
		}

		customerProfileID := strings.TrimSpace(req.GetString("customer_profile_id", ""))
		if customerProfileID == "" {
			return mcp.NewToolResultError("customer_profile_id argument is required"), nil
		}
		statusFilter := strings.TrimSpace(req.GetString("status", ""))

		results, err := service.ListInvoices(ctx, customerProfileID, statusFilter)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		// Ensure empty slice (not nil) for structured output
		if results == nil {
			results = []app.InvoiceSummaryDTO{}
		}
		return mcp.NewToolResultStructured(results, invoiceListText(results, customerProfileID, statusFilter)), nil
	}
}

func invoiceShowText(inv app.InvoiceDTO) string {
	var b strings.Builder
	b.WriteString("Invoice\n")
	b.WriteString("───────\n")
	b.WriteString(fmt.Sprintf("ID: %s\n", inv.ID))
	b.WriteString(fmt.Sprintf("Number: %s\n", inv.InvoiceNumber))
	b.WriteString(fmt.Sprintf("Customer: %s\n", inv.CustomerID))
	b.WriteString(fmt.Sprintf("Status: %s\n", inv.Status))
	b.WriteString(fmt.Sprintf("Currency: %s\n", inv.Currency))
	b.WriteString("\n")
	b.WriteString("Lines\n")
	b.WriteString("─────\n")
	b.WriteString(fmt.Sprintf("%-26s %-10s %-12s %s\n", "Description", "Qty(min)", "Rate", "Total"))
	for _, line := range inv.Lines {
		b.WriteString(fmt.Sprintf("%-26s %-10d %-7d %s  %-7d %s\n",
			line.Description,
			line.QuantityMin,
			line.UnitRateAmount, line.UnitRateCurrency,
			line.LineTotalAmount, line.LineTotalCurrency,
		))
	}
	b.WriteString("\n")
	b.WriteString("Totals\n")
	b.WriteString("──────\n")
	b.WriteString(fmt.Sprintf("Subtotal: %d %s\n", inv.Subtotal, inv.Currency))
	b.WriteString(fmt.Sprintf("Grand Total: %d %s\n", inv.GrandTotal, inv.Currency))
	return b.String()
}

func invoiceListText(summaries []app.InvoiceSummaryDTO, customerID, statusFilter string) string {
	if len(summaries) == 0 {
		return "No invoices found.\n"
	}

	var b strings.Builder
	b.WriteString("Invoices\n")
	b.WriteString("────────\n")
	b.WriteString(fmt.Sprintf("Customer: %s\n", customerID))
	if statusFilter != "" {
		b.WriteString(fmt.Sprintf("Status: %s\n", statusFilter))
	}
	b.WriteString(fmt.Sprintf("Count: %d\n", len(summaries)))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("%-16s %-12s %-10s %s\n", "Number", "Status", "Currency", "Total"))
	for _, s := range summaries {
		num := s.InvoiceNumber
		if num == "" {
			num = "—"
		}
		b.WriteString(fmt.Sprintf("%-16s %-12s %-10s %d\n", num, s.Status, s.Currency, s.GrandTotal))
	}
	return b.String()
}
