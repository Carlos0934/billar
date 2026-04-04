package mcp

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Carlos0934/billar/internal/app"
	"github.com/mark3labs/mcp-go/mcp"
)

type contextAwareCustomerProvider struct{}

func (contextAwareCustomerProvider) List(ctx context.Context, _ app.ListQuery) (app.ListResult[app.CustomerDTO], error) {
	identity, ok, err := (app.ContextIdentitySource{}).CurrentIdentity(ctx)
	if err != nil {
		return app.ListResult[app.CustomerDTO]{}, err
	}
	if !ok || identity.Email != "user@example.com" {
		return app.ListResult[app.CustomerDTO]{}, app.ErrCustomerListAccessDenied
	}
	return app.ListResult[app.CustomerDTO]{}, nil
}

type customerListServiceStub struct {
	called bool
	query  app.ListQuery
	result app.ListResult[app.CustomerDTO]
	err    error
}

func (s *customerListServiceStub) List(ctx context.Context, query app.ListQuery) (app.ListResult[app.CustomerDTO], error) {
	_ = ctx
	s.called = true
	s.query = query
	return s.result, s.err
}

func TestCustomerListToolHandlers(t *testing.T) {
	t.Parallel()

	service := &customerListServiceStub{
		result: app.ListResult[app.CustomerDTO]{
			Items: []app.CustomerDTO{{
				ID:              "cus_123",
				Type:            "company",
				LegalName:       "Acme SRL",
				TradeName:       "Acme",
				Email:           "billing@acme.example",
				Status:          "active",
				DefaultCurrency: "USD",
				CreatedAt:       "2026-04-03T10:00:00Z",
				UpdatedAt:       "2026-04-03T10:05:00Z",
			}},
			Total:    1,
			Page:     2,
			PageSize: 1,
		},
	}
	guard := NewIngressGuard([]string{"127.0.0.1"})

	_, handler := customerListTool(service, guard, nil)
	result, err := handler(context.Background(), mcp.CallToolRequest{Header: headerWithValues(map[string]string{
		"X-Forwarded-For": "127.0.0.1",
	}), Params: mcp.CallToolParams{Name: "customer.list", Arguments: map[string]any{
		"search":    "  Acme  ",
		"sort":      "created_at:desc",
		"page":      2,
		"page_size": 1,
	}}})
	if err != nil {
		t.Fatalf("handler error = %v", err)
	}
	if result == nil || result.IsError {
		t.Fatalf("handler result = %+v, want success result", result)
	}
	if len(result.Content) != 1 {
		t.Fatalf("handler result content = %d, want 1", len(result.Content))
	}
	want := "Billar Customers\n───────────────\nPage: 2\nPage size: 1\nTotal: 1\n\n1. Acme SRL\n   Trade name: Acme\n   Type: company\n   Status: active\n   Email: billing@acme.example\n   Default currency: USD\n   Created at: 2026-04-03T10:00:00Z\n   Updated at: 2026-04-03T10:05:00Z\n"
	if got := mcp.GetTextFromContent(result.Content[0]); got != want {
		t.Fatalf("handler text = %q, want %q", got, want)
	}
	if !service.called {
		t.Fatal("List() was not called")
	}
	if service.query != (app.ListQuery{Search: "Acme", SortField: "created_at", SortDir: "desc", Page: 2, PageSize: 1}) {
		t.Fatalf("List() query = %+v", service.query)
	}
}

func TestCustomerListToolHandlersRejectIngress(t *testing.T) {
	t.Parallel()

	service := &customerListServiceStub{}
	guard := NewIngressGuard([]string{"127.0.0.1"})

	_, handler := customerListTool(service, guard, nil)
	result, err := handler(context.Background(), mcp.CallToolRequest{Header: headerWithValues(map[string]string{
		"X-Forwarded-For": "192.0.2.10",
	}), Params: mcp.CallToolParams{Name: "customer.list"}})
	if err != nil {
		t.Fatalf("handler error = %v", err)
	}
	if result == nil || !result.IsError {
		t.Fatalf("handler result = %+v, want error result", result)
	}
	if service.called {
		t.Fatal("List() was called for rejected request")
	}
}

func TestCustomerListToolUsesContextAuthenticatedIdentity(t *testing.T) {
	t.Parallel()

	service := contextAwareCustomerProvider{}
	_, handler := customerListTool(service, NewIngressGuard(nil), nil)
	result, err := handler(app.WithAuthenticatedIdentity(context.Background(), app.AuthenticatedIdentity{Email: "user@example.com", EmailVerified: true}), mcp.CallToolRequest{Params: mcp.CallToolParams{Name: "customer.list"}})
	if err != nil {
		t.Fatalf("handler error = %v", err)
	}
	if result == nil || result.IsError {
		t.Fatalf("handler result = %+v, want success result", result)
	}
}

