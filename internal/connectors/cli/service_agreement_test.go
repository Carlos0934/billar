package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Carlos0934/billar/internal/app"
)

// stubAgreementService is the test double for AgreementServiceProvider.
type stubAgreementService struct {
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

func (s *stubAgreementService) Create(ctx context.Context, cmd app.CreateServiceAgreementCommand) (app.ServiceAgreementDTO, error) {
	_ = ctx
	s.createArg = &cmd
	return s.createRes, s.createErr
}

func (s *stubAgreementService) Get(ctx context.Context, id string) (app.ServiceAgreementDTO, error) {
	_ = ctx
	s.getID = id
	return s.getRes, s.getErr
}

func (s *stubAgreementService) ListByCustomerProfile(ctx context.Context, profileID string) ([]app.ServiceAgreementDTO, error) {
	_ = ctx
	s.listProfileID = profileID
	return s.listRes, s.listErr
}

func (s *stubAgreementService) UpdateRate(ctx context.Context, id string, cmd app.UpdateServiceAgreementRateCommand) (app.ServiceAgreementDTO, error) {
	_ = ctx
	s.updateRateID = id
	s.updateRateArg = &cmd
	return s.updateRateRes, s.updateRateErr
}

func (s *stubAgreementService) Activate(ctx context.Context, id string) (app.ServiceAgreementDTO, error) {
	_ = ctx
	s.activateID = id
	return s.activateRes, s.activateErr
}

func (s *stubAgreementService) Deactivate(ctx context.Context, id string) (app.ServiceAgreementDTO, error) {
	_ = ctx
	s.deactivateID = id
	return s.deactivateRes, s.deactivateErr
}

// newTestAgreementCommand builds a minimal Command with a health stub and the given agreement service.
func newTestAgreementCommand(svc AgreementServiceProvider) Command {
	return NewCommand(stubHealthService{status: app.HealthDTO{Name: "billar", Status: "ok"}}, nil, nil, nil, svc, nil, nil, false)
}

// -- agreement create --

func TestAgreementCreateCLI(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		args          []string
		svc           *stubAgreementService
		wantErr       bool
		wantErrSubstr string
		wantContain   string
	}{
		{
			name: "creates agreement from json",
			args: []string{"agreement", "create", `--json={"customer_profile_id":"cus_1","name":"Retainer","billing_mode":"hourly","hourly_rate":15000,"currency":"USD"}`},
			svc: &stubAgreementService{
				createRes: app.ServiceAgreementDTO{
					ID:                "sa_123",
					CustomerProfileID: "cus_1",
					Name:              "Retainer",
					BillingMode:       "hourly",
					HourlyRate:        15000,
					Currency:          "USD",
				},
			},
			wantContain: "sa_123",
		},
		{
			name:          "returns error when --json is missing",
			args:          []string{"agreement", "create"},
			svc:           &stubAgreementService{},
			wantErr:       true,
			wantErrSubstr: "--json",
		},
		{
			name:          "returns error when json is invalid",
			args:          []string{"agreement", "create", "--json=not-json"},
			svc:           &stubAgreementService{},
			wantErr:       true,
			wantErrSubstr: "invalid",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var out bytes.Buffer
			cmd := newTestAgreementCommand(tc.svc)
			err := cmd.Run(context.Background(), tc.args, &out)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("Run() error = nil, want error")
				}
				if tc.wantErrSubstr != "" && !strings.Contains(err.Error(), tc.wantErrSubstr) {
					t.Errorf("Run() error = %q, want substring %q", err.Error(), tc.wantErrSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			if tc.wantContain != "" && !strings.Contains(out.String(), tc.wantContain) {
				t.Errorf("Run() output = %q, want contains %q", out.String(), tc.wantContain)
			}
		})
	}
}

// -- agreement get --

