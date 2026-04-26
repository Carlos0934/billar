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
	pdfArg          *app.RenderInvoicePDFCommand
	pdfRes          app.RenderedFileDTO
	pdfErr          error
	addLineArg      *app.AddDraftLineCommand
	addLineRes      app.InvoiceDTO
	addLineErr      error
	removeLineArg   *app.RemoveDraftLineCommand
	removeLineRes   app.InvoiceDTO
	removeLineErr   error
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

func (s *invoiceServiceStub) RenderInvoicePDF(ctx context.Context, cmd app.RenderInvoicePDFCommand) (app.RenderedFileDTO, error) {
	_ = ctx
	s.pdfArg = &cmd
	return s.pdfRes, s.pdfErr
}

func (s *invoiceServiceStub) AddDraftLine(ctx context.Context, cmd app.AddDraftLineCommand) (app.InvoiceDTO, error) {
	_ = ctx
	s.addLineArg = &cmd
	return s.addLineRes, s.addLineErr
}

func (s *invoiceServiceStub) RemoveDraftLine(ctx context.Context, cmd app.RemoveDraftLineCommand) (app.InvoiceDTO, error) {
	_ = ctx
	s.removeLineArg = &cmd
	return s.removeLineRes, s.removeLineErr
}

func TestInvoiceRenderPDFToolHandler(t *testing.T) {
	t.Parallel()
	svc := &invoiceServiceStub{pdfRes: app.RenderedFileDTO{InvoiceID: "inv_abc", Filename: "inv_abc.pdf", Path: "/tmp/exports/inv_abc.pdf", MimeType: "application/pdf", SizeBytes: 1234}}
	_, handler := invoiceRenderPDFTool(svc, nil)
	result, err := handler(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Name: "invoice.render_pdf", Arguments: map[string]any{"invoice_id": "inv_abc", "filename": "inv_abc.pdf"}}})
	if err != nil {
		t.Fatalf("handler error = %v", err)
	}
	if result == nil || result.IsError {
		t.Fatalf("result = %+v, want success", result)
	}
	if svc.pdfArg == nil || svc.pdfArg.InvoiceID != "inv_abc" || svc.pdfArg.Filename != "inv_abc.pdf" {
		t.Fatalf("pdf arg = %+v", svc.pdfArg)
	}
	if result.StructuredContent == nil {
		t.Fatal("StructuredContent must not be nil")
	}
	raw, _ := json.Marshal(result.StructuredContent)
	var dto app.RenderedFileDTO
	if err := json.Unmarshal(raw, &dto); err != nil {
		t.Fatalf("structured content decode: %v", err)
	}
	if dto.Path != "/tmp/exports/inv_abc.pdf" || dto.MimeType != "application/pdf" || dto.SizeBytes != 1234 {
		t.Fatalf("dto = %+v", dto)
	}
}

func TestInvoiceRenderPDFToolHandlerErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		args map[string]any
		svc  *invoiceServiceStub
		want string
	}{
		{"missing invoice id", map[string]any{}, &invoiceServiceStub{}, "invoice_id"},
		{"absolute path", map[string]any{"invoice_id": "inv_abc", "output_path": "/etc/passwd.pdf"}, &invoiceServiceStub{pdfErr: errors.New("output path must be relative to export root")}, "relative"},
		{"traversal", map[string]any{"invoice_id": "inv_abc", "output_path": "../../etc/x.pdf"}, &invoiceServiceStub{pdfErr: errors.New("output path escapes export root")}, "escapes export root"},
		{"filename separators", map[string]any{"invoice_id": "inv_abc", "filename": "a/b.pdf"}, &invoiceServiceStub{pdfErr: errors.New("filename must not contain path separators")}, "path separators"},
		{"unset root", map[string]any{"invoice_id": "inv_abc"}, &invoiceServiceStub{pdfErr: errors.New("pdf export disabled: BILLAR_EXPORT_DIR not configured")}, "BILLAR_EXPORT_DIR"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, handler := invoiceRenderPDFTool(tc.svc, nil)
			result, err := handler(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Name: "invoice.render_pdf", Arguments: tc.args}})
			if err != nil {
				t.Fatalf("handler error = %v", err)
			}
			if result == nil || !result.IsError {
				t.Fatalf("result = %+v, want error", result)
			}
			if !strings.Contains(mcp.GetTextFromContent(result.Content[0]), tc.want) {
				t.Fatalf("error = %q, want %q", mcp.GetTextFromContent(result.Content[0]), tc.want)
			}
		})
	}
}

