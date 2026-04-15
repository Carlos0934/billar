package mcp

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Carlos0934/billar/internal/app"
	"github.com/mark3labs/mcp-go/mcp"
)

type invoiceServiceStub struct {
	draftArg   *app.CreateDraftFromUnbilledCommand
	draftRes   app.InvoiceDTO
	draftErr   error
	issueArg   *app.IssueInvoiceCommand
	issueRes   app.InvoiceDTO
	issueErr   error
	discardID  string
	discardRes app.DiscardResult
	discardErr error
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