func TestCustomerListToolHandlersRejectBadSort(t *testing.T) {
	t.Parallel()

	service := &customerListServiceStub{}
	_, handler := customerListTool(service, NewIngressGuard(nil), nil)
	result, err := handler(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Name: "customer.list", Arguments: map[string]any{
		"sort": "foo:bar",
	}}})
	if err != nil {
		t.Fatalf("handler error = %v", err)
	}
	if result == nil || result.IsError {
		t.Fatalf("handler result = %+v, want success result", result)
	}
	if !strings.Contains(mcp.GetTextFromContent(result.Content[0]), "Billar Customers") {
		t.Fatalf("handler text = %q", mcp.GetTextFromContent(result.Content[0]))
	}
}

// CustomerWriteServiceStub for testing write operations
type customerWriteServiceStub struct {
	customerListServiceStub
	createArg *app.CreateCustomerCommand
	createRes app.CustomerDTO
	createErr error
	updateID  string
	updateArg *app.PatchCustomerCommand
	updateRes app.CustomerDTO
	updateErr error
	deleteID  string
	deleteErr error
}

func (s *customerWriteServiceStub) Create(ctx context.Context, cmd app.CreateCustomerCommand) (app.CustomerDTO, error) {
	_ = ctx
	s.createArg = &cmd
	return s.createRes, s.createErr
}

func (s *customerWriteServiceStub) Update(ctx context.Context, id string, cmd app.PatchCustomerCommand) (app.CustomerDTO, error) {
	_ = ctx
	s.updateID = id
	s.updateArg = &cmd
	return s.updateRes, s.updateErr
}

func (s *customerWriteServiceStub) Delete(ctx context.Context, id string) error {
	_ = ctx
	s.deleteID = id
	return s.deleteErr
}

func TestCustomerCreateToolHandlers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		service       *customerWriteServiceStub
		arguments     map[string]any
		wantErr       bool
		wantErrSubstr string
		wantCreateArg *app.CreateCustomerCommand
		wantResult    string
	}{
		{
			name: "creates customer with valid JSON",
			service: &customerWriteServiceStub{
				createRes: app.CustomerDTO{
					ID:        "cus_123",
					Type:      "company",
					LegalName: "Acme SRL",
					Email:     "billing@acme.example",
					Status:    "active",
				},
			},
			arguments: map[string]any{
				"json": `{"type":"company","legal_name":"Acme SRL","email":"billing@acme.example"}`,
			},
			wantCreateArg: &app.CreateCustomerCommand{
				Type:      "company",
				LegalName: "Acme SRL",
				Email:     "billing@acme.example",
			},
			wantResult: "Customer created: cus_123\nType: company\nLegal name: Acme SRL\nEmail: billing@acme.example\nStatus: active\n",
		},
		{
			name: "returns error for missing required field",
			service: &customerWriteServiceStub{
				createErr: errors.New("legal name is required"),
			},
			arguments: map[string]any{
				"json": `{"type":"company"}`,
			},
			wantErr:       true,
			wantErrSubstr: "legal name is required",
		},
		{
			name: "returns error for unauthenticated request",
			service: &customerWriteServiceStub{
				createErr: app.ErrCustomerCreateAccessDenied,
			},
			arguments: map[string]any{
				"json": `{"type":"company","legal_name":"Test"}`,
			},
			wantErr:       true,
			wantErrSubstr: "authenticated",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, handler := customerCreateTool(tc.service, NewIngressGuard(nil), nil)
			result, err := handler(context.Background(), mcp.CallToolRequest{
				Params: mcp.CallToolParams{Name: "customer.create", Arguments: tc.arguments},
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

			if tc.wantCreateArg != nil {
				if tc.service.createArg == nil {
					t.Fatal("Create() was not called")
				}
				if tc.service.createArg.Type != tc.wantCreateArg.Type {
					t.Errorf("Create() type = %q, want %q", tc.service.createArg.Type, tc.wantCreateArg.Type)
				}
				if tc.service.createArg.LegalName != tc.wantCreateArg.LegalName {
					t.Errorf("Create() legal_name = %q, want %q", tc.service.createArg.LegalName, tc.wantCreateArg.LegalName)
				}
			}

			if got := mcp.GetTextFromContent(result.Content[0]); got != tc.wantResult {
				t.Errorf("handler text = %q, want %q", got, tc.wantResult)
			}
		})
	}
}

