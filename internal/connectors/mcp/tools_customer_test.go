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
			name: "creates customer with explicit arguments",
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
				"type":       "company",
				"legal_name": "Acme SRL",
				"email":      "billing@acme.example",
			},
			wantCreateArg: &app.CreateCustomerCommand{
				Type:      "company",
				LegalName: "Acme SRL",
				Email:     "billing@acme.example",
			},
			wantResult: "Customer created: cus_123\nType: company\nLegal name: Acme SRL\nEmail: billing@acme.example\nStatus: active\n",
		},
		{
			name: "creates customer with all fields",
			service: &customerWriteServiceStub{
				createRes: app.CustomerDTO{
					ID:        "cus_456",
					Type:      "individual",
					LegalName: "John Doe",
					Email:     "john@example.com",
					Status:    "active",
				},
			},
			arguments: map[string]any{
				"type":             "individual",
				"legal_name":       "John Doe",
				"email":            "john@example.com",
				"phone":            "+1-555-1234",
				"website":          "https://johndoe.example",
				"tax_id":           "123-456-789",
				"default_currency": "USD",
				"notes":            "Important client",
			},
			wantCreateArg: &app.CreateCustomerCommand{
				Type:            "individual",
				LegalName:       "John Doe",
				Email:           "john@example.com",
				Phone:           "+1-555-1234",
				Website:         "https://johndoe.example",
				TaxID:           "123-456-789",
				DefaultCurrency: "USD",
				Notes:           "Important client",
			},
			wantResult: "Customer created: cus_456\nType: individual\nLegal name: John Doe\nEmail: john@example.com\nStatus: active\n",
		},
		{
			name: "creates customer with billing address",
			service: &customerWriteServiceStub{
				createRes: app.CustomerDTO{
					ID:        "cus_789",
					Type:      "company",
					LegalName: "Acme Corp",
					Status:    "active",
				},
			},
			arguments: map[string]any{
				"type":       "company",
				"legal_name": "Acme Corp",
				"billing_address": map[string]any{
					"street":      "123 Main St",
					"city":        "Santo Domingo",
					"state":       "Distrito Nacional",
					"postal_code": "10101",
					"country":     "DO",
				},
			},
			wantCreateArg: &app.CreateCustomerCommand{
				Type:      "company",
				LegalName: "Acme Corp",
				BillingAddress: app.AddressDTO{
					Street:     "123 Main St",
					City:       "Santo Domingo",
					State:      "Distrito Nacional",
					PostalCode: "10101",
					Country:    "DO",
				},
			},
			wantResult: "Customer created: cus_789\nType: company\nLegal name: Acme Corp\nStatus: active\n",
		},
		{
			name: "returns error for missing required field",
			service: &customerWriteServiceStub{
				createErr: errors.New("legal name is required"),
			},
			arguments: map[string]any{
				"type": "company",
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
				"type":       "company",
				"legal_name": "Test",
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
				if tc.wantCreateArg.BillingAddress != (app.AddressDTO{}) {
					if tc.service.createArg.BillingAddress != tc.wantCreateArg.BillingAddress {
						t.Errorf("Create() billing_address = %+v, want %+v", tc.service.createArg.BillingAddress, tc.wantCreateArg.BillingAddress)
					}
				}
			}

			if got := mcp.GetTextFromContent(result.Content[0]); got != tc.wantResult {
				t.Errorf("handler text = %q, want %q", got, tc.wantResult)
			}
		})
	}
}

