package mcp

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Carlos0934/billar/internal/app"
	"github.com/mark3labs/mcp-go/mcp"
)

// legalEntityListServiceStub for testing list operations
type legalEntityListServiceStub struct {
	called bool
	query  app.ListQuery
	result app.ListResult[app.LegalEntityDTO]
	err    error
}

func (s *legalEntityListServiceStub) List(ctx context.Context, query app.ListQuery) (app.ListResult[app.LegalEntityDTO], error) {
	_ = ctx
	s.called = true
	s.query = query
	return s.result, s.err
}

// legalEntityWriteServiceStub for testing write operations
type legalEntityWriteServiceStub struct {
	legalEntityListServiceStub
	createArg *app.CreateLegalEntityCommand
	createRes app.LegalEntityDTO
	createErr error
	updateID  string
	updateArg *app.PatchLegalEntityCommand
	updateRes app.LegalEntityDTO
	updateErr error
	deleteID  string
	deleteErr error
}

func (s *legalEntityWriteServiceStub) Create(ctx context.Context, cmd app.CreateLegalEntityCommand) (app.LegalEntityDTO, error) {
	_ = ctx
	s.createArg = &cmd
	return s.createRes, s.createErr
}

func (s *legalEntityWriteServiceStub) Get(ctx context.Context, id string) (app.LegalEntityDTO, error) {
	_ = ctx
	return app.LegalEntityDTO{}, errors.New("not implemented in test stub")
}

func (s *legalEntityWriteServiceStub) Update(ctx context.Context, id string, cmd app.PatchLegalEntityCommand) (app.LegalEntityDTO, error) {
	_ = ctx
	s.updateID = id
	s.updateArg = &cmd
	return s.updateRes, s.updateErr
}

func (s *legalEntityWriteServiceStub) Delete(ctx context.Context, id string) error {
	_ = ctx
	s.deleteID = id
	return s.deleteErr
}

