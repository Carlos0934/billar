package mcp

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Carlos0934/billar/internal/app"
	"github.com/mark3labs/mcp-go/mcp"
)

// customerProfileListServiceStub for testing list operations
type customerProfileListServiceStub struct {
	called bool
	query  app.ListQuery
	result app.ListResult[app.CustomerProfileDTO]
	err    error
}

func (s *customerProfileListServiceStub) List(ctx context.Context, query app.ListQuery) (app.ListResult[app.CustomerProfileDTO], error) {
	_ = ctx
	s.called = true
	s.query = query
	return s.result, s.err
}

// customerProfileWriteServiceStub for testing write operations
type customerProfileWriteServiceStub struct {
	customerProfileListServiceStub
	createArg *app.CreateCustomerProfileCommand
	createRes app.CustomerProfileDTO
	createErr error
	updateID  string
	updateArg *app.PatchCustomerProfileCommand
	updateRes app.CustomerProfileDTO
	updateErr error
	deleteID  string
	deleteErr error
}

func (s *customerProfileWriteServiceStub) Create(ctx context.Context, cmd app.CreateCustomerProfileCommand) (app.CustomerProfileDTO, error) {
	_ = ctx
	s.createArg = &cmd
	return s.createRes, s.createErr
}

func (s *customerProfileWriteServiceStub) Get(ctx context.Context, id string) (app.CustomerProfileDTO, error) {
	_ = ctx
	return app.CustomerProfileDTO{}, errors.New("not implemented in test stub")
}

func (s *customerProfileWriteServiceStub) Update(ctx context.Context, id string, cmd app.PatchCustomerProfileCommand) (app.CustomerProfileDTO, error) {
	_ = ctx
	s.updateID = id
	s.updateArg = &cmd
	return s.updateRes, s.updateErr
}

func (s *customerProfileWriteServiceStub) Delete(ctx context.Context, id string) error {
	_ = ctx
	s.deleteID = id
	return s.deleteErr
}

func TestCustomerProfileListToolHandlers(t *testing.T) {
	t.Parallel()

	service := &customerProfileListServiceStub{
		result: app.ListResult[app.CustomerProfileDTO]{
			Items: []app.CustomerProfileDTO{{
				ID:              "cus_123",
				LegalEntityID:   "le_456",
				Status:          "active",
				DefaultCurrency: "USD",
				CreatedAt:       "2026-04-03T10:00:00Z",
				UpdatedAt:       "2026-04-03T10:05:00Z",
			}},
			Total:    1,
			Page:     1,
			PageSize: 10,
		},
	}
	guard := NewIngressGuard([]string{"127.0.0.1"})

	_, handler := customerProfileListTool(service, guard, nil)
	result, err := handler(context.Background(), mcp.CallToolRequest{Header: headerWithValues(map[string]string{
		"X-Forwarded-For": "127.0.0.1",
	}), Params: mcp.CallToolParams{Name: "customer_profile.list"}})
	if err != nil {
		t.Fatalf("handler error = %v", err)
	}
	if result == nil || result.IsError {
		t.Fatalf("handler result = %+v, want success result", result)
	}
	if !service.called {
		t.Fatal("List() was not called")
	}
}

func TestCustomerProfileCreateToolHandlers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		service       *customerProfileWriteServiceStub
		arguments     map[string]any
		wantErr       bool
		wantErrSubstr string
		wantCreateArg *app.CreateCustomerProfileCommand
		wantResult    string
	}{
		{
			name: "creates customer profile successfully with inline legal entity",
			service: &customerProfileWriteServiceStub{
				createRes: app.CustomerProfileDTO{
					ID:              "cus_123",
					LegalEntityID:   "le_456",
					Status:          "active",
					DefaultCurrency: "USD",
				},
			},
			arguments: map[string]any{
				"type":             "company",
				"legal_name":       "Acme SRL",
				"default_currency": "USD",
			},
			wantCreateArg: &app.CreateCustomerProfileCommand{
				LegalEntityType: "company",
				LegalName:       "Acme SRL",
				DefaultCurrency: "USD",
			},
			wantResult: "Customer profile created: cus_123\nLegal entity ID: le_456\nStatus: active\nDefault currency: USD\n",
		},
		{
			name: "returns error when service rejects command",
			service: &customerProfileWriteServiceStub{
				createErr: errors.New("legal name is required"),
			},
			arguments: map[string]any{
				"type":             "company",
				"default_currency": "USD",
			},
			wantErr:       true,
			wantErrSubstr: "legal name is required",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, handler := customerProfileCreateTool(tc.service, NewIngressGuard(nil), nil)
			result, err := handler(context.Background(), mcp.CallToolRequest{
				Params: mcp.CallToolParams{Name: "customer_profile.create", Arguments: tc.arguments},
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
				if tc.service.createArg.LegalEntityType != tc.wantCreateArg.LegalEntityType {
					t.Errorf("Create() type = %q, want %q", tc.service.createArg.LegalEntityType, tc.wantCreateArg.LegalEntityType)
				}
				if tc.service.createArg.LegalName != tc.wantCreateArg.LegalName {
					t.Errorf("Create() legal_name = %q, want %q", tc.service.createArg.LegalName, tc.wantCreateArg.LegalName)
				}
				if tc.service.createArg.DefaultCurrency != tc.wantCreateArg.DefaultCurrency {
					t.Errorf("Create() default_currency = %q, want %q", tc.service.createArg.DefaultCurrency, tc.wantCreateArg.DefaultCurrency)
				}
			}

			if got := mcp.GetTextFromContent(result.Content[0]); got != tc.wantResult {
				t.Errorf("handler text = %q, want %q", got, tc.wantResult)
			}
		})
	}
}

