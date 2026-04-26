package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/Carlos0934/billar/internal/app"
	"github.com/mark3labs/mcp-go/mcp"
)

func assertStructuredContent[T any](t *testing.T, result *mcp.CallToolResult, want T) T {
	t.Helper()
	if result == nil || result.IsError {
		t.Fatalf("result = %+v, want success", result)
	}
	if result.StructuredContent == nil {
		t.Fatal("StructuredContent = nil, want payload")
	}
	text := mcp.GetTextFromContent(result.Content[0])
	if strings.TrimSpace(text) == "" {
		t.Fatal("text fallback is empty")
	}
	raw, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatalf("marshal structured content: %v", err)
	}
	var got T
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("decode structured content: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("structured content = %+v, want %+v", got, want)
	}
	return got
}

func assertStructuredSlice[T any](t *testing.T, result *mcp.CallToolResult, want []T) []T {
	t.Helper()
	if result == nil || result.IsError {
		t.Fatalf("result = %+v, want success", result)
	}
	if result.StructuredContent == nil {
		t.Fatal("StructuredContent = nil, want payload")
	}
	text := mcp.GetTextFromContent(result.Content[0])
	if strings.TrimSpace(text) == "" {
		t.Fatal("text fallback is empty")
	}
	raw, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatalf("marshal structured content: %v", err)
	}
	if len(want) == 0 && string(raw) != "[]" {
		t.Fatalf("empty structured slice JSON = %s, want []", raw)
	}
	var got []T
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("decode structured content: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("structured content = %+v, want %+v", got, want)
	}
	return got
}

func assertTextOnlyError(t *testing.T, result *mcp.CallToolResult, wantText string) {
	t.Helper()
	if result == nil || !result.IsError {
		t.Fatalf("result = %+v, want error", result)
	}
	if result.StructuredContent != nil {
		t.Fatalf("StructuredContent = %+v, want nil for error", result.StructuredContent)
	}
	if got := mcp.GetTextFromContent(result.Content[0]); !strings.Contains(got, wantText) {
		t.Fatalf("error text = %q, want contains %q", got, wantText)
	}
}

