package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/Carlos0934/billar/internal/app"
	"github.com/mark3labs/mcp-go/mcp"
)

type invoiceServiceStub struct {
	draftArg        *app.CreateDraftFromUnbilledCommand
	draftRes        app.InvoiceDTO
	draftErr        error
	issueArg        *app.IssueInvoiceCommand
	issueRes        app.InvoiceDTO
	issueErr        error
	discardID       string
	discardRes      app.DiscardResult
	discardErr      error
	getInvoiceRes   app.InvoiceDTO
	getInvoiceErr   error
	listInvoicesRes []app.InvoiceSummaryDTO
	listInvoicesErr error
}

func (s *invoiceServiceStub) CreateDraftFromUnbilled(ctx context.Context, cmd app.CreateDraftFromUnbilledCommand) (app.InvoiceDTO, error) {
	_ = ctx
	s.draftArg = &cmd
	return s.draftRes, s.draftErr
}

func (s *invoiceServiceStub) IssueDraft(ctx context.Context, cmd app.IssueInvoiceCommand) (app.InvoiceDTO, error) {
	_ = ctx
	s.issueArg = &cmd
	return s.issueRes, s.issueErr
}

func (s *invoiceServiceStub) Discard(ctx context.Context, id string) (app.DiscardResult, error) {
	_ = ctx
	s.discardID = id
	return s.discardRes, s.discardErr
}

func (s *invoiceServiceStub) GetInvoice(ctx context.Context, id string) (app.InvoiceDTO, error) {
	_ = ctx
	return s.getInvoiceRes, s.getInvoiceErr
}

func (s *invoiceServiceStub) ListInvoices(ctx context.Context, customerID string, statusFilter string) ([]app.InvoiceSummaryDTO, error) {
	_ = ctx
	return s.listInvoicesRes, s.listInvoicesErr
}

// -- invoice.draft --

func TestInvoiceDraftToolHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		service       *invoiceServiceStub
		arguments     map[string]any
		wantErr       bool
		wantErrSubstr string
		wantResult    string
	}{
		{
			name: "creates draft successfully",
			service: &invoiceServiceStub{
				draftRes: app.InvoiceDTO{ID: "inv_abc", CustomerID: "cus_1", Status: "draft", IsDraft: true},
			},
			arguments:  map[string]any{"customer_profile_id": "cus_1"},
			wantResult: "inv_abc",
		},
		{
			name:          "returns error when customer_profile_id is empty",
			service:       &invoiceServiceStub{},
			arguments:     map[string]any{},
			wantErr:       true,
			wantErrSubstr: "customer_profile_id",
		},
		{
			name:          "returns tool error when service is nil",
			service:       nil,
			arguments:     map[string]any{"customer_profile_id": "cus_1"},
			wantErr:       true,
			wantErrSubstr: "invoice service is required",
		},
		{
			name: "propagates service error",
			service: &invoiceServiceStub{
				draftErr: errors.New("no unbilled time entries"),
			},
			arguments:     map[string]any{"customer_profile_id": "cus_1"},
			wantErr:       true,
			wantErrSubstr: "no unbilled time entries",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var svc InvoiceServiceProvider
			if tc.service != nil {
				svc = tc.service
			}
			_, handler := invoiceDraftTool(svc, NewIngressGuard(nil), nil)
			result, err := handler(context.Background(), mcp.CallToolRequest{
				Params: mcp.CallToolParams{Name: "invoice.draft", Arguments: tc.arguments},
			})
			if err != nil {
				t.Fatalf("handler error = %v", err)
			}

			if tc.wantErr {
				if result == nil || !result.IsError {
					t.Fatalf("handler result = %+v, want error result", result)
				}
				if tc.wantErrSubstr != "" && !strings.Contains(mcp.GetTextFromContent(result.Content[0]), tc.wantErrSubstr) {
					t.Fatalf("handler error = %q, want substring %q", mcp.GetTextFromContent(result.Content[0]), tc.wantErrSubstr)
				}
				return
			}
			if result == nil || result.IsError {
				t.Fatalf("handler result = %+v, want success result", result)
			}
			if tc.wantResult != "" && !strings.Contains(mcp.GetTextFromContent(result.Content[0]), tc.wantResult) {
				t.Fatalf("handler result = %q, want contains %q", mcp.GetTextFromContent(result.Content[0]), tc.wantResult)
			}
		})
	}
}

// -- invoice.issue --

func TestInvoiceIssueToolHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		service       *invoiceServiceStub
		arguments     map[string]any
		wantErr       bool
		wantErrSubstr string
		wantResult    string
	}{
		{
			name: "issues invoice successfully",
			service: &invoiceServiceStub{
				issueRes: app.InvoiceDTO{ID: "inv_abc", InvoiceNumber: "INV-2026-0001", Status: "issued", IsIssued: true},
			},
			arguments:  map[string]any{"id": "inv_abc"},
			wantResult: "INV-2026-0001",
		},
		{
			name:          "returns error when id is empty",
			service:       &invoiceServiceStub{},
			arguments:     map[string]any{},
			wantErr:       true,
			wantErrSubstr: "id",
		},
		{
			name: "propagates service error",
			service: &invoiceServiceStub{
				issueErr: errors.New("invoice is not draft"),
			},
			arguments:     map[string]any{"id": "inv_abc"},
			wantErr:       true,
			wantErrSubstr: "not draft",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var svc InvoiceServiceProvider
			if tc.service != nil {
				svc = tc.service
			}
			_, handler := invoiceIssueTool(svc, NewIngressGuard(nil), nil)
			result, err := handler(context.Background(), mcp.CallToolRequest{
				Params: mcp.CallToolParams{Name: "invoice.issue", Arguments: tc.arguments},
			})
			if err != nil {
				t.Fatalf("handler error = %v", err)
			}

			if tc.wantErr {
				if result == nil || !result.IsError {
					t.Fatalf("handler result = %+v, want error result", result)
				}
				if tc.wantErrSubstr != "" && !strings.Contains(mcp.GetTextFromContent(result.Content[0]), tc.wantErrSubstr) {
					t.Fatalf("handler error = %q, want substring %q", mcp.GetTextFromContent(result.Content[0]), tc.wantErrSubstr)
				}
				return
			}
			if result == nil || result.IsError {
				t.Fatalf("handler result = %+v, want success result", result)
			}
			if tc.wantResult != "" && !strings.Contains(mcp.GetTextFromContent(result.Content[0]), tc.wantResult) {
				t.Fatalf("handler result = %q, want contains %q", mcp.GetTextFromContent(result.Content[0]), tc.wantResult)
			}
		})
	}
}

// -- invoice.discard --

func TestInvoiceDiscardToolHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		service       *invoiceServiceStub
		arguments     map[string]any
		wantErr       bool
		wantErrSubstr string
		wantResult    string
	}{
		{
			name: "hard-discard draft successfully",
			service: &invoiceServiceStub{
				discardRes: app.DiscardResult{WasSoftDiscard: false},
			},
			arguments:  map[string]any{"id": "inv_abc"},
			wantResult: "deleted",
		},
		{
			name: "soft-discard issued shows warning",
			service: &invoiceServiceStub{
				discardRes: app.DiscardResult{WasSoftDiscard: true, InvoiceNumber: "INV-2026-0001"},
			},
			arguments:  map[string]any{"id": "inv_abc"},
			wantResult: "INV-2026-0001",
		},
		{
			name:          "returns error when id is empty",
			service:       &invoiceServiceStub{},
			arguments:     map[string]any{},
			wantErr:       true,
			wantErrSubstr: "id",
		},
		{
			name: "propagates service error",
			service: &invoiceServiceStub{
				discardErr: errors.New("invoice is already discarded"),
			},
			arguments:     map[string]any{"id": "inv_abc"},
			wantErr:       true,
			wantErrSubstr: "already discarded",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var svc InvoiceServiceProvider
			if tc.service != nil {
				svc = tc.service
			}
			_, handler := invoiceDiscardTool(svc, NewIngressGuard(nil), nil)
			result, err := handler(context.Background(), mcp.CallToolRequest{
				Params: mcp.CallToolParams{Name: "invoice.discard", Arguments: tc.arguments},
			})
			if err != nil {
				t.Fatalf("handler error = %v", err)
			}

			if tc.wantErr {
				if result == nil || !result.IsError {
					t.Fatalf("handler result = %+v, want error result", result)
				}
				if tc.wantErrSubstr != "" && !strings.Contains(mcp.GetTextFromContent(result.Content[0]), tc.wantErrSubstr) {
					t.Fatalf("handler error = %q, want substring %q", mcp.GetTextFromContent(result.Content[0]), tc.wantErrSubstr)
				}
				return
			}
			if result == nil || result.IsError {
				t.Fatalf("handler result = %+v, want success result", result)
			}
			if tc.wantResult != "" && !strings.Contains(mcp.GetTextFromContent(result.Content[0]), tc.wantResult) {
				t.Fatalf("handler result = %q, want contains %q", mcp.GetTextFromContent(result.Content[0]), tc.wantResult)
			}
		})
	}
}