func TestAgreementGetCLI(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		args          []string
		svc           *stubAgreementService
		wantErr       bool
		wantErrSubstr string
		wantContain   string
		wantGetID     string
	}{
		{
			name: "gets agreement by id",
			args: []string{"agreement", "get", "--id=sa_123"},
			svc: &stubAgreementService{
				getRes: app.ServiceAgreementDTO{
					ID:                "sa_123",
					CustomerProfileID: "cus_1",
					Name:              "Retainer",
					BillingMode:       "hourly",
					HourlyRate:        15000,
					Currency:          "USD",
					Active:            true,
				},
			},
			wantGetID:   "sa_123",
			wantContain: "sa_123",
		},
		{
			name:          "returns error when --id is missing",
			args:          []string{"agreement", "get"},
			svc:           &stubAgreementService{},
			wantErr:       true,
			wantErrSubstr: "--id",
		},
		{
			name: "propagates service not found error",
			args: []string{"agreement", "get", "--id=sa_nonexistent"},
			svc: &stubAgreementService{
				getErr: errors.New("service agreement not found"),
			},
			wantErr:       true,
			wantErrSubstr: "not found",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var out bytes.Buffer
			cmd := newTestAgreementCommand(tc.svc)
			err := cmd.Run(context.Background(), tc.args, &out)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("Run() error = nil, want error")
				}
				if tc.wantErrSubstr != "" && !strings.Contains(err.Error(), tc.wantErrSubstr) {
					t.Errorf("Run() error = %q, want substring %q", err.Error(), tc.wantErrSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			if tc.wantGetID != "" && tc.svc.getID != tc.wantGetID {
				t.Errorf("Get() id = %q, want %q", tc.svc.getID, tc.wantGetID)
			}
			if tc.wantContain != "" && !strings.Contains(out.String(), tc.wantContain) {
				t.Errorf("Run() output = %q, want contains %q", out.String(), tc.wantContain)
			}
		})
	}
}

// -- agreement list --

func TestAgreementListCLI(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		args          []string
		svc           *stubAgreementService
		wantErr       bool
		wantErrSubstr string
		wantContain   string
		wantProfileID string
	}{
		{
			name: "lists agreements for customer profile",
			args: []string{"agreement", "list", "--customer-id=cus_1"},
			svc: &stubAgreementService{
				listRes: []app.ServiceAgreementDTO{
					{
						ID:                "sa_001",
						CustomerProfileID: "cus_1",
						Name:              "Retainer",
						BillingMode:       "hourly",
						HourlyRate:        12000,
						Currency:          "USD",
						Active:            true,
					},
				},
			},
			wantProfileID: "cus_1",
			wantContain:   "sa_001",
		},
		{
			name:          "returns error when --customer-id is missing",
			args:          []string{"agreement", "list"},
			svc:           &stubAgreementService{},
			wantErr:       true,
			wantErrSubstr: "--customer-id",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var out bytes.Buffer
			cmd := newTestAgreementCommand(tc.svc)
			err := cmd.Run(context.Background(), tc.args, &out)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("Run() error = nil, want error")
				}
				if tc.wantErrSubstr != "" && !strings.Contains(err.Error(), tc.wantErrSubstr) {
					t.Errorf("Run() error = %q, want substring %q", err.Error(), tc.wantErrSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			if tc.wantProfileID != "" && tc.svc.listProfileID != tc.wantProfileID {
				t.Errorf("ListByCustomerProfile() profileID = %q, want %q", tc.svc.listProfileID, tc.wantProfileID)
			}
			if tc.wantContain != "" && !strings.Contains(out.String(), tc.wantContain) {
				t.Errorf("Run() output = %q, want contains %q", out.String(), tc.wantContain)
			}
		})
	}
}

// -- agreement update-rate --

func TestAgreementUpdateRateCLI(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		args          []string
		svc           *stubAgreementService
		wantErr       bool
		wantErrSubstr string
		wantContain   string
		wantUpdateID  string
		wantRate      int64
	}{
		{
			name: "updates rate from json",
			args: []string{"agreement", "update-rate", "--id=sa_123", `--json={"hourly_rate":20000}`},
			svc: &stubAgreementService{
				updateRateRes: app.ServiceAgreementDTO{
					ID:         "sa_123",
					HourlyRate: 20000,
					Currency:   "USD",
				},
			},
			wantUpdateID: "sa_123",
			wantRate:     20000,
			wantContain:  "sa_123",
		},
		{
			name:          "returns error when --id is missing",
			args:          []string{"agreement", "update-rate", `--json={"hourly_rate":20000}`},
			svc:           &stubAgreementService{},
			wantErr:       true,
			wantErrSubstr: "--id",
		},
		{
			name:          "returns error when --json is missing",
			args:          []string{"agreement", "update-rate", "--id=sa_123"},
			svc:           &stubAgreementService{},
			wantErr:       true,
			wantErrSubstr: "--json",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var out bytes.Buffer
			cmd := newTestAgreementCommand(tc.svc)
			err := cmd.Run(context.Background(), tc.args, &out)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("Run() error = nil, want error")
				}
				if tc.wantErrSubstr != "" && !strings.Contains(err.Error(), tc.wantErrSubstr) {
					t.Errorf("Run() error = %q, want substring %q", err.Error(), tc.wantErrSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			if tc.wantUpdateID != "" && tc.svc.updateRateID != tc.wantUpdateID {
				t.Errorf("UpdateRate() id = %q, want %q", tc.svc.updateRateID, tc.wantUpdateID)
			}
			if tc.wantRate > 0 {
				if tc.svc.updateRateArg == nil {
					t.Fatal("UpdateRate() was not called")
				}
				if tc.svc.updateRateArg.HourlyRate != tc.wantRate {
					t.Errorf("UpdateRate() hourly_rate = %d, want %d", tc.svc.updateRateArg.HourlyRate, tc.wantRate)
				}
			}
			if tc.wantContain != "" && !strings.Contains(out.String(), tc.wantContain) {
				t.Errorf("Run() output = %q, want contains %q", out.String(), tc.wantContain)
			}
		})
	}
}