func TestInvoiceToolsStructuredResults(t *testing.T) {
	t.Parallel()

	draft := app.InvoiceDTO{ID: "inv_draft", CustomerID: "cus_1", Status: "draft", Currency: "USD", IsDraft: true}
	issue := app.InvoiceDTO{ID: "inv_issue", InvoiceNumber: "INV-1", CustomerID: "cus_1", Status: "issued", Currency: "USD", IsIssued: true}
	got := app.InvoiceDTO{ID: "inv_get", CustomerID: "cus_1", Status: "draft", Currency: "USD", Lines: []app.InvoiceLineDTO{{ID: "line_1", Description: "Work"}}}
	summary := app.InvoiceSummaryDTO{ID: "inv_sum", CustomerID: "cus_1", Status: "draft", Currency: "USD", GrandTotal: 12500}
	file := app.RenderedFileDTO{InvoiceID: "inv_pdf", Filename: "inv_pdf.pdf", Path: "/tmp/inv_pdf.pdf", MimeType: "application/pdf", SizeBytes: 42}

	t.Run("draft", func(t *testing.T) {
		svc := &invoiceServiceStub{draftRes: draft}
		_, handler := invoiceDraftTool(svc, nil)
		result, err := handler(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Name: "invoice.draft", Arguments: map[string]any{"customer_profile_id": "cus_1"}}})
		if err != nil {
			t.Fatalf("handler error = %v", err)
		}
		assertStructuredContent(t, result, draft)
	})
	t.Run("issue", func(t *testing.T) {
		svc := &invoiceServiceStub{issueRes: issue}
		_, handler := invoiceIssueTool(svc, nil)
		result, err := handler(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Name: "invoice.issue", Arguments: map[string]any{"id": "inv_issue"}}})
		if err != nil {
			t.Fatalf("handler error = %v", err)
		}
		assertStructuredContent(t, result, issue)
		if gotText, wantText := mcp.GetTextFromContent(result.Content[0]), invoiceIssueText(issue); gotText != wantText {
			t.Fatalf("text = %q, want %q", gotText, wantText)
		}
	})
	t.Run("discard hard and soft", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			id   string
			res  app.DiscardResult
			want InvoiceDiscardAck
		}{
			{name: "hard", id: "inv_hard", res: app.DiscardResult{Invoice: app.InvoiceDTO{ID: "inv_hard"}}, want: InvoiceDiscardAck{ID: "inv_hard", Action: "discarded", Invoice: app.InvoiceDTO{ID: "inv_hard"}}},
			{name: "soft", id: "inv_soft", res: app.DiscardResult{WasSoftDiscard: true, InvoiceNumber: "INV-9", Invoice: app.InvoiceDTO{ID: "inv_soft", Status: "discarded"}}, want: InvoiceDiscardAck{ID: "inv_soft", Action: "soft_discarded", WasSoftDiscard: true, InvoiceNumber: "INV-9", Invoice: app.InvoiceDTO{ID: "inv_soft", Status: "discarded"}}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				svc := &invoiceServiceStub{discardRes: tc.res}
				_, handler := invoiceDiscardTool(svc, nil)
				result, err := handler(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Name: "invoice.discard", Arguments: map[string]any{"id": tc.id}}})
				if err != nil {
					t.Fatalf("handler error = %v", err)
				}
				assertStructuredContent(t, result, tc.want)
			})
		}
	})
	t.Run("get list render and error", func(t *testing.T) {
		svc := &invoiceServiceStub{getInvoiceRes: got, listInvoicesRes: []app.InvoiceSummaryDTO{summary}, pdfRes: file}
		_, getHandler := invoiceGetTool(svc, nil)
		result, err := getHandler(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Name: "invoice.get", Arguments: map[string]any{"id": "inv_get"}}})
		if err != nil {
			t.Fatalf("get handler error = %v", err)
		}
		assertStructuredContent(t, result, got)

		_, listHandler := invoiceListTool(svc, nil)
		result, err = listHandler(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Name: "invoice.list", Arguments: map[string]any{"customer_profile_id": "cus_1"}}})
		if err != nil {
			t.Fatalf("list handler error = %v", err)
		}
		assertStructuredSlice(t, result, []app.InvoiceSummaryDTO{summary})

		svc.listInvoicesRes = nil
		result, err = listHandler(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Name: "invoice.list", Arguments: map[string]any{"customer_profile_id": "cus_1"}}})
		if err != nil {
			t.Fatalf("empty list handler error = %v", err)
		}
		assertStructuredSlice(t, result, []app.InvoiceSummaryDTO{})

		_, pdfHandler := invoiceRenderPDFTool(svc, nil)
		result, err = pdfHandler(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Name: "invoice.render_pdf", Arguments: map[string]any{"invoice_id": "inv_pdf"}}})
		if err != nil {
			t.Fatalf("pdf handler error = %v", err)
		}
		assertStructuredContent(t, result, file)

		errSvc := &invoiceServiceStub{getInvoiceErr: errors.New("invoice not found")}
		_, errHandler := invoiceGetTool(errSvc, nil)
		result, err = errHandler(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Name: "invoice.get", Arguments: map[string]any{"id": "missing"}}})
		if err != nil {
			t.Fatalf("error handler error = %v", err)
		}
		assertTextOnlyError(t, result, "invoice not found")
	})
}

