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
	getID     string
	getRes    app.CustomerProfileDTO
	getErr    error
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
	s.getID = id
	return s.getRes, s.getErr
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

	_, handler := customerProfileListTool(service, nil)
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

			_, handler := customerProfileCreateTool(tc.service, nil)
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

			_, handler := customerProfileUpdateTool(tc.service, nil)
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

// TestCustomerProfileCreateToolHandlers_BillingAddress verifies that billing_address
// is deserialized from the JSON arguments into the correct app.AddressDTO on the command.
func TestCustomerProfileCreateToolHandlers_BillingAddress(t *testing.T) {
	t.Parallel()

	svc := &customerProfileWriteServiceStub{
		createRes: app.CustomerProfileDTO{
			ID:            "cus_addr_01",
			LegalEntityID: "le_001",
			Status:        "active",
		},
	}

	_, handler := customerProfileCreateTool(svc, nil)
	result, err := handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "customer_profile.create",
			Arguments: map[string]any{
				"type":             "company",
				"legal_name":       "Addr Corp",
				"default_currency": "USD",
				"billing_address": map[string]any{
					"street":      "5 King Rd",
					"city":        "Kingston",
					"state":       "KN",
					"postal_code": "00001",
					"country":     "JM",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("handler error = %v", err)
	}
	if result == nil || result.IsError {
		t.Fatalf("handler result = %+v, want success", result)
	}
	if svc.createArg == nil {
		t.Fatal("Create() was not called")
	}
	if svc.createArg.BillingAddress.Street != "5 King Rd" {
		t.Errorf("BillingAddress.Street = %q, want %q", svc.createArg.BillingAddress.Street, "5 King Rd")
	}
	if svc.createArg.BillingAddress.City != "Kingston" {
		t.Errorf("BillingAddress.City = %q, want %q", svc.createArg.BillingAddress.City, "Kingston")
	}
	if svc.createArg.BillingAddress.Country != "JM" {
		t.Errorf("BillingAddress.Country = %q, want %q", svc.createArg.BillingAddress.Country, "JM")
	}
}

// TestCustomerProfileUpdateToolHandlers_PatchPointerSemantics verifies absent vs.
// explicitly-cleared field distinction in the update handler.
func TestCustomerProfileUpdateToolHandlers_PatchPointerSemantics(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		arguments          map[string]any
		wantEmailNil       bool
		wantEmailValue     string
		wantAddressNil     bool
		wantAddressCountry string
	}{
		{
			name: "absent optional field is nil pointer in PatchCommand",
			arguments: map[string]any{
				"id":               "cus_123",
				"default_currency": "USD",
				// email absent
			},
			wantEmailNil:   true,
			wantAddressNil: true,
		},
		{
			name: "explicitly cleared field is non-nil pointer to empty string",
			arguments: map[string]any{
				"id":    "cus_123",
				"email": "",
			},
			wantEmailNil:   false,
			wantEmailValue: "",
			wantAddressNil: true,
		},
		{
			name: "billing_address provided binds nested struct",
			arguments: map[string]any{
				"id": "cus_123",
				"billing_address": map[string]any{
					"country": "DO",
					"city":    "Santiago",
				},
			},
			wantEmailNil:       true,
			wantAddressNil:     false,
			wantAddressCountry: "DO",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := &customerProfileWriteServiceStub{
				updateRes: app.CustomerProfileDTO{ID: "cus_123", LegalEntityID: "le_456", Status: "active"},
			}
			_, handler := customerProfileUpdateTool(svc, nil)
			result, err := handler(context.Background(), mcp.CallToolRequest{
				Params: mcp.CallToolParams{Name: "customer_profile.update", Arguments: tc.arguments},
			})
			if err != nil {
				t.Fatalf("handler error = %v", err)
			}
			if result == nil || result.IsError {
				t.Fatalf("handler result = %+v, want success", result)
			}
			if svc.updateArg == nil {
				t.Fatal("Update() was not called")
			}

			// Email pointer semantics.
			if tc.wantEmailNil && svc.updateArg.Email != nil {
				t.Errorf("Email = %v, want nil (absent field)", svc.updateArg.Email)
			}
			if !tc.wantEmailNil {
				if svc.updateArg.Email == nil {
					t.Fatal("Email = nil, want non-nil pointer (field explicitly provided)")
				}
				if *svc.updateArg.Email != tc.wantEmailValue {
					t.Errorf("*Email = %q, want %q", *svc.updateArg.Email, tc.wantEmailValue)
				}
			}

			// BillingAddress pointer semantics.
			if tc.wantAddressNil && svc.updateArg.BillingAddress != nil {
				t.Errorf("BillingAddress = %v, want nil (address absent)", svc.updateArg.BillingAddress)
			}
			if !tc.wantAddressNil {
				if svc.updateArg.BillingAddress == nil {
					t.Fatal("BillingAddress = nil, want non-nil (address was provided)")
				}
				if svc.updateArg.BillingAddress.Country != tc.wantAddressCountry {
					t.Errorf("BillingAddress.Country = %q, want %q", svc.updateArg.BillingAddress.Country, tc.wantAddressCountry)
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

			_, handler := customerProfileDeleteTool(tc.service, nil)
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