func TestCustomerCreateToolRejectsMalformedJSON(t *testing.T) {
	t.Parallel()

	service := &customerWriteServiceStub{}
	_, handler := customerCreateTool(service, NewIngressGuard(nil), nil)
	result, err := handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{Name: "customer.create", Arguments: map[string]any{
			"json": `{invalid json}`,
		}},
	})
	if err != nil {
		t.Fatalf("handler error = %v", err)
	}
	if result == nil || !result.IsError {
		t.Fatalf("handler result = %+v, want error result", result)
	}
	if !strings.Contains(mcp.GetTextFromContent(result.Content[0]), "parse") {
		t.Errorf("handler error = %q, want substring 'parse'", mcp.GetTextFromContent(result.Content[0]))
	}
	if service.createArg != nil {
		t.Fatal("Create() was called for malformed JSON")
	}
}

func TestCustomerUpdateToolHandlers(t *testing.T) {
	t.Parallel()

	email := "updated@example.com"
	tests := []struct {
		name          string
		service       *customerWriteServiceStub
		arguments     map[string]any
		wantErr       bool
		wantErrSubstr string
		wantUpdateID  string
		wantUpdateArg *app.PatchCustomerCommand
		wantResult    string
	}{
		{
			name: "applies partial patch successfully",
			service: &customerWriteServiceStub{
				updateRes: app.CustomerDTO{
					ID:        "cus_123",
					Type:      "company",
					LegalName: "Acme SRL",
					Email:     "updated@example.com",
					Status:    "active",
				},
			},
			arguments: map[string]any{
				"id":   "cus_123",
				"json": `{"email":"updated@example.com"}`,
			},
			wantUpdateID: "cus_123",
			wantUpdateArg: &app.PatchCustomerCommand{
				Email: &email,
			},
			wantResult: "Customer updated: cus_123\nType: company\nLegal name: Acme SRL\nEmail: updated@example.com\nStatus: active\n",
		},
		{
			name:          "returns not-found error",
			service:       &customerWriteServiceStub{updateErr: app.ErrCustomerNotFound},
			arguments:     map[string]any{"id": "cus_nonexistent", "json": `{}`},
			wantErr:       true,
			wantErrSubstr: "not found",
		},
		{
			name:          "returns error for unauthenticated request",
			service:       &customerWriteServiceStub{updateErr: app.ErrCustomerUpdateAccessDenied},
			arguments:     map[string]any{"id": "cus_123", "json": `{}`},
			wantErr:       true,
			wantErrSubstr: "authenticated",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, handler := customerUpdateTool(tc.service, NewIngressGuard(nil), nil)
			result, err := handler(context.Background(), mcp.CallToolRequest{
				Params: mcp.CallToolParams{Name: "customer.update", Arguments: tc.arguments},
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

			if tc.wantUpdateID != "" && tc.service.updateID != tc.wantUpdateID {
				t.Errorf("Update() id = %q, want %q", tc.service.updateID, tc.wantUpdateID)
			}

			if tc.wantUpdateArg != nil && tc.service.updateArg != nil {
				if tc.wantUpdateArg.Email != nil && *tc.service.updateArg.Email != *tc.wantUpdateArg.Email {
					t.Errorf("Update() email = %q, want %q", *tc.service.updateArg.Email, *tc.wantUpdateArg.Email)
				}
			}

			if got := mcp.GetTextFromContent(result.Content[0]); got != tc.wantResult {
				t.Errorf("handler text = %q, want %q", got, tc.wantResult)
			}
		})
	}
}

func TestCustomerDeleteToolHandlers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		service       *customerWriteServiceStub
		arguments     map[string]any
		wantErr       bool
		wantErrSubstr string
		wantDeleteID  string
	}{
		{
			name:         "deletes customer successfully",
			service:      &customerWriteServiceStub{},
			arguments:    map[string]any{"id": "cus_123"},
			wantDeleteID: "cus_123",
		},
		{
			name:          "returns not-found error",
			service:       &customerWriteServiceStub{deleteErr: app.ErrCustomerNotFound},
			arguments:     map[string]any{"id": "cus_nonexistent"},
			wantErr:       true,
			wantErrSubstr: "not found",
		},
		{
			name:          "returns error for unauthenticated request",
			service:       &customerWriteServiceStub{deleteErr: app.ErrCustomerDeleteAccessDenied},
			arguments:     map[string]any{"id": "cus_123"},
			wantErr:       true,
			wantErrSubstr: "authenticated",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, handler := customerDeleteTool(tc.service, NewIngressGuard(nil), nil)
			result, err := handler(context.Background(), mcp.CallToolRequest{
				Params: mcp.CallToolParams{Name: "customer.delete", Arguments: tc.arguments},
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

			if tc.wantDeleteID != "" && tc.service.deleteID != tc.wantDeleteID {
				t.Errorf("Delete() id = %q, want %q", tc.service.deleteID, tc.wantDeleteID)
			}

			if !tc.wantErr && tc.wantDeleteID != "" {
				if got := mcp.GetTextFromContent(result.Content[0]); !strings.Contains(got, "deleted") {
					t.Errorf("handler text = %q, want substring 'deleted'", got)
				}
			}
		})
	}
}