func TestLegalEntityListToolHandlers(t *testing.T) {
	t.Parallel()

	service := &legalEntityListServiceStub{
		result: app.ListResult[app.LegalEntityDTO]{
			Items: []app.LegalEntityDTO{{
				ID:        "le_123",
				Type:      "company",
				LegalName: "Acme Corporation SRL",
				TradeName: "Acme",
				TaxID:     "123-456-789",
				Email:     "billing@acme.example",
				Phone:     "+1-555-1234",
				Website:   "https://acme.example",
				CreatedAt: "2026-04-03T10:00:00Z",
				UpdatedAt: "2026-04-03T10:05:00Z",
			}},
			Total:    1,
			Page:     2,
			PageSize: 1,
		},
	}
	guard := NewIngressGuard([]string{"127.0.0.1"})

	_, handler := legalEntityListTool(service, guard, nil)
	result, err := handler(context.Background(), mcp.CallToolRequest{Header: headerWithValues(map[string]string{
		"X-Forwarded-For": "127.0.0.1",
	}), Params: mcp.CallToolParams{Name: "legal_entity.list", Arguments: map[string]any{
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
	want := "Billar Legal Entities\n───────────────\nPage: 2\nPage size: 1\nTotal: 1\n\n1. Acme Corporation SRL\n   Trade name: Acme\n   Type: company\n   Tax ID: 123-456-789\n   Email: billing@acme.example\n   Phone: +1-555-1234\n   Website: https://acme.example\n   Created at: 2026-04-03T10:00:00Z\n   Updated at: 2026-04-03T10:05:00Z\n"
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

func TestLegalEntityListToolRejectsIngress(t *testing.T) {
	t.Parallel()

	service := &legalEntityListServiceStub{}
	guard := NewIngressGuard([]string{"127.0.0.1"})

	_, handler := legalEntityListTool(service, guard, nil)
	result, err := handler(context.Background(), mcp.CallToolRequest{Header: headerWithValues(map[string]string{
		"X-Forwarded-For": "192.0.2.10",
	}), Params: mcp.CallToolParams{Name: "legal_entity.list"}})
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

func TestLegalEntityCreateToolHandlers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		service       *legalEntityWriteServiceStub
		arguments     map[string]any
		wantErr       bool
		wantErrSubstr string
		wantCreateArg *app.CreateLegalEntityCommand
		wantResult    string
	}{
		{
			name: "creates legal entity with required fields",
			service: &legalEntityWriteServiceStub{
				createRes: app.LegalEntityDTO{
					ID:        "le_123",
					Type:      "company",
					LegalName: "Acme Corporation SRL",
				},
			},
			arguments: map[string]any{
				"type":       "company",
				"legal_name": "Acme Corporation SRL",
			},
			wantCreateArg: &app.CreateLegalEntityCommand{
				Type:      "company",
				LegalName: "Acme Corporation SRL",
			},
			wantResult: "Legal entity created: le_123\nType: company\nLegal name: Acme Corporation SRL\n",
		},
		{
			name: "creates legal entity with all fields",
			service: &legalEntityWriteServiceStub{
				createRes: app.LegalEntityDTO{
					ID:        "le_456",
					Type:      "individual",
					LegalName: "John Doe",
					Email:     "john@example.com",
					Phone:     "+1-555-9999",
				},
			},
			arguments: map[string]any{
				"type":       "individual",
				"legal_name": "John Doe",
				"trade_name": "JD Enterprises",
				"tax_id":     "123-456-789",
				"email":      "john@example.com",
				"phone":      "+1-555-9999",
				"website":    "https://johndoe.example",
			},
			wantCreateArg: &app.CreateLegalEntityCommand{
				Type:      "individual",
				LegalName: "John Doe",
				TradeName: "JD Enterprises",
				TaxID:     "123-456-789",
				Email:     "john@example.com",
				Phone:     "+1-555-9999",
				Website:   "https://johndoe.example",
			},
			wantResult: "Legal entity created: le_456\nType: individual\nLegal name: John Doe\nEmail: john@example.com\nPhone: +1-555-9999\n",
		},
		{
			name: "returns error for missing required field",
			service: &legalEntityWriteServiceStub{
				createErr: errors.New("legal name is required"),
			},
			arguments: map[string]any{
				"type": "company",
			},
			wantErr:       true,
			wantErrSubstr: "legal name is required",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, handler := legalEntityCreateTool(tc.service, NewIngressGuard(nil), nil)
			result, err := handler(context.Background(), mcp.CallToolRequest{
				Params: mcp.CallToolParams{Name: "legal_entity.create", Arguments: tc.arguments},
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

func TestLegalEntityUpdateToolHandlers(t *testing.T) {
	t.Parallel()

	email := "updated@example.com"
	tests := []struct {
		name          string
		service       *legalEntityWriteServiceStub
		arguments     map[string]any
		wantErr       bool
		wantErrSubstr string
		wantUpdateID  string
		wantUpdateArg *app.PatchLegalEntityCommand
		wantResult    string
	}{
		{
			name: "applies partial patch successfully",
			service: &legalEntityWriteServiceStub{
				updateRes: app.LegalEntityDTO{
					ID:        "le_123",
					Type:      "company",
					LegalName: "Acme SRL",
					Email:     "updated@example.com",
				},
			},
			arguments: map[string]any{
				"id":    "le_123",
				"email": "updated@example.com",
			},
			wantUpdateID: "le_123",
			wantUpdateArg: &app.PatchLegalEntityCommand{
				Email: &email,
			},
			wantResult: "Legal entity updated: le_123\nType: company\nLegal name: Acme SRL\nEmail: updated@example.com\n",
		},
		{
			name:          "returns not-found error",
			service:       &legalEntityWriteServiceStub{updateErr: app.ErrLegalEntityNotFound},
			arguments:     map[string]any{"id": "le_nonexistent"},
			wantErr:       true,
			wantErrSubstr: "not found",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, handler := legalEntityUpdateTool(tc.service, NewIngressGuard(nil), nil)
			result, err := handler(context.Background(), mcp.CallToolRequest{
				Params: mcp.CallToolParams{Name: "legal_entity.update", Arguments: tc.arguments},
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

			if got := mcp.GetTextFromContent(result.Content[0]); got != tc.wantResult {
				t.Errorf("handler text = %q, want %q", got, tc.wantResult)
			}
		})
	}
}

func TestLegalEntityDeleteToolHandlers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		service       *legalEntityWriteServiceStub
		arguments     map[string]any
		wantErr       bool
		wantErrSubstr string
		wantDeleteID  string
	}{
		{
			name:         "deletes legal entity successfully",
			service:      &legalEntityWriteServiceStub{},
			arguments:    map[string]any{"id": "le_123"},
			wantDeleteID: "le_123",
		},
		{
			name:          "returns not-found error",
			service:       &legalEntityWriteServiceStub{deleteErr: app.ErrLegalEntityNotFound},
			arguments:     map[string]any{"id": "le_nonexistent"},
			wantErr:       true,
			wantErrSubstr: "not found",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, handler := legalEntityDeleteTool(tc.service, NewIngressGuard(nil), nil)
			result, err := handler(context.Background(), mcp.CallToolRequest{
				Params: mcp.CallToolParams{Name: "legal_entity.delete", Arguments: tc.arguments},
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