func TestTimeEntryToolsStructuredResults(t *testing.T) {
	t.Parallel()

	entry := app.TimeEntryDTO{ID: "te_1", CustomerProfileID: "cus_1", ServiceAgreementID: "sa_1", Description: "Work", Hours: 90, Billable: true, Date: "2026-04-10T00:00:00Z"}
	svc := &timeEntryServiceStub{recordRes: entry, getRes: entry, updateRes: entry, listRes: []app.TimeEntryDTO{entry}, listUnbilledRes: []app.TimeEntryDTO{entry}}
	for _, tc := range []struct {
		name    string
		tool    string
		handler func(TimeEntryServiceProvider, any) (mcp.Tool, func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error))
		args    map[string]any
	}{
		{name: "record", tool: "time_entry.record", handler: func(s TimeEntryServiceProvider, _ any) (mcp.Tool, func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
			return timeEntryRecordTool(s, nil)
		}, args: map[string]any{"customer_profile_id": "cus_1", "service_agreement_id": "sa_1", "description": "Work", "hours": float64(90), "billable": true, "date": "2026-04-10T00:00:00Z"}},
		{name: "get", tool: "time_entry.get", handler: func(s TimeEntryServiceProvider, _ any) (mcp.Tool, func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
			return timeEntryGetTool(s, nil)
		}, args: map[string]any{"id": "te_1"}},
		{name: "update", tool: "time_entry.update", handler: func(s TimeEntryServiceProvider, _ any) (mcp.Tool, func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
			return timeEntryUpdateTool(s, nil)
		}, args: map[string]any{"id": "te_1", "description": "Work", "hours": float64(90)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, handler := tc.handler(svc, nil)
			result, err := handler(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Name: tc.tool, Arguments: tc.args}})
			if err != nil {
				t.Fatalf("handler error = %v", err)
			}
			assertStructuredContent(t, result, entry)
		})
	}

	_, listHandler := timeEntryListTool(svc, nil)
	result, err := listHandler(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Name: "time_entry.list_by_customer_profile", Arguments: map[string]any{"customer_profile_id": "cus_1"}}})
	if err != nil {
		t.Fatalf("list handler error = %v", err)
	}
	assertStructuredSlice(t, result, []app.TimeEntryDTO{entry})
	svc.listRes = nil
	result, err = listHandler(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Name: "time_entry.list_by_customer_profile", Arguments: map[string]any{"customer_profile_id": "cus_1"}}})
	if err != nil {
		t.Fatalf("empty list handler error = %v", err)
	}
	assertStructuredSlice(t, result, []app.TimeEntryDTO{})

	_, unbilledHandler := timeEntryListUnbilledTool(svc, nil)
	result, err = unbilledHandler(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Name: "time_entry.list_unbilled", Arguments: map[string]any{"customer_profile_id": "cus_1"}}})
	if err != nil {
		t.Fatalf("unbilled handler error = %v", err)
	}
	assertStructuredSlice(t, result, []app.TimeEntryDTO{entry})

	_, deleteHandler := timeEntryDeleteTool(svc, nil)
	result, err = deleteHandler(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Name: "time_entry.delete", Arguments: map[string]any{"id": "te_1"}}})
	if err != nil {
		t.Fatalf("delete handler error = %v", err)
	}
	assertStructuredContent(t, result, DeleteAck{ID: "te_1", Action: "delete", Status: "ok"})

	result, err = deleteHandler(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Name: "time_entry.delete", Arguments: map[string]any{}}})
	if err != nil {
		t.Fatalf("error handler error = %v", err)
	}
	assertTextOnlyError(t, result, "id")
}