// -- invoice.get --

func TestInvoiceGetToolHandler_StructuredDTO(t *testing.T) {
	t.Parallel()

	wantDTO := app.InvoiceDTO{
		ID: "inv_abc", InvoiceNumber: "INV-001", CustomerID: "cus_1", Status: "issued", Currency: "USD",
		Lines: []app.InvoiceLineDTO{
			{Description: "Consulting", QuantityMin: 90, UnitRateAmount: 10000, UnitRateCurrency: "USD", LineTotalAmount: 15000, LineTotalCurrency: "USD"},
		},
		Subtotal: 15000, GrandTotal: 15000,
	}
	svc := &invoiceServiceStub{getInvoiceRes: wantDTO}
	_, handler := invoiceGetTool(svc, NewIngressGuard(nil), nil)
	result, err := handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{Name: "invoice.get", Arguments: map[string]any{"id": "inv_abc"}},
	})
	if err != nil {
		t.Fatalf("unexpected handler error: %v", err)
	}
	if result == nil || result.IsError {
		t.Fatalf("expected success result, got: %+v", result)
	}
	// StructuredContent must be the canonical InvoiceDTO — assert full field equality
	if result.StructuredContent == nil {
		t.Fatal("StructuredContent must not be nil — invoice.get must return canonical DTO")
	}
	// Round-trip through JSON to compare DTO values regardless of underlying type
	raw, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatalf("StructuredContent not JSON-serialisable: %v", err)
	}
	var gotDTO app.InvoiceDTO
	if err := json.Unmarshal(raw, &gotDTO); err != nil {
		t.Fatalf("StructuredContent could not be decoded as InvoiceDTO: %v", err)
	}
	if gotDTO.ID != wantDTO.ID {
		t.Errorf("StructuredContent.ID = %q, want %q", gotDTO.ID, wantDTO.ID)
	}
	if gotDTO.InvoiceNumber != wantDTO.InvoiceNumber {
		t.Errorf("StructuredContent.InvoiceNumber = %q, want %q", gotDTO.InvoiceNumber, wantDTO.InvoiceNumber)
	}
	if gotDTO.CustomerID != wantDTO.CustomerID {
		t.Errorf("StructuredContent.CustomerID = %q, want %q", gotDTO.CustomerID, wantDTO.CustomerID)
	}
	if gotDTO.Status != wantDTO.Status {
		t.Errorf("StructuredContent.Status = %q, want %q", gotDTO.Status, wantDTO.Status)
	}
	if gotDTO.GrandTotal != wantDTO.GrandTotal {
		t.Errorf("StructuredContent.GrandTotal = %d, want %d", gotDTO.GrandTotal, wantDTO.GrandTotal)
	}
	if len(gotDTO.Lines) != 1 || gotDTO.Lines[0].Description != "Consulting" {
		t.Errorf("StructuredContent.Lines = %+v, want one Consulting line", gotDTO.Lines)
	}
	// Text fallback must contain full show layout
	textFallback := mcp.GetTextFromContent(result.Content[0])
	if !strings.Contains(textFallback, "Invoice\n") {
		t.Fatalf("text fallback must start with 'Invoice\\n', got: %q", textFallback)
	}
	if !strings.Contains(textFallback, "ID: inv_abc") {
		t.Fatalf("text fallback must contain 'ID: inv_abc', got: %q", textFallback)
	}
	if !strings.Contains(textFallback, "Grand Total: 15000 USD") {
		t.Fatalf("text fallback must contain 'Grand Total: 15000 USD', got: %q", textFallback)
	}
}

func TestInvoiceGetToolHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		service       *invoiceServiceStub
		arguments     map[string]any
		wantErr       bool
		wantErrSubstr string
		wantResult    string
	}{
		{
			name: "returns invoice details",
			service: &invoiceServiceStub{
				getInvoiceRes: app.InvoiceDTO{
					ID: "inv_abc", InvoiceNumber: "INV-001", CustomerID: "cus_1", Status: "issued", Currency: "USD",
					Lines: []app.InvoiceLineDTO{
						{Description: "Consulting", QuantityMin: 90, UnitRateAmount: 10000, UnitRateCurrency: "USD", LineTotalAmount: 15000, LineTotalCurrency: "USD"},
					},
					Subtotal: 15000, GrandTotal: 15000,
				},
			},
			arguments:  map[string]any{"id": "inv_abc"},
			wantResult: "Invoice",
		},
		{
			name:          "returns error when id is empty",
			service:       &invoiceServiceStub{},
			arguments:     map[string]any{},
			wantErr:       true,
			wantErrSubstr: "id",
		},
		{
			name:          "propagates service error",
			service:       &invoiceServiceStub{getInvoiceErr: errors.New("not found")},
			arguments:     map[string]any{"id": "inv_abc"},
			wantErr:       true,
			wantErrSubstr: "not found",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var svc InvoiceServiceProvider
			if tc.service != nil {
				svc = tc.service
			}
			_, handler := invoiceGetTool(svc, NewIngressGuard(nil), nil)
			result, err := handler(context.Background(), mcp.CallToolRequest{
				Params: mcp.CallToolParams{Name: "invoice.get", Arguments: tc.arguments},
			})
			if err != nil {
				t.Fatalf("handler error = %v", err)
			}

			if tc.wantErr {
				if result == nil || !result.IsError {
					t.Fatalf("handler result = %+v, want error result", result)
				}
				if tc.wantErrSubstr != "" && !strings.Contains(mcp.GetTextFromContent(result.Content[0]), tc.wantErrSubstr) {
					t.Fatalf("handler error = %q, want substring %q", mcp.GetTextFromContent(result.Content[0]), tc.wantErrSubstr)
				}
				return
			}
			if result == nil || result.IsError {
				t.Fatalf("handler result = %+v, want success result", result)
			}
			if tc.wantResult != "" && !strings.Contains(mcp.GetTextFromContent(result.Content[0]), tc.wantResult) {
				t.Fatalf("handler result = %q, want contains %q", mcp.GetTextFromContent(result.Content[0]), tc.wantResult)
			}
		})
	}
}

// -- invoice.list --

func TestInvoiceListToolHandler_StructuredDTO(t *testing.T) {
	t.Parallel()

	summaries := []app.InvoiceSummaryDTO{
		{ID: "inv_001", InvoiceNumber: "INV-001", CustomerID: "cus_1", Status: "issued", Currency: "USD", GrandTotal: 5000},
		{ID: "inv_002", InvoiceNumber: "", CustomerID: "cus_1", Status: "draft", Currency: "USD", GrandTotal: 15000},
	}
	svc := &invoiceServiceStub{listInvoicesRes: summaries}
	_, handler := invoiceListTool(svc, NewIngressGuard(nil), nil)
	result, err := handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{Name: "invoice.list", Arguments: map[string]any{"customer_profile_id": "cus_1"}},
	})
	if err != nil {
		t.Fatalf("unexpected handler error: %v", err)
	}
	if result == nil || result.IsError {
		t.Fatalf("expected success result, got: %+v", result)
	}
	// StructuredContent must be the canonical []InvoiceSummaryDTO — assert full field equality
	if result.StructuredContent == nil {
		t.Fatal("StructuredContent must not be nil — invoice.list must return canonical DTO slice")
	}
	raw, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatalf("StructuredContent not JSON-serialisable: %v", err)
	}
	var gotSummaries []app.InvoiceSummaryDTO
	if err := json.Unmarshal(raw, &gotSummaries); err != nil {
		t.Fatalf("StructuredContent could not be decoded as []InvoiceSummaryDTO: %v", err)
	}
	if len(gotSummaries) != 2 {
		t.Fatalf("StructuredContent len = %d, want 2", len(gotSummaries))
	}
	if gotSummaries[0].ID != "inv_001" || gotSummaries[0].GrandTotal != 5000 {
		t.Errorf("StructuredContent[0] = %+v, want ID=inv_001 GrandTotal=5000", gotSummaries[0])
	}
	if gotSummaries[1].ID != "inv_002" || gotSummaries[1].Status != "draft" {
		t.Errorf("StructuredContent[1] = %+v, want ID=inv_002 Status=draft", gotSummaries[1])
	}
	// Text fallback must contain "Invoices" header and count
	textFallback := mcp.GetTextFromContent(result.Content[0])
	if !strings.Contains(textFallback, "Invoices\n") {
		t.Fatalf("text fallback must contain 'Invoices\\n', got: %q", textFallback)
	}
	if !strings.Contains(textFallback, "Count: 2") {
		t.Fatalf("text fallback must contain 'Count: 2', got: %q", textFallback)
	}
	if !strings.Contains(textFallback, "INV-001") {
		t.Fatalf("text fallback must contain 'INV-001', got: %q", textFallback)
	}
}

