package mcp

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Carlos0934/billar/internal/app"
	"github.com/mark3labs/mcp-go/mcp"
)

// issuerProfileServiceStub for testing
type issuerProfileServiceStub struct {
	createArg *app.CreateIssuerProfileCommand
	createRes app.IssuerProfileDTO
	createErr error
	updateID  string
	updateArg *app.PatchIssuerProfileCommand
	updateRes app.IssuerProfileDTO
	updateErr error
	deleteID  string
	deleteErr error
}

func (s *issuerProfileServiceStub) Create(ctx context.Context, cmd app.CreateIssuerProfileCommand) (app.IssuerProfileDTO, error) {
	_ = ctx
	s.createArg = &cmd
	return s.createRes, s.createErr
}

func (s *issuerProfileServiceStub) Get(ctx context.Context, id string) (app.IssuerProfileDTO, error) {
	_ = ctx
	return app.IssuerProfileDTO{}, errors.New("not implemented in test stub")
}

func (s *issuerProfileServiceStub) Update(ctx context.Context, id string, cmd app.PatchIssuerProfileCommand) (app.IssuerProfileDTO, error) {
	_ = ctx
	s.updateID = id
	s.updateArg = &cmd
	return s.updateRes, s.updateErr
}

func (s *issuerProfileServiceStub) Delete(ctx context.Context, id string) error {
	_ = ctx
	s.deleteID = id
	return s.deleteErr
}