func TestInvoiceRenderPDFToolHandlerDefaultFilename(t *testing.T) {
	t.Parallel()
	svc := &invoiceServiceStub{pdfRes: app.RenderedFileDTO{InvoiceID: "inv_abc", Filename: "invoice-inv_abc.pdf", Path: "/tmp/exports/invoice-inv_abc.pdf", MimeType: "application/pdf", SizeBytes: 12}}
	_, handler := invoiceRenderPDFTool(svc, nil)
	result, err := handler(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Name: "invoice.render_pdf", Arguments: map[string]any{"invoice_id": "inv_abc"}}})
	if err != nil {
		t.Fatalf("handler error = %v", err)
	}
	if result == nil || result.IsError {
		t.Fatalf("result = %+v, want success", result)
	}
	if svc.pdfArg == nil || svc.pdfArg.Filename != "" || svc.pdfArg.OutputPath != "" {
		t.Fatalf("pdf arg = %+v, want service to synthesize default", svc.pdfArg)
	}
	if !strings.Contains(mcp.GetTextFromContent(result.Content[0]), "invoice-inv_abc.pdf") {
		t.Fatalf("text = %q", mcp.GetTextFromContent(result.Content[0]))
	}
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
				draftRes: app.InvoiceDTO{ID: "inv_abc", CustomerID: "cus_1", Status: "draft", IsDraft: true, PeriodStart: "2026-04-01T00:00:00Z", PeriodEnd: "2026-04-30T00:00:00Z", DueDate: "2026-05-15T00:00:00Z", Notes: "Net 15"},
			},
			arguments:  map[string]any{"customer_profile_id": "cus_1", "period_start": "2026-04-01", "period_end": "2026-04-30", "due_date": "2026-05-15", "notes": "Net 15"},
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
			_, handler := invoiceDraftTool(svc, nil)
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
			if tc.service != nil && tc.service.draftArg != nil && tc.service.draftArg.PeriodStart != "2026-04-01" {
				t.Fatalf("draft arg = %+v, want metadata args", tc.service.draftArg)
			}
		})
	}
}