func TestCustomerProfileToolsStructuredResults(t *testing.T) {
	t.Parallel()

	profile := app.CustomerProfileDTO{ID: "cus_1", LegalEntityID: "le_1", Status: "active", DefaultCurrency: "USD"}
	svc := &customerProfileWriteServiceStub{customerProfileListServiceStub: customerProfileListServiceStub{result: app.ListResult[app.CustomerProfileDTO]{Items: []app.CustomerProfileDTO{profile}, Total: 1, Page: 1, PageSize: 10}}, createRes: profile, getRes: profile, updateRes: profile}
	for _, tc := range []struct {
		name string
		tool string
		call func() (*mcp.CallToolResult, error)
	}{
		{name: "create", tool: "customer_profile.create", call: func() (*mcp.CallToolResult, error) {
			_, h := customerProfileCreateTool(svc, nil)
			return h(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Name: "customer_profile.create", Arguments: map[string]any{"type": "company", "legal_name": "Acme", "default_currency": "USD"}}})
		}},
		{name: "get", tool: "customer_profile.get", call: func() (*mcp.CallToolResult, error) {
			_, h := customerProfileGetTool(svc, nil)
			return h(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Name: "customer_profile.get", Arguments: map[string]any{"id": "cus_1"}}})
		}},
		{name: "update", tool: "customer_profile.update", call: func() (*mcp.CallToolResult, error) {
			_, h := customerProfileUpdateTool(svc, nil)
			return h(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Name: "customer_profile.update", Arguments: map[string]any{"id": "cus_1", "default_currency": "USD"}}})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tc.call()
			if err != nil {
				t.Fatalf("handler error = %v", err)
			}
			assertStructuredContent(t, result, profile)
		})
	}
	_, listHandler := customerProfileListTool(svc, nil)
	result, err := listHandler(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Name: "customer_profile.list"}})
	if err != nil {
		t.Fatalf("list handler error = %v", err)
	}
	assertStructuredSlice(t, result, []app.CustomerProfileDTO{profile})
	svc.result.Items = nil
	result, err = listHandler(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Name: "customer_profile.list"}})
	if err != nil {
		t.Fatalf("empty list handler error = %v", err)
	}
	assertStructuredSlice(t, result, []app.CustomerProfileDTO{})
	_, deleteHandler := customerProfileDeleteTool(svc, nil)
	result, err = deleteHandler(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Name: "customer_profile.delete", Arguments: map[string]any{"id": "cus_1"}}})
	if err != nil {
		t.Fatalf("delete handler error = %v", err)
	}
	assertStructuredContent(t, result, DeleteAck{ID: "cus_1", Action: "delete", Status: "ok"})
	errSvc := &customerProfileWriteServiceStub{getErr: app.ErrCustomerProfileNotFound}
	_, getHandler := customerProfileGetTool(errSvc, nil)
	result, err = getHandler(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Name: "customer_profile.get", Arguments: map[string]any{"id": "missing"}}})
	if err != nil {
		t.Fatalf("error handler error = %v", err)
	}
	assertTextOnlyError(t, result, "not found")
}