func TestCustomerProfileUpdateToolHandlers(t *testing.T) {
	t.Parallel()

	currency := "EUR"
	legalName := "Acme Corp"

	tests := []struct {
		name          string
		service       *customerProfileWriteServiceStub
		arguments     map[string]any
		wantErr       bool
		wantErrSubstr string
		wantUpdateID  string
		wantUpdateArg *app.PatchCustomerProfileCommand
		wantResult    string
	}{
		{
			name: "updates profile fields without LE cascade",
			service: &customerProfileWriteServiceStub{
				updateRes: app.CustomerProfileDTO{
					ID:              "cus_123",
					LegalEntityID:   "le_456",
					Status:          "active",
					DefaultCurrency: "EUR",
				},
			},
			arguments: map[string]any{
				"id":               "cus_123",
				"default_currency": "EUR",
			},
			wantUpdateID: "cus_123",
			wantUpdateArg: &app.PatchCustomerProfileCommand{
				DefaultCurrency: &currency,
			},
			wantResult: "Customer profile updated: cus_123\nLegal entity ID: le_456\nStatus: active\nDefault currency: EUR\n",
		},
		{
			name: "passes legal_name LE field in patch command",
			service: &customerProfileWriteServiceStub{
				updateRes: app.CustomerProfileDTO{
					ID:            "cus_123",
					LegalEntityID: "le_456",
					Status:        "active",
				},
			},
			arguments: map[string]any{
				"id":         "cus_123",
				"legal_name": "Acme Corp",
			},
			wantUpdateID: "cus_123",
			wantUpdateArg: &app.PatchCustomerProfileCommand{
				LegalName: &legalName,
			},
		},
		{
			name: "returns service error",
			service: &customerProfileWriteServiceStub{
				updateErr: app.ErrCustomerProfileNotFound,
			},
			arguments:     map[string]any{"id": "cus_nonexistent"},
			wantErr:       true,
			wantErrSubstr: "not found",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, handler := customerProfileUpdateTool(tc.service, NewIngressGuard(nil), nil)
			result, err := handler(context.Background(), mcp.CallToolRequest{
				Params: mcp.CallToolParams{Name: "customer_profile.update", Arguments: tc.arguments},
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

			if tc.wantUpdateArg != nil {
				if tc.service.updateArg == nil {
					t.Fatal("Update() was not called")
				}
				if tc.wantUpdateArg.DefaultCurrency != nil {
					if tc.service.updateArg.DefaultCurrency == nil || *tc.service.updateArg.DefaultCurrency != *tc.wantUpdateArg.DefaultCurrency {
						t.Errorf("Update() DefaultCurrency = %v, want %v", tc.service.updateArg.DefaultCurrency, tc.wantUpdateArg.DefaultCurrency)
					}
				}
				if tc.wantUpdateArg.LegalName != nil {
					if tc.service.updateArg.LegalName == nil || *tc.service.updateArg.LegalName != *tc.wantUpdateArg.LegalName {
						t.Errorf("Update() LegalName = %v, want %v", tc.service.updateArg.LegalName, tc.wantUpdateArg.LegalName)
					}
				}
			}

			if tc.wantResult != "" {
				if got := mcp.GetTextFromContent(result.Content[0]); got != tc.wantResult {
					t.Errorf("handler text = %q, want %q", got, tc.wantResult)
				}
			}
		})
	}
}

func TestCustomerProfileDeleteToolHandlers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		service       *customerProfileWriteServiceStub
		arguments     map[string]any
		wantErr       bool
		wantErrSubstr string
		wantDeleteID  string
	}{
		{
			name:         "deletes customer profile successfully",
			service:      &customerProfileWriteServiceStub{},
			arguments:    map[string]any{"id": "cus_123"},
			wantDeleteID: "cus_123",
		},
		{
			name:          "returns not-found error",
			service:       &customerProfileWriteServiceStub{deleteErr: app.ErrCustomerProfileNotFound},
			arguments:     map[string]any{"id": "cus_nonexistent"},
			wantErr:       true,
			wantErrSubstr: "not found",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, handler := customerProfileDeleteTool(tc.service, NewIngressGuard(nil), nil)
			result, err := handler(context.Background(), mcp.CallToolRequest{
				Params: mcp.CallToolParams{Name: "customer_profile.delete", Arguments: tc.arguments},
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
		})
	}
}
