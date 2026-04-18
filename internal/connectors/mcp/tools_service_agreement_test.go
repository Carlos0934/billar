package mcp

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Carlos0934/billar/internal/app"
	"github.com/mark3labs/mcp-go/mcp"
)

// agreementServiceStub implements AgreementServiceProvider for testing.
type agreementServiceStub struct {
	createArg     *app.CreateServiceAgreementCommand
	createRes     app.ServiceAgreementDTO
	createErr     error
	getID         string
	getRes        app.ServiceAgreementDTO
	getErr        error
	listProfileID string
	listRes       []app.ServiceAgreementDTO
	listErr       error
	updateRateID  string
	updateRateArg *app.UpdateServiceAgreementRateCommand
	updateRateRes app.ServiceAgreementDTO
	updateRateErr error
	activateID    string
	activateRes   app.ServiceAgreementDTO
	activateErr   error
	deactivateID  string
	deactivateRes app.ServiceAgreementDTO
	deactivateErr error
}

func (s *agreementServiceStub) Create(ctx context.Context, cmd app.CreateServiceAgreementCommand) (app.ServiceAgreementDTO, error) {
	_ = ctx
	s.createArg = &cmd
	return s.createRes, s.createErr
}

func (s *agreementServiceStub) Get(ctx context.Context, id string) (app.ServiceAgreementDTO, error) {
	_ = ctx
	s.getID = id
	return s.getRes, s.getErr
}

func (s *agreementServiceStub) ListByCustomerProfile(ctx context.Context, profileID string) ([]app.ServiceAgreementDTO, error) {
	_ = ctx
	s.listProfileID = profileID
	return s.listRes, s.listErr
}

func (s *agreementServiceStub) UpdateRate(ctx context.Context, id string, cmd app.UpdateServiceAgreementRateCommand) (app.ServiceAgreementDTO, error) {
	_ = ctx
	s.updateRateID = id
	s.updateRateArg = &cmd
	return s.updateRateRes, s.updateRateErr
}

func (s *agreementServiceStub) Activate(ctx context.Context, id string) (app.ServiceAgreementDTO, error) {
	_ = ctx
	s.activateID = id
	return s.activateRes, s.activateErr
}

func (s *agreementServiceStub) Deactivate(ctx context.Context, id string) (app.ServiceAgreementDTO, error) {
	_ = ctx
	s.deactivateID = id
	return s.deactivateRes, s.deactivateErr
}

// -- service_agreement.create --

func TestServiceAgreementCreateToolHandlers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		service       *agreementServiceStub
		arguments     map[string]any
		wantErr       bool
		wantErrSubstr string
		wantCreateArg *app.CreateServiceAgreementCommand
		wantResult    string
	}{
		{
			name: "creates service agreement successfully",
			service: &agreementServiceStub{
				createRes: app.ServiceAgreementDTO{
					ID:                "sa_123",
					CustomerProfileID: "cus_456",
					Name:              "Standard Support",
					BillingMode:       "hourly",
					HourlyRate:        15000,
					Currency:          "USD",
					Active:            false,
				},
			},
			arguments: map[string]any{
				"customer_profile_id": "cus_456",
				"name":                "Standard Support",
				"billing_mode":        "hourly",
				"hourly_rate":         float64(15000),
				"currency":            "USD",
			},
			wantCreateArg: &app.CreateServiceAgreementCommand{
				CustomerProfileID: "cus_456",
				Name:              "Standard Support",
				BillingMode:       "hourly",
				HourlyRate:        15000,
				Currency:          "USD",
			},
			wantResult: "sa_123",
		},
		{
			name: "returns error when customer_profile_id is missing",
			service: &agreementServiceStub{
				createErr: errors.New("customer profile not found"),
			},
			arguments:     map[string]any{},
			wantErr:       true,
			wantErrSubstr: "customer_profile_id",
		},
		{
			name:    "returns tool error when service is nil",
			service: nil,
			arguments: map[string]any{
				"customer_profile_id": "cus_456",
			},
			wantErr:       true,
			wantErrSubstr: "service agreement service is required",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var svc AgreementServiceProvider
			if tc.service != nil {
				svc = tc.service
			}
			_, handler := serviceAgreementCreateTool(svc, nil)
			result, err := handler(context.Background(), mcp.CallToolRequest{
				Params: mcp.CallToolParams{Name: "service_agreement.create", Arguments: tc.arguments},
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
				if tc.service.createArg.CustomerProfileID != tc.wantCreateArg.CustomerProfileID {
					t.Errorf("Create() customer_profile_id = %q, want %q", tc.service.createArg.CustomerProfileID, tc.wantCreateArg.CustomerProfileID)
				}
				if tc.service.createArg.Name != tc.wantCreateArg.Name {
					t.Errorf("Create() name = %q, want %q", tc.service.createArg.Name, tc.wantCreateArg.Name)
				}
				if tc.service.createArg.BillingMode != tc.wantCreateArg.BillingMode {
					t.Errorf("Create() billing_mode = %q, want %q", tc.service.createArg.BillingMode, tc.wantCreateArg.BillingMode)
				}
				if tc.service.createArg.HourlyRate != tc.wantCreateArg.HourlyRate {
					t.Errorf("Create() hourly_rate = %d, want %d", tc.service.createArg.HourlyRate, tc.wantCreateArg.HourlyRate)
				}
				if tc.service.createArg.Currency != tc.wantCreateArg.Currency {
					t.Errorf("Create() currency = %q, want %q", tc.service.createArg.Currency, tc.wantCreateArg.Currency)
				}
			}

			if tc.wantResult != "" {
				got := mcp.GetTextFromContent(result.Content[0])
				if !strings.Contains(got, tc.wantResult) {
					t.Errorf("handler text = %q, want contains %q", got, tc.wantResult)
				}
			}
		})
	}
}