func TestServiceAgreementToolsStructuredResults(t *testing.T) {
	t.Parallel()

	agreement := app.ServiceAgreementDTO{ID: "sa_1", CustomerProfileID: "cus_1", Name: "Support", BillingMode: "hourly", HourlyRate: 15000, Currency: "USD", Active: true}
	svc := &agreementServiceStub{createRes: agreement, getRes: agreement, updateRateRes: agreement, activateRes: agreement, deactivateRes: agreement, listRes: []app.ServiceAgreementDTO{agreement}}
	for _, tc := range []struct {
		name string
		call func() (*mcp.CallToolResult, error)
	}{
		{name: "create", call: func() (*mcp.CallToolResult, error) {
			_, h := serviceAgreementCreateTool(svc, nil)
			return h(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Name: "service_agreement.create", Arguments: map[string]any{"customer_profile_id": "cus_1", "name": "Support", "billing_mode": "hourly", "hourly_rate": float64(15000), "currency": "USD"}}})
		}},
		{name: "get", call: func() (*mcp.CallToolResult, error) {
			_, h := serviceAgreementGetTool(svc, nil)
			return h(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Name: "service_agreement.get", Arguments: map[string]any{"id": "sa_1"}}})
		}},
		{name: "update_rate", call: func() (*mcp.CallToolResult, error) {
			_, h := serviceAgreementUpdateRateTool(svc, nil)
			return h(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Name: "service_agreement.update_rate", Arguments: map[string]any{"id": "sa_1", "hourly_rate": float64(15000)}}})
		}},
		{name: "activate", call: func() (*mcp.CallToolResult, error) {
			_, h := serviceAgreementActivateTool(svc, nil)
			return h(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Name: "service_agreement.activate", Arguments: map[string]any{"id": "sa_1"}}})
		}},
		{name: "deactivate", call: func() (*mcp.CallToolResult, error) {
			_, h := serviceAgreementDeactivateTool(svc, nil)
			return h(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Name: "service_agreement.deactivate", Arguments: map[string]any{"id": "sa_1"}}})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tc.call()
			if err != nil {
				t.Fatalf("handler error = %v", err)
			}
			assertStructuredContent(t, result, agreement)
		})
	}
	_, listHandler := serviceAgreementListTool(svc, nil)
	result, err := listHandler(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Name: "service_agreement.list_by_customer_profile", Arguments: map[string]any{"customer_profile_id": "cus_1"}}})
	if err != nil {
		t.Fatalf("list handler error = %v", err)
	}
	assertStructuredSlice(t, result, []app.ServiceAgreementDTO{agreement})
	svc.listRes = nil
	result, err = listHandler(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Name: "service_agreement.list_by_customer_profile", Arguments: map[string]any{"customer_profile_id": "cus_1"}}})
	if err != nil {
		t.Fatalf("empty list handler error = %v", err)
	}
	assertStructuredSlice(t, result, []app.ServiceAgreementDTO{})
	errSvc := &agreementServiceStub{activateErr: app.ErrServiceAgreementNotFound}
	_, activateHandler := serviceAgreementActivateTool(errSvc, nil)
	result, err = activateHandler(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Name: "service_agreement.activate", Arguments: map[string]any{"id": "missing"}}})
	if err != nil {
		t.Fatalf("error handler error = %v", err)
	}
	assertTextOnlyError(t, result, "not found")
}

func TestIssuerProfileToolsStructuredResults(t *testing.T) {
	t.Parallel()

	issuer := app.IssuerProfileDTO{ID: "iss_1", LegalEntityID: "le_1", DefaultCurrency: "USD", DefaultNotes: "Thanks"}
	svc := &issuerProfileServiceStub{createRes: issuer, getRes: issuer, updateRes: issuer}
	for _, tc := range []struct {
		name string
		call func() (*mcp.CallToolResult, error)
	}{
		{name: "create", call: func() (*mcp.CallToolResult, error) {
			_, h := issuerProfileCreateTool(svc, nil)
			return h(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Name: "issuer_profile.create", Arguments: map[string]any{"type": "company", "legal_name": "Billar", "default_currency": "USD"}}})
		}},
		{name: "get", call: func() (*mcp.CallToolResult, error) {
			_, h := issuerProfileGetTool(svc, nil)
			return h(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Name: "issuer_profile.get", Arguments: map[string]any{"id": "iss_1"}}})
		}},
		{name: "update", call: func() (*mcp.CallToolResult, error) {
			_, h := issuerProfileUpdateTool(svc, nil)
			return h(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Name: "issuer_profile.update", Arguments: map[string]any{"id": "iss_1", "default_currency": "USD"}}})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tc.call()
			if err != nil {
				t.Fatalf("handler error = %v", err)
			}
			assertStructuredContent(t, result, issuer)
		})
	}
	_, deleteHandler := issuerProfileDeleteTool(svc, nil)
	result, err := deleteHandler(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Name: "issuer_profile.delete", Arguments: map[string]any{"id": "iss_1"}}})
	if err != nil {
		t.Fatalf("delete handler error = %v", err)
	}
	assertStructuredContent(t, result, DeleteAck{ID: "iss_1", Action: "delete", Status: "ok"})
	errSvc := &issuerProfileServiceStub{getErr: errors.New("issuer profile not found")}
	_, getHandler := issuerProfileGetTool(errSvc, nil)
	result, err = getHandler(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Name: "issuer_profile.get", Arguments: map[string]any{"id": "missing"}}})
	if err != nil {
		t.Fatalf("error handler error = %v", err)
	}
	assertTextOnlyError(t, result, "not found")
}