// -- agreement activate --

func TestAgreementActivateCLI(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		args           []string
		svc            *stubAgreementService
		wantErr        bool
		wantErrSubstr  string
		wantContain    string
		wantActivateID string
	}{
		{
			name: "activates agreement",
			args: []string{"agreement", "activate", "--id=sa_789"},
			svc: &stubAgreementService{
				activateRes: app.ServiceAgreementDTO{ID: "sa_789", Active: true},
			},
			wantActivateID: "sa_789",
			wantContain:    "sa_789",
		},
		{
			name:          "returns error when --id is missing",
			args:          []string{"agreement", "activate"},
			svc:           &stubAgreementService{},
			wantErr:       true,
			wantErrSubstr: "--id",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var out bytes.Buffer
			cmd := newTestAgreementCommand(tc.svc)
			err := cmd.Run(context.Background(), tc.args, &out)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("Run() error = nil, want error")
				}
				if tc.wantErrSubstr != "" && !strings.Contains(err.Error(), tc.wantErrSubstr) {
					t.Errorf("Run() error = %q, want substring %q", err.Error(), tc.wantErrSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			if tc.wantActivateID != "" && tc.svc.activateID != tc.wantActivateID {
				t.Errorf("Activate() id = %q, want %q", tc.svc.activateID, tc.wantActivateID)
			}
			if tc.wantContain != "" && !strings.Contains(out.String(), tc.wantContain) {
				t.Errorf("Run() output = %q, want contains %q", out.String(), tc.wantContain)
			}
		})
	}
}

// -- agreement deactivate --

func TestAgreementDeactivateCLI(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		args             []string
		svc              *stubAgreementService
		wantErr          bool
		wantErrSubstr    string
		wantContain      string
		wantDeactivateID string
	}{
		{
			name: "deactivates agreement",
			args: []string{"agreement", "deactivate", "--id=sa_789"},
			svc: &stubAgreementService{
				deactivateRes: app.ServiceAgreementDTO{ID: "sa_789", Active: false},
			},
			wantDeactivateID: "sa_789",
			wantContain:      "sa_789",
		},
		{
			name:          "returns error when --id is missing",
			args:          []string{"agreement", "deactivate"},
			svc:           &stubAgreementService{},
			wantErr:       true,
			wantErrSubstr: "--id",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var out bytes.Buffer
			cmd := newTestAgreementCommand(tc.svc)
			err := cmd.Run(context.Background(), tc.args, &out)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("Run() error = nil, want error")
				}
				if tc.wantErrSubstr != "" && !strings.Contains(err.Error(), tc.wantErrSubstr) {
					t.Errorf("Run() error = %q, want substring %q", err.Error(), tc.wantErrSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			if tc.wantDeactivateID != "" && tc.svc.deactivateID != tc.wantDeactivateID {
				t.Errorf("Deactivate() id = %q, want %q", tc.svc.deactivateID, tc.wantDeactivateID)
			}
			if tc.wantContain != "" && !strings.Contains(out.String(), tc.wantContain) {
				t.Errorf("Run() output = %q, want contains %q", out.String(), tc.wantContain)
			}
		})
	}
}

// -- unknown subcommand --

func TestAgreementUnknownSubcommand(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer
	cmd := newTestAgreementCommand(&stubAgreementService{})
	err := cmd.Run(context.Background(), []string{"agreement", "bogus"}, &out)
	if err == nil {
		t.Fatal("Run() error = nil, want error for unknown subcommand")
	}
	if !strings.Contains(err.Error(), "unknown") {
		t.Errorf("Run() error = %q, want contains 'unknown'", err.Error())
	}
}