func TestCustomerCreateToolPreservesWhitespace(t *testing.T) {
	t.Parallel()

	service := &customerWriteServiceStub{
		createRes: app.CustomerDTO{
			ID:        "cus_123",
			Type:      "company",
			LegalName: "Acme SRL",
			Status:    "active",
		},
	}
	_, handler := customerCreateTool(service, NewIngressGuard(nil), nil)
	_, err := handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{Name: "customer.create", Arguments: map[string]any{
			"type":       "  company  ",
			"legal_name": "  Acme SRL  ",
		}},
	})
	if err != nil {
		t.Fatalf("handler error = %v", err)
	}
	if service.createArg == nil {
		t.Fatal("Create() was not called")
	}
	// Verify whitespace is trimmed
	if service.createArg.Type != "company" {
		t.Errorf("Create() type = %q, want trimmed %q", service.createArg.Type, "company")
	}
	if service.createArg.LegalName != "Acme SRL" {
		t.Errorf("Create() legal_name = %q, want trimmed %q", service.createArg.LegalName, "Acme SRL")
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
				"id":    "cus_123",
				"email": "updated@example.com",
			},
			wantUpdateID: "cus_123",
			wantUpdateArg: &app.PatchCustomerCommand{
				Email: &email,
			},
			wantResult: "Customer updated: cus_123\nType: company\nLegal name: Acme SRL\nEmail: updated@example.com\nStatus: active\n",
		},
		{
			name: "patches multiple fields",
			service: &customerWriteServiceStub{
				updateRes: app.CustomerDTO{
					ID:        "cus_456",
					Type:      "individual",
					LegalName: "John Doe",
					Email:     "new@example.com",
					Phone:     "+1-555-9999",
					Status:    "active",
				},
			},
			arguments: map[string]any{
				"id":    "cus_456",
				"type":  "individual",
				"email": "new@example.com",
				"phone": "+1-555-9999",
				"notes": "Updated notes",
			},
			wantUpdateID: "cus_456",
			wantUpdateArg: &app.PatchCustomerCommand{
				Type:  ptrToStr("individual"),
				Email: ptrToStr("new@example.com"),
				Phone: ptrToStr("+1-555-9999"),
				Notes: ptrToStr("Updated notes"),
			},
			wantResult: "Customer updated: cus_456\nType: individual\nLegal name: John Doe\nEmail: new@example.com\nStatus: active\n",
		},
		{
			name: "patches billing address",
			service: &customerWriteServiceStub{
				updateRes: app.CustomerDTO{
					ID:        "cus_789",
					Type:      "company",
					LegalName: "Corp Inc",
					Status:    "active",
				},
			},
			arguments: map[string]any{
				"id": "cus_789",
				"billing_address": map[string]any{
					"street":  "456 Oak Ave",
					"city":    "New York",
					"country": "US",
				},
			},
			wantUpdateID: "cus_789",
			wantUpdateArg: &app.PatchCustomerCommand{
				BillingAddress: &app.AddressDTO{
					Street:  "456 Oak Ave",
					City:    "New York",
					Country: "US",
				},
			},
			wantResult: "Customer updated: cus_789\nType: company\nLegal name: Corp Inc\nStatus: active\n",
		},
		{
			name: "only provided fields are patched",
			service: &customerWriteServiceStub{
				updateRes: app.CustomerDTO{
					ID:        "cus_123",
					Type:      "company",
					LegalName: "Acme SRL",
					Status:    "active",
				},
			},
			arguments: map[string]any{
				"id": "cus_123",
				// No other fields - should result in empty patch
			},
			wantUpdateID:  "cus_123",
			wantUpdateArg: &app.PatchCustomerCommand{
				// All fields should be nil (not provided)
			},
			wantResult: "Customer updated: cus_123\nType: company\nLegal name: Acme SRL\nStatus: active\n",
		},
		{
			name:          "returns not-found error",
			service:       &customerWriteServiceStub{updateErr: app.ErrCustomerNotFound},
			arguments:     map[string]any{"id": "cus_nonexistent"},
			wantErr:       true,
			wantErrSubstr: "not found",
		},
		{
			name:          "returns error for unauthenticated request",
			service:       &customerWriteServiceStub{updateErr: app.ErrCustomerUpdateAccessDenied},
			arguments:     map[string]any{"id": "cus_123"},
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
				if tc.wantUpdateArg.Type != nil && *tc.service.updateArg.Type != *tc.wantUpdateArg.Type {
					t.Errorf("Update() type = %q, want %q", *tc.service.updateArg.Type, *tc.wantUpdateArg.Type)
				}
				if tc.wantUpdateArg.Phone != nil && *tc.service.updateArg.Phone != *tc.wantUpdateArg.Phone {
					t.Errorf("Update() phone = %q, want %q", *tc.service.updateArg.Phone, *tc.wantUpdateArg.Phone)
				}
				if tc.wantUpdateArg.Notes != nil && *tc.service.updateArg.Notes != *tc.wantUpdateArg.Notes {
					t.Errorf("Update() notes = %q, want %q", *tc.service.updateArg.Notes, *tc.wantUpdateArg.Notes)
				}
				if tc.wantUpdateArg.BillingAddress != nil {
					if tc.service.updateArg.BillingAddress == nil {
						t.Errorf("Update() billing_address = nil, want %+v", tc.wantUpdateArg.BillingAddress)
					} else if *tc.service.updateArg.BillingAddress != *tc.wantUpdateArg.BillingAddress {
						t.Errorf("Update() billing_address = %+v, want %+v", *tc.service.updateArg.BillingAddress, *tc.wantUpdateArg.BillingAddress)
					}
				}
			}

			if got := mcp.GetTextFromContent(result.Content[0]); got != tc.wantResult {
				t.Errorf("handler text = %q, want %q", got, tc.wantResult)
			}
		})
	}
}