func TestInvoiceListToolHandler_EmptyStructuredDTO(t *testing.T) {
	t.Parallel()

	svc := &invoiceServiceStub{listInvoicesRes: nil}
	_, handler := invoiceListTool(svc, NewIngressGuard(nil), nil)
	result, err := handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{Name: "invoice.list", Arguments: map[string]any{"customer_profile_id": "cus_999"}},
	})
	if err != nil {
		t.Fatalf("unexpected handler error: %v", err)
	}
	if result == nil || result.IsError {
		t.Fatalf("expected success result, got: %+v", result)
	}
	// Even for empty lists, StructuredContent must be set (empty slice, not nil)
	if result.StructuredContent == nil {
		t.Fatal("StructuredContent must not be nil for empty list — must return empty slice DTO")
	}
	// Text fallback must be empty state
	textFallback := mcp.GetTextFromContent(result.Content[0])
	if !strings.Contains(textFallback, "No invoices found.") {
		t.Fatalf("text fallback must contain 'No invoices found.', got: %q", textFallback)
	}
}

func TestInvoiceListToolHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		service       *invoiceServiceStub
		arguments     map[string]any
		wantErr       bool
		wantErrSubstr string
		wantResult    string
	}{
		{
			name: "returns invoice summaries",
			service: &invoiceServiceStub{
				listInvoicesRes: []app.InvoiceSummaryDTO{
					{ID: "inv_001", InvoiceNumber: "INV-001", CustomerID: "cus_1", Status: "issued", Currency: "USD", GrandTotal: 5000},
				},
			},
			arguments:  map[string]any{"customer_profile_id": "cus_1"},
			wantResult: "Invoices",
		},
		{
			name:       "returns empty state",
			service:    &invoiceServiceStub{listInvoicesRes: nil},
			arguments:  map[string]any{"customer_profile_id": "cus_999"},
			wantResult: "No invoices found.",
		},
		{
			name:          "returns error when customer_profile_id is empty",
			service:       &invoiceServiceStub{},
			arguments:     map[string]any{},
			wantErr:       true,
			wantErrSubstr: "customer_profile_id",
		},
		{
			name:          "propagates service error",
			service:       &invoiceServiceStub{listInvoicesErr: errors.New("store failure")},
			arguments:     map[string]any{"customer_profile_id": "cus_1"},
			wantErr:       true,
			wantErrSubstr: "store failure",
		},
		{
			name:          "propagates invalid status filter error",
			service:       &invoiceServiceStub{listInvoicesErr: errors.New("invalid invoice status filter")},
			arguments:     map[string]any{"customer_profile_id": "cus_1", "status": "pending"},
			wantErr:       true,
			wantErrSubstr: "invalid invoice status filter",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var svc InvoiceServiceProvider
			if tc.service != nil {
				svc = tc.service
			}
			_, handler := invoiceListTool(svc, NewIngressGuard(nil), nil)
			result, err := handler(context.Background(), mcp.CallToolRequest{
				Params: mcp.CallToolParams{Name: "invoice.list", Arguments: tc.arguments},
			})
			if err != nil {
				t.Fatalf("handler error = %v", err)
			}

			if tc.wantErr {
				if result == nil || !result.IsError {
					t.Fatalf("handler result = %+v, want error result", result)
				}
				if tc.wantErrSubstr != "" && !strings.Contains(mcp.GetTextFromContent(result.Content[0]), tc.wantErrSubstr) {
					t.Fatalf("handler error = %q, want substring %q", mcp.GetTextFromContent(result.Content[0]), tc.wantErrSubstr)
				}
				return
			}
			if result == nil || result.IsError {
				t.Fatalf("handler result = %+v, want success result", result)
			}
			if tc.wantResult != "" && !strings.Contains(mcp.GetTextFromContent(result.Content[0]), tc.wantResult) {
				t.Fatalf("handler result = %q, want contains %q", mcp.GetTextFromContent(result.Content[0]), tc.wantResult)
			}
		})
	}
}