// -- service_agreement.get --

func TestServiceAgreementGetToolHandlers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		service       *agreementServiceStub
		arguments     map[string]any
		wantErr       bool
		wantErrSubstr string
		wantGetID     string
		wantResult    string
	}{
		{
			name: "returns agreement by id",
			service: &agreementServiceStub{
				getRes: app.ServiceAgreementDTO{
					ID:                "sa_123",
					CustomerProfileID: "cus_456",
					Name:              "Standard Support",
					BillingMode:       "hourly",
					HourlyRate:        15000,
					Currency:          "USD",
					Active:            true,
				},
			},
			arguments:  map[string]any{"id": "sa_123"},
			wantGetID:  "sa_123",
			wantResult: "sa_123",
		},
		{
			name: "returns error for unknown id",
			service: &agreementServiceStub{
				getErr: app.ErrServiceAgreementNotFound,
			},
			arguments:     map[string]any{"id": "sa_nonexistent"},
			wantErr:       true,
			wantErrSubstr: "not found",
		},
		{
			name:          "returns error when id is missing",
			service:       &agreementServiceStub{},
			arguments:     map[string]any{},
			wantErr:       true,
			wantErrSubstr: "id",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, handler := serviceAgreementGetTool(tc.service, nil)
			result, err := handler(context.Background(), mcp.CallToolRequest{
				Params: mcp.CallToolParams{Name: "service_agreement.get", Arguments: tc.arguments},
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

			if tc.wantGetID != "" && tc.service.getID != tc.wantGetID {
				t.Errorf("Get() id = %q, want %q", tc.service.getID, tc.wantGetID)
			}

			if tc.wantResult != "" {
				got := mcp.GetTextFromContent(result.Content[0])
				if !strings.Contains(got, tc.wantResult) {
					t.Errorf("handler text = %q, want contains %q", got, tc.wantResult)
				}
			}
		})
	}
}

// -- service_agreement.list_by_customer_profile --

func TestServiceAgreementListByCustomerProfileToolHandlers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		service       *agreementServiceStub
		arguments     map[string]any
		wantErr       bool
		wantErrSubstr string
		wantProfileID string
		wantResult    string
	}{
		{
			name: "returns list for customer profile",
			service: &agreementServiceStub{
				listRes: []app.ServiceAgreementDTO{
					{
						ID:                "sa_001",
						CustomerProfileID: "cus_456",
						Name:              "Retainer",
						BillingMode:       "hourly",
						HourlyRate:        12000,
						Currency:          "USD",
						Active:            true,
					},
					{
						ID:                "sa_002",
						CustomerProfileID: "cus_456",
						Name:              "Support",
						BillingMode:       "hourly",
						HourlyRate:        8000,
						Currency:          "USD",
						Active:            false,
					},
				},
			},
			arguments:     map[string]any{"customer_profile_id": "cus_456"},
			wantProfileID: "cus_456",
			wantResult:    "sa_001",
		},
		{
			name:          "returns error when customer_profile_id missing",
			service:       &agreementServiceStub{},
			arguments:     map[string]any{},
			wantErr:       true,
			wantErrSubstr: "customer_profile_id",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, handler := serviceAgreementListTool(tc.service, nil)
			result, err := handler(context.Background(), mcp.CallToolRequest{
				Params: mcp.CallToolParams{Name: "service_agreement.list_by_customer_profile", Arguments: tc.arguments},
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

			if tc.wantProfileID != "" && tc.service.listProfileID != tc.wantProfileID {
				t.Errorf("ListByCustomerProfile() profileID = %q, want %q", tc.service.listProfileID, tc.wantProfileID)
			}

			if tc.wantResult != "" {
				got := mcp.GetTextFromContent(result.Content[0])
				if !strings.Contains(got, tc.wantResult) {
					t.Errorf("handler text = %q, want contains %q", got, tc.wantResult)
				}
			}
		})
	}
}