func ptrToStr(s string) *string {
	return &s
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

// Contract-oriented tests that inspect the emitted MCP input schema
// These verify that the schema contains explicit guidance to help AI clients
// avoid common mistakes like using wrong field names (customer_type, name, company, top-level country).

func TestCustomerCreateToolSchemaContract(t *testing.T) {
	t.Parallel()

	tool, _ := customerCreateTool(nil, NewIngressGuard(nil), nil)

	// Test: Tool description contains field naming guidance
	t.Run("description_contains_field_naming_guidance", func(t *testing.T) {
		t.Parallel()
		desc := tool.Description
		if !strings.Contains(desc, "type") || !strings.Contains(desc, "NOT 'customer_type'") {
			t.Errorf("tool description should explicitly warn against 'customer_type' alias, got: %s", desc)
		}
		if !strings.Contains(desc, "legal_name") || !strings.Contains(strings.ToLower(desc), "not 'name'") {
			t.Errorf("tool description should explicitly warn against 'name' alias, got: %s", desc)
		}
		if !strings.Contains(desc, "billing_address") || !strings.Contains(desc, "top-level") {
			t.Errorf("tool description should mention that country must be inside billing_address (not top-level), got: %s", desc)
		}
	})

	// Test: 'type' property has anti-alias guidance
	t.Run("type_property_has_anti_alias_guidance", func(t *testing.T) {
		t.Parallel()
		typeProp, ok := tool.InputSchema.Properties["type"].(map[string]any)
		if !ok {
			t.Fatalf("type property not found or invalid type: %T", tool.InputSchema.Properties["type"])
		}
		desc, _ := typeProp["description"].(string)
		if !strings.Contains(desc, "'customer_type'") {
			t.Errorf("type property description should warn against 'customer_type' alias, got: %s", desc)
		}
		if !strings.Contains(desc, "company") || !strings.Contains(desc, "individual") {
			t.Errorf("type property description should mention valid values 'company' and 'individual', got: %s", desc)
		}
	})

	// Test: 'legal_name' property has anti-alias guidance
	t.Run("legal_name_property_has_anti_alias_guidance", func(t *testing.T) {
		t.Parallel()
		legalNameProp, ok := tool.InputSchema.Properties["legal_name"].(map[string]any)
		if !ok {
			t.Fatalf("legal_name property not found or invalid type: %T", tool.InputSchema.Properties["legal_name"])
		}
		desc, _ := legalNameProp["description"].(string)
		if !strings.Contains(desc, "'name'") || !strings.Contains(desc, "'company'") {
			t.Errorf("legal_name property description should warn against 'name' or 'company' aliases, got: %s", desc)
		}
	})

	// Test: 'billing_address' property has nesting guidance
	t.Run("billing_address_property_has_nesting_guidance", func(t *testing.T) {
		t.Parallel()
		billingAddrProp, ok := tool.InputSchema.Properties["billing_address"].(map[string]any)
		if !ok {
			t.Fatalf("billing_address property not found or invalid type: %T", tool.InputSchema.Properties["billing_address"])
		}
		desc, _ := billingAddrProp["description"].(string)
		if !strings.Contains(desc, "inside") && !strings.Contains(desc, "nested") {
			t.Errorf("billing_address property description should mention nesting requirement, got: %s", desc)
		}
		if !strings.Contains(desc, "top-level") {
			t.Errorf("billing_address property description should warn against top-level country, got: %s", desc)
		}
	})

	// Test: billing_address.country subfield has guidance about placement
	t.Run("country_subfield_has_placement_guidance", func(t *testing.T) {
		t.Parallel()
		billingAddrProp, ok := tool.InputSchema.Properties["billing_address"].(map[string]any)
		if !ok {
			t.Fatalf("billing_address property not found or invalid type")
		}
		props, _ := billingAddrProp["properties"].(map[string]any)
		if props == nil {
			t.Fatalf("billing_address.properties not found")
		}
		countryProp, ok := props["country"].(map[string]any)
		if !ok {
			t.Fatalf("country subproperty not found or invalid type")
		}
		desc, _ := countryProp["description"].(string)
		if !strings.Contains(desc, "billing_address") && !strings.Contains(desc, "inside") && !strings.Contains(desc, "top-level") {
			t.Errorf("country subfield description should mention it must be inside billing_address, got: %s", desc)
		}
	})

	// Test: Required fields are declared correctly
	t.Run("required_fields_declared", func(t *testing.T) {
		t.Parallel()
		required := tool.InputSchema.Required
		if len(required) == 0 {
			t.Error("customer.create should have required fields declared")
		}
		hasType := false
		hasLegalName := false
		for _, r := range required {
			if r == "type" {
				hasType = true
			}
			if r == "legal_name" {
				hasLegalName = true
			}
		}
		if !hasType {
			t.Error("customer.create should require 'type' field")
		}
		if !hasLegalName {
			t.Error("customer.create should require 'legal_name' field")
		}
	})

	// Test: Schema does NOT contain legacy 'json' property
	t.Run("no_legacy_json_property", func(t *testing.T) {
		t.Parallel()
		if _, exists := tool.InputSchema.Properties["json"]; exists {
			t.Error("customer.create schema should NOT have a legacy 'json' property")
		}
	})
}

func TestCustomerUpdateToolSchemaContract(t *testing.T) {
	t.Parallel()

	tool, _ := customerUpdateTool(nil, NewIngressGuard(nil), nil)

	// Test: Tool description contains field naming guidance
	t.Run("description_contains_field_naming_guidance", func(t *testing.T) {
		t.Parallel()
		desc := tool.Description
		if !strings.Contains(desc, "type") || !strings.Contains(desc, "NOT 'customer_type'") {
			t.Errorf("tool description should explicitly warn against 'customer_type' alias, got: %s", desc)
		}
		if !strings.Contains(desc, "legal_name") || !strings.Contains(strings.ToLower(desc), "not 'name'") {
			t.Errorf("tool description should explicitly warn against 'name' alias, got: %s", desc)
		}
		if !strings.Contains(desc, "billing_address") || !strings.Contains(desc, "top-level") {
			t.Errorf("tool description should mention that country must be inside billing_address (not top-level), got: %s", desc)
		}
	})

	// Test: 'type' property has anti-alias guidance
	t.Run("type_property_has_anti_alias_guidance", func(t *testing.T) {
		t.Parallel()
		typeProp, ok := tool.InputSchema.Properties["type"].(map[string]any)
		if !ok {
			t.Fatalf("type property not found or invalid type: %T", tool.InputSchema.Properties["type"])
		}
		desc, _ := typeProp["description"].(string)
		if !strings.Contains(desc, "'customer_type'") {
			t.Errorf("type property description should warn against 'customer_type' alias, got: %s", desc)
		}
	})

	// Test: 'legal_name' property has anti-alias guidance
	t.Run("legal_name_property_has_anti_alias_guidance", func(t *testing.T) {
		t.Parallel()
		legalNameProp, ok := tool.InputSchema.Properties["legal_name"].(map[string]any)
		if !ok {
			t.Fatalf("legal_name property not found or invalid type: %T", tool.InputSchema.Properties["legal_name"])
		}
		desc, _ := legalNameProp["description"].(string)
		if !strings.Contains(desc, "'name'") || !strings.Contains(desc, "'company'") {
			t.Errorf("legal_name property description should warn against 'name' or 'company' aliases, got: %s", desc)
		}
	})

	// Test: 'billing_address' property has nesting guidance
	t.Run("billing_address_property_has_nesting_guidance", func(t *testing.T) {
		t.Parallel()
		billingAddrProp, ok := tool.InputSchema.Properties["billing_address"].(map[string]any)
		if !ok {
			t.Fatalf("billing_address property not found or invalid type: %T", tool.InputSchema.Properties["billing_address"])
		}
		desc, _ := billingAddrProp["description"].(string)
		if !strings.Contains(desc, "inside") && !strings.Contains(desc, "nested") {
			t.Errorf("billing_address property description should mention nesting requirement, got: %s", desc)
		}
		if !strings.Contains(desc, "top-level") {
			t.Errorf("billing_address property description should warn against top-level country, got: %s", desc)
		}
	})

	// Test: billing_address.country subfield has guidance about placement
	t.Run("country_subfield_has_placement_guidance", func(t *testing.T) {
		t.Parallel()
		billingAddrProp, ok := tool.InputSchema.Properties["billing_address"].(map[string]any)
		if !ok {
			t.Fatalf("billing_address property not found or invalid type")
		}
		props, _ := billingAddrProp["properties"].(map[string]any)
		if props == nil {
			t.Fatalf("billing_address.properties not found")
		}
		countryProp, ok := props["country"].(map[string]any)
		if !ok {
			t.Fatalf("country subproperty not found or invalid type")
		}
		desc, _ := countryProp["description"].(string)
		if !strings.Contains(desc, "billing_address") && !strings.Contains(desc, "inside") && !strings.Contains(desc, "top-level") {
			t.Errorf("country subfield description should mention it must be inside billing_address, got: %s", desc)
		}
	})

	// Test: Required fields are declared correctly (id is required)
	t.Run("required_fields_declared", func(t *testing.T) {
		t.Parallel()
		required := tool.InputSchema.Required
		if len(required) == 0 {
			t.Error("customer.update should have required fields declared")
		}
		hasID := false
		for _, r := range required {
			if r == "id" {
				hasID = true
			}
		}
		if !hasID {
			t.Error("customer.update should require 'id' field")
		}
	})

	// Test: Schema does NOT contain legacy 'json' property
	t.Run("no_legacy_json_property", func(t *testing.T) {
		t.Parallel()
		if _, exists := tool.InputSchema.Properties["json"]; exists {
			t.Error("customer.update schema should NOT have a legacy 'json' property")
		}
	})
}

// Test that address subfield descriptions are improved
func TestBillingAddressSubfieldDescriptions(t *testing.T) {
	t.Parallel()

	createTool, _ := customerCreateTool(nil, NewIngressGuard(nil), nil)
	updateTool, _ := customerUpdateTool(nil, NewIngressGuard(nil), nil)

	for _, tool := range []mcp.Tool{createTool, updateTool} {
		toolName := tool.Name
		t.Run(toolName, func(t *testing.T) {
			t.Parallel()
			billingAddrProp, ok := tool.InputSchema.Properties["billing_address"].(map[string]any)
			if !ok {
				t.Fatalf("billing_address property not found")
			}
			props, _ := billingAddrProp["properties"].(map[string]any)
			if props == nil {
				t.Fatalf("billing_address.properties not found")
			}

			// Test: street has description with example
			streetProp, _ := props["street"].(map[string]any)
			streetDesc, _ := streetProp["description"].(string)
			if !strings.Contains(streetDesc, "Street") || !strings.Contains(streetDesc, "123") {
				t.Errorf("street subfield description should have example, got: %s", streetDesc)
			}

			// Test: city has description with example
			cityProp, _ := props["city"].(map[string]any)
			cityDesc, _ := cityProp["description"].(string)
			if !strings.Contains(cityDesc, "City") || !strings.Contains(cityDesc, "Santo Domingo") {
				t.Errorf("city subfield description should have example, got: %s", cityDesc)
			}

			// Test: country has placement guidance
			countryProp, _ := props["country"].(map[string]any)
			countryDesc, _ := countryProp["description"].(string)
			if !strings.Contains(countryDesc, "billing_address") && !strings.Contains(countryDesc, "inside") && !strings.Contains(countryDesc, "top-level") {
				t.Errorf("country subfield description should mention placement guidance, got: %s", countryDesc)
			}
		})
	}
}