func TestInvoiceLineToolHandlers(t *testing.T) {
	t.Parallel()

	updated := app.InvoiceDTO{ID: "inv_001", CustomerID: "cus_1", Status: "draft", Currency: "USD", IsDraft: true, Lines: []app.InvoiceLineDTO{{ID: "inl_manual", Description: "Setup fee", QuantityMin: 60, UnitRateAmount: 5000, UnitRateCurrency: "USD", LineTotalAmount: 5000, LineTotalCurrency: "USD"}}, Subtotal: 5000, GrandTotal: 5000}
	t.Run("invoice.line_add returns canonical dto", func(t *testing.T) {
		t.Parallel()
		svc := &invoiceServiceStub{addLineRes: updated}
		_, handler := invoiceLineAddTool(svc, nil)
		result, err := handler(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Name: "invoice.line_add", Arguments: map[string]any{"invoice_id": "inv_001", "description": "Setup fee", "quantity_min": float64(60), "unit_rate_amount": float64(5000), "currency": "USD"}}})
		if err != nil {
			t.Fatalf("handler error = %v", err)
		}
		if result == nil || result.IsError || result.StructuredContent == nil {
			t.Fatalf("result = %+v, want structured success", result)
		}
		if svc.addLineArg == nil || svc.addLineArg.InvoiceID != "inv_001" || svc.addLineArg.Description != "Setup fee" || svc.addLineArg.QuantityMin != 60 || svc.addLineArg.UnitRate != 5000 || svc.addLineArg.Currency != "USD" {
			t.Fatalf("add arg = %+v", svc.addLineArg)
		}
		raw, _ := json.Marshal(result.StructuredContent)
		var dto app.InvoiceDTO
		if err := json.Unmarshal(raw, &dto); err != nil {
			t.Fatalf("structured decode: %v", err)
		}
		if dto.GrandTotal != 5000 || len(dto.Lines) != 1 || dto.Lines[0].ID != "inl_manual" {
			t.Fatalf("dto = %+v, want updated invoice", dto)
		}
	})
	t.Run("invoice.line_remove returns canonical dto", func(t *testing.T) {
		t.Parallel()
		svc := &invoiceServiceStub{removeLineRes: updated}
		_, handler := invoiceLineRemoveTool(svc, nil)
		result, err := handler(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Name: "invoice.line_remove", Arguments: map[string]any{"invoice_id": "inv_001", "invoice_line_id": "inl_manual"}}})
		if err != nil {
			t.Fatalf("handler error = %v", err)
		}
		if result == nil || result.IsError || result.StructuredContent == nil {
			t.Fatalf("result = %+v, want structured success", result)
		}
		if svc.removeLineArg == nil || svc.removeLineArg.InvoiceID != "inv_001" || svc.removeLineArg.InvoiceLineID != "inl_manual" {
			t.Fatalf("remove arg = %+v", svc.removeLineArg)
		}
	})
	t.Run("errors", func(t *testing.T) {
		t.Parallel()
		_, addHandler := invoiceLineAddTool(&invoiceServiceStub{addLineErr: errors.New("invoice is not draft")}, nil)
		addResult, err := addHandler(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Name: "invoice.line_add", Arguments: map[string]any{"invoice_id": "inv_001", "description": "Setup", "quantity_min": float64(60), "unit_rate_amount": float64(5000), "currency": "USD"}}})
		if err != nil || addResult == nil || !addResult.IsError || !strings.Contains(mcp.GetTextFromContent(addResult.Content[0]), "not draft") {
			t.Fatalf("add error result = %+v err=%v", addResult, err)
		}
		_, removeHandler := invoiceLineRemoveTool(&invoiceServiceStub{removeLineErr: errors.New("cannot remove last line")}, nil)
		removeResult, err := removeHandler(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Name: "invoice.line_remove", Arguments: map[string]any{"invoice_id": "inv_001", "invoice_line_id": "inl_last"}}})
		if err != nil || removeResult == nil || !removeResult.IsError || !strings.Contains(mcp.GetTextFromContent(removeResult.Content[0]), "last line") {
			t.Fatalf("remove error result = %+v err=%v", removeResult, err)
		}
	})
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
			_, handler := invoiceIssueTool(svc, nil)
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
			_, handler := invoiceDiscardTool(svc, nil)
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
		ID: "inv_abc", InvoiceNumber: "INV-001", CustomerID: "cus_1", Status: "issued", Currency: "USD", PeriodStart: "2026-04-01T00:00:00Z", PeriodEnd: "2026-04-30T00:00:00Z", DueDate: "2026-05-15T00:00:00Z", Notes: "Net 15",
		Lines: []app.InvoiceLineDTO{
			{Description: "Consulting", QuantityMin: 90, UnitRateAmount: 10000, UnitRateCurrency: "USD", LineTotalAmount: 15000, LineTotalCurrency: "USD"},
		},
		Subtotal: 15000, GrandTotal: 15000,
	}
	svc := &invoiceServiceStub{getInvoiceRes: wantDTO}
	_, handler := invoiceGetTool(svc, nil)
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
	if gotDTO.PeriodStart != wantDTO.PeriodStart || gotDTO.PeriodEnd != wantDTO.PeriodEnd || gotDTO.DueDate != wantDTO.DueDate || gotDTO.Notes != wantDTO.Notes {
		t.Errorf("StructuredContent metadata = (%q,%q,%q,%q), want (%q,%q,%q,%q)", gotDTO.PeriodStart, gotDTO.PeriodEnd, gotDTO.DueDate, gotDTO.Notes, wantDTO.PeriodStart, wantDTO.PeriodEnd, wantDTO.DueDate, wantDTO.Notes)
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
			_, handler := invoiceGetTool(svc, nil)
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
		{ID: "inv_001", InvoiceNumber: "INV-001", CustomerID: "cus_1", Status: "issued", Currency: "USD", PeriodStart: "2026-04-01T00:00:00Z", PeriodEnd: "2026-04-30T00:00:00Z", DueDate: "2026-05-15T00:00:00Z", GrandTotal: 5000},
		{ID: "inv_002", InvoiceNumber: "", CustomerID: "cus_1", Status: "draft", Currency: "USD", GrandTotal: 15000},
	}
	svc := &invoiceServiceStub{listInvoicesRes: summaries}
	_, handler := invoiceListTool(svc, nil)
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
	if gotSummaries[0].PeriodStart != "2026-04-01T00:00:00Z" || gotSummaries[0].PeriodEnd != "2026-04-30T00:00:00Z" || gotSummaries[0].DueDate != "2026-05-15T00:00:00Z" {
		t.Errorf("StructuredContent[0] metadata = (%q,%q,%q), want persisted period and due date", gotSummaries[0].PeriodStart, gotSummaries[0].PeriodEnd, gotSummaries[0].DueDate)
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
	_, handler := invoiceListTool(svc, nil)
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
			_, handler := invoiceListTool(svc, nil)
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