// -- service_agreement.update_rate --

func TestServiceAgreementUpdateRateToolHandlers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		service          *agreementServiceStub
		arguments        map[string]any
		wantErr          bool
		wantErrSubstr    string
		wantUpdateRateID string
		wantHourlyRate   int64
		wantResult       string
	}{
		{
			name: "updates rate successfully",
			service: &agreementServiceStub{
				updateRateRes: app.ServiceAgreementDTO{
					ID:         "sa_456",
					HourlyRate: 20000,
					Currency:   "USD",
				},
			},
			arguments:        map[string]any{"id": "sa_456", "hourly_rate": float64(20000)},
			wantUpdateRateID: "sa_456",
			wantHourlyRate:   20000,
			wantResult:       "sa_456",
		},
		{
			name:          "returns error for missing id",
			service:       &agreementServiceStub{},
			arguments:     map[string]any{"hourly_rate": float64(20000)},
			wantErr:       true,
			wantErrSubstr: "id",
		},
		{
			name: "returns service error when agreement not found",
			service: &agreementServiceStub{
				updateRateErr: app.ErrServiceAgreementNotFound,
			},
			arguments:     map[string]any{"id": "sa_nonexistent", "hourly_rate": float64(1000)},
			wantErr:       true,
			wantErrSubstr: "not found",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, handler := serviceAgreementUpdateRateTool(tc.service, nil)
			result, err := handler(context.Background(), mcp.CallToolRequest{
				Params: mcp.CallToolParams{Name: "service_agreement.update_rate", Arguments: tc.arguments},
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

			if tc.wantUpdateRateID != "" && tc.service.updateRateID != tc.wantUpdateRateID {
				t.Errorf("UpdateRate() id = %q, want %q", tc.service.updateRateID, tc.wantUpdateRateID)
			}

			if tc.wantHourlyRate > 0 {
				if tc.service.updateRateArg == nil {
					t.Fatal("UpdateRate() was not called")
				}
				if tc.service.updateRateArg.HourlyRate != tc.wantHourlyRate {
					t.Errorf("UpdateRate() hourly_rate = %d, want %d", tc.service.updateRateArg.HourlyRate, tc.wantHourlyRate)
				}
			}

			if tc.wantResult != "" {
				got := mcp.GetTextFromContent(result.Content[0])
				if !strings.Contains(got, tc.wantResult) {
					t.Errorf("handler text = %q, want contains %q", got, tc.wantResult)
				}
			}
		})
	}
}

// -- service_agreement.activate --

func TestServiceAgreementActivateToolHandlers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		service        *agreementServiceStub
		arguments      map[string]any
		wantErr        bool
		wantErrSubstr  string
		wantActivateID string
		wantResult     string
	}{
		{
			name: "activate succeeds",
			service: &agreementServiceStub{
				activateRes: app.ServiceAgreementDTO{ID: "sa_789", Active: true},
			},
			arguments:      map[string]any{"id": "sa_789"},
			wantActivateID: "sa_789",
			wantResult:     "sa_789",
		},
		{
			name:          "activate returns error for missing id",
			service:       &agreementServiceStub{},
			arguments:     map[string]any{},
			wantErr:       true,
			wantErrSubstr: "id",
		},
		{
			name: "activate returns not-found error",
			service: &agreementServiceStub{
				activateErr: app.ErrServiceAgreementNotFound,
			},
			arguments:     map[string]any{"id": "sa_nonexistent"},
			wantErr:       true,
			wantErrSubstr: "not found",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, handler := serviceAgreementActivateTool(tc.service, nil)
			result, err := handler(context.Background(), mcp.CallToolRequest{
				Params: mcp.CallToolParams{Arguments: tc.arguments},
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

			if tc.wantActivateID != "" && tc.service.activateID != tc.wantActivateID {
				t.Errorf("Activate() id = %q, want %q", tc.service.activateID, tc.wantActivateID)
			}

			if tc.wantResult != "" {
				got := mcp.GetTextFromContent(result.Content[0])
				if !strings.Contains(got, tc.wantResult) {
					t.Errorf("handler text = %q, want contains %q", got, tc.wantResult)
				}
			}
		})
	}
}

