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
			name: "creates issuer profile successfully",
			service: &issuerProfileServiceStub{
				createRes: app.IssuerProfileDTO{
					ID:              "iss_123",
					LegalEntityID:   "le_456",
					DefaultCurrency: "USD",
				},
			},
			arguments: map[string]any{
				"legal_entity_id":  "le_456",
				"default_currency": "USD",
			},
			wantCreateArg: &app.CreateIssuerProfileCommand{
				LegalEntityID:   "le_456",
				DefaultCurrency: "USD",
			},
			wantResult: "Issuer profile created: iss_123\nLegal entity ID: le_456\nDefault currency: USD\n",
		},
		{
			name: "returns error for orphaned legal entity",
			service: &issuerProfileServiceStub{
				createErr: app.ErrLegalEntityNotFound,
			},
			arguments: map[string]any{
				"legal_entity_id":  "le_nonexistent",
				"default_currency": "USD",
			},
			wantErr:       true,
			wantErrSubstr: "not found",
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
				if tc.service.createArg.LegalEntityID != tc.wantCreateArg.LegalEntityID {
					t.Errorf("Create() legal_entity_id = %q, want %q", tc.service.createArg.LegalEntityID, tc.wantCreateArg.LegalEntityID)
				}
			}

			if got := mcp.GetTextFromContent(result.Content[0]); got != tc.wantResult {
				t.Errorf("handler text = %q, want %q", got, tc.wantResult)
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