// TestIssuerProfileCreateToolHandlers_BillingAddress verifies billing_address
// deserialization through the create handler.
func TestIssuerProfileCreateToolHandlers_BillingAddress(t *testing.T) {
	t.Parallel()

	svc := &issuerProfileServiceStub{
		createRes: app.IssuerProfileDTO{
			ID:            "iss_addr_01",
			LegalEntityID: "le_001",
		},
	}

	_, handler := issuerProfileCreateTool(svc, NewIngressGuard(nil), nil)
	result, err := handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "issuer_profile.create",
			Arguments: map[string]any{
				"type":             "company",
				"legal_name":       "Billing Co",
				"default_currency": "EUR",
				"billing_address": map[string]any{
					"street":  "Av. Libertad 10",
					"city":    "Madrid",
					"country": "ES",
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
	if svc.createArg.BillingAddress.Street != "Av. Libertad 10" {
		t.Errorf("BillingAddress.Street = %q, want %q", svc.createArg.BillingAddress.Street, "Av. Libertad 10")
	}
	if svc.createArg.BillingAddress.Country != "ES" {
		t.Errorf("BillingAddress.Country = %q, want %q", svc.createArg.BillingAddress.Country, "ES")
	}
}

// TestIssuerProfileUpdateToolHandlers_PatchPointerSemantics verifies absent vs.
// explicitly-cleared field distinction in the issuer update handler.
func TestIssuerProfileUpdateToolHandlers_PatchPointerSemantics(t *testing.T) {
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
				"id":               "iss_123",
				"default_currency": "USD",
				// email absent
			},
			wantEmailNil:   true,
			wantAddressNil: true,
		},
		{
			name: "explicitly cleared email field is non-nil pointer to empty string",
			arguments: map[string]any{
				"id":    "iss_123",
				"email": "",
			},
			wantEmailNil:   false,
			wantEmailValue: "",
			wantAddressNil: true,
		},
		{
			name: "billing_address provided binds nested struct",
			arguments: map[string]any{
				"id": "iss_123",
				"billing_address": map[string]any{
					"country": "DE",
					"city":    "Frankfurt",
				},
			},
			wantEmailNil:       true,
			wantAddressNil:     false,
			wantAddressCountry: "DE",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := &issuerProfileServiceStub{
				updateRes: app.IssuerProfileDTO{ID: "iss_123", LegalEntityID: "le_456"},
			}
			_, handler := issuerProfileUpdateTool(svc, NewIngressGuard(nil), nil)
			result, err := handler(context.Background(), mcp.CallToolRequest{
				Params: mcp.CallToolParams{Name: "issuer_profile.update", Arguments: tc.arguments},
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

func TestIssuerProfileCreateToolHandlers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		service       *issuerProfileServiceStub
		arguments     map[string]any
		wantErr       bool
		wantErrSubstr string
		wantCreateArg *app.CreateIssuerProfileCommand
		wantResult    string
	}{
		{
			name: "creates issuer profile successfully with inline legal entity",
			service: &issuerProfileServiceStub{
				createRes: app.IssuerProfileDTO{
					ID:              "iss_123",
					LegalEntityID:   "le_456",
					DefaultCurrency: "USD",
				},
			},
			arguments: map[string]any{
				"type":             "company",
				"legal_name":       "Acme SRL",
				"default_currency": "USD",
			},
			wantCreateArg: &app.CreateIssuerProfileCommand{
				LegalEntityType: "company",
				LegalName:       "Acme SRL",
				DefaultCurrency: "USD",
			},
			wantResult: "Issuer profile created: iss_123\nLegal entity ID: le_456\nDefault currency: USD\n",
		},
		{
			name: "returns error when service rejects command",
			service: &issuerProfileServiceStub{
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

			_, handler := issuerProfileCreateTool(tc.service, NewIngressGuard(nil), nil)
			result, err := handler(context.Background(), mcp.CallToolRequest{
				Params: mcp.CallToolParams{Name: "issuer_profile.create", Arguments: tc.arguments},
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

func TestIssuerProfileDeleteToolHandlers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		service       *issuerProfileServiceStub
		arguments     map[string]any
		wantErr       bool
		wantErrSubstr string
		wantDeleteID  string
	}{
		{
			name:         "deletes issuer profile successfully",
			service:      &issuerProfileServiceStub{},
			arguments:    map[string]any{"id": "iss_123"},
			wantDeleteID: "iss_123",
		},
		{
			name:          "returns not-found error",
			service:       &issuerProfileServiceStub{deleteErr: app.ErrIssuerProfileNotFound},
			arguments:     map[string]any{"id": "iss_nonexistent"},
			wantErr:       true,
			wantErrSubstr: "not found",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, handler := issuerProfileDeleteTool(tc.service, NewIngressGuard(nil), nil)
			result, err := handler(context.Background(), mcp.CallToolRequest{
				Params: mcp.CallToolParams{Name: "issuer_profile.delete", Arguments: tc.arguments},
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
func TestIssuerProfileUpdateToolHandlers_LEFields(t *testing.T) {
	t.Parallel()

	legalName := "Acme Corp"

	tests := []struct {
		name          string
		service       *issuerProfileServiceStub
		arguments     map[string]any
		wantErr       bool
		wantErrSubstr string
		wantUpdateID  string
		wantUpdateArg *app.PatchIssuerProfileCommand
	}{
		{
			name: "passes legal_name LE field in patch command",
			service: &issuerProfileServiceStub{
				updateRes: app.IssuerProfileDTO{
					ID:            "iss_123",
					LegalEntityID: "le_456",
				},
			},
			arguments: map[string]any{
				"id":         "iss_123",
				"legal_name": "Acme Corp",
			},
			wantUpdateID: "iss_123",
			wantUpdateArg: &app.PatchIssuerProfileCommand{
				LegalName: &legalName,
			},
		},
		{
			name: "passes billing_address LE field in patch command",
			service: &issuerProfileServiceStub{
				updateRes: app.IssuerProfileDTO{
					ID:            "iss_123",
					LegalEntityID: "le_456",
				},
			},
			arguments: map[string]any{
				"id":              "iss_123",
				"billing_address": map[string]any{"country": "DO", "city": "Santo Domingo"},
			},
			wantUpdateID: "iss_123",
		},
		{
			name: "returns service error for LE cascade failure",
			service: &issuerProfileServiceStub{
				updateErr: app.ErrIssuerProfileNotFound,
			},
			arguments:     map[string]any{"id": "iss_nonexistent", "legal_name": "X"},
			wantErr:       true,
			wantErrSubstr: "not found",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, handler := issuerProfileUpdateTool(tc.service, NewIngressGuard(nil), nil)
			result, err := handler(context.Background(), mcp.CallToolRequest{
				Params: mcp.CallToolParams{Name: "issuer_profile.update", Arguments: tc.arguments},
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

			if tc.wantUpdateArg != nil && tc.wantUpdateArg.LegalName != nil {
				if tc.service.updateArg == nil || tc.service.updateArg.LegalName == nil {
					t.Fatal("Update() LegalName not set")
				}
				if *tc.service.updateArg.LegalName != *tc.wantUpdateArg.LegalName {
					t.Errorf("Update() LegalName = %q, want %q", *tc.service.updateArg.LegalName, *tc.wantUpdateArg.LegalName)
				}
			}
		})
	}
}

func TestIssuerProfileUpdateToolHandlers(t *testing.T) {
	t.Parallel()

	currency := "EUR"
	tests := []struct {
		name          string
		service       *issuerProfileServiceStub
		arguments     map[string]any
		wantErr       bool
		wantErrSubstr string
		wantUpdateID  string
		wantUpdateArg *app.PatchIssuerProfileCommand
		wantResult    string
	}{
		{
			name: "updates default currency successfully",
			service: &issuerProfileServiceStub{
				updateRes: app.IssuerProfileDTO{
					ID:              "iss_123",
					LegalEntityID:   "le_456",
					DefaultCurrency: "EUR",
				},
			},
			arguments: map[string]any{
				"id":               "iss_123",
				"default_currency": "EUR",
			},
			wantUpdateID: "iss_123",
			wantUpdateArg: &app.PatchIssuerProfileCommand{
				DefaultCurrency: &currency,
			},
			wantResult: "Issuer profile updated: iss_123\nLegal entity ID: le_456\nDefault currency: EUR\n",
		},
		{
			name:          "returns not-found error",
			service:       &issuerProfileServiceStub{updateErr: app.ErrIssuerProfileNotFound},
			arguments:     map[string]any{"id": "iss_nonexistent"},
			wantErr:       true,
			wantErrSubstr: "not found",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, handler := issuerProfileUpdateTool(tc.service, NewIngressGuard(nil), nil)
			result, err := handler(context.Background(), mcp.CallToolRequest{
				Params: mcp.CallToolParams{Name: "issuer_profile.update", Arguments: tc.arguments},
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