// -- service_agreement.deactivate --

func TestServiceAgreementDeactivateToolHandlers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		service          *agreementServiceStub
		arguments        map[string]any
		wantErr          bool
		wantErrSubstr    string
		wantDeactivateID string
		wantResult       string
	}{
		{
			name: "deactivate succeeds",
			service: &agreementServiceStub{
				deactivateRes: app.ServiceAgreementDTO{ID: "sa_789", Active: false},
			},
			arguments:        map[string]any{"id": "sa_789"},
			wantDeactivateID: "sa_789",
			wantResult:       "sa_789",
		},
		{
			name:          "deactivate returns error for missing id",
			service:       &agreementServiceStub{},
			arguments:     map[string]any{},
			wantErr:       true,
			wantErrSubstr: "id",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, handler := serviceAgreementDeactivateTool(tc.service, nil)
			result, err := handler(context.Background(), mcp.CallToolRequest{
				Params: mcp.CallToolParams{Arguments: tc.arguments},
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

			if tc.wantDeactivateID != "" && tc.service.deactivateID != tc.wantDeactivateID {
				t.Errorf("Deactivate() id = %q, want %q", tc.service.deactivateID, tc.wantDeactivateID)
			}

			if tc.wantResult != "" {
				got := mcp.GetTextFromContent(result.Content[0])
				if !strings.Contains(got, tc.wantResult) {
					t.Errorf("handler text = %q, want contains %q", got, tc.wantResult)
				}
			}
		})
	}
}

// TestServiceAgreementCreateToolHandlers_AllFields verifies that all fields bind
// correctly through the service_agreement.create typed handler.
func TestServiceAgreementCreateToolHandlers_AllFields(t *testing.T) {
	t.Parallel()

	svc := &agreementServiceStub{
		createRes: app.ServiceAgreementDTO{
			ID:                "sa_full",
			CustomerProfileID: "cus_001",
			Name:              "Full Plan",
			Description:       "Full-featured plan",
			BillingMode:       "hourly",
			HourlyRate:        50000,
			Currency:          "EUR",
			Active:            false,
		},
	}

	_, handler := serviceAgreementCreateTool(svc, nil)
	result, err := handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "service_agreement.create",
			Arguments: map[string]any{
				"customer_profile_id": "cus_001",
				"name":                "Full Plan",
				"description":         "Full-featured plan",
				"billing_mode":        "hourly",
				"hourly_rate":         float64(50000),
				"currency":            "EUR",
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
	if svc.createArg.CustomerProfileID != "cus_001" {
		t.Errorf("CustomerProfileID = %q, want %q", svc.createArg.CustomerProfileID, "cus_001")
	}
	if svc.createArg.Description != "Full-featured plan" {
		t.Errorf("Description = %q, want %q", svc.createArg.Description, "Full-featured plan")
	}
	if svc.createArg.HourlyRate != 50000 {
		t.Errorf("HourlyRate = %d, want %d", svc.createArg.HourlyRate, 50000)
	}
	if svc.createArg.Currency != "EUR" {
		t.Errorf("Currency = %q, want %q", svc.createArg.Currency, "EUR")
	}
}

// TestServiceAgreementUpdateRateToolHandlers_Int64Binding verifies that the
// hourly_rate integer field binds correctly via the typed handler.
func TestServiceAgreementUpdateRateToolHandlers_Int64Binding(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		rate     float64 // JSON numbers come in as float64 from map[string]any
		wantRate int64
	}{
		{name: "small rate", rate: 1000, wantRate: 1000},
		{name: "large rate", rate: 999999, wantRate: 999999},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := &agreementServiceStub{
				updateRateRes: app.ServiceAgreementDTO{ID: "sa_rate", HourlyRate: tc.wantRate, Currency: "USD"},
			}
			_, handler := serviceAgreementUpdateRateTool(svc, nil)
			result, err := handler(context.Background(), mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name:      "service_agreement.update_rate",
					Arguments: map[string]any{"id": "sa_rate", "hourly_rate": tc.rate},
				},
			})
			if err != nil {
				t.Fatalf("handler error = %v", err)
			}
			if result == nil || result.IsError {
				t.Fatalf("handler result = %+v, want success", result)
			}
			if svc.updateRateArg == nil {
				t.Fatal("UpdateRate() was not called")
			}
			if svc.updateRateArg.HourlyRate != tc.wantRate {
				t.Errorf("HourlyRate = %d, want %d", svc.updateRateArg.HourlyRate, tc.wantRate)
			}
		})
	}
}
