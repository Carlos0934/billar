package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Carlos0934/billar/internal/app"
)

type stubHealthService struct {
	status app.HealthDTO
	err    error
}

func (s stubHealthService) Status(ctx context.Context) (app.HealthDTO, error) {
	return s.status, s.err
}

// Stub services for CustomerProfile
type stubCustomerProfileListService struct {
	called bool
	query  app.ListQuery
	result app.ListResult[app.CustomerProfileDTO]
	err    error
}

func (s *stubCustomerProfileListService) List(ctx context.Context, query app.ListQuery) (app.ListResult[app.CustomerProfileDTO], error) {
	_ = ctx
	s.called = true
	s.query = query
	return s.result, s.err
}

type stubCustomerProfileWriteService struct {
	stubCustomerProfileListService
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

func (s *stubCustomerProfileWriteService) Create(ctx context.Context, cmd app.CreateCustomerProfileCommand) (app.CustomerProfileDTO, error) {
	_ = ctx
	s.createArg = &cmd
	return s.createRes, s.createErr
}

func (s *stubCustomerProfileWriteService) Get(ctx context.Context, id string) (app.CustomerProfileDTO, error) {
	_ = ctx
	return app.CustomerProfileDTO{}, nil
}

func (s *stubCustomerProfileWriteService) Update(ctx context.Context, id string, cmd app.PatchCustomerProfileCommand) (app.CustomerProfileDTO, error) {
	_ = ctx
	s.updateID = id
	s.updateArg = &cmd
	return s.updateRes, s.updateErr
}

func (s *stubCustomerProfileWriteService) Delete(ctx context.Context, id string) error {
	_ = ctx
	s.deleteID = id
	return s.deleteErr
}

// Stub services for LegalEntity
type stubLegalEntityListService struct {
	called bool
	query  app.ListQuery
	result app.ListResult[app.LegalEntityDTO]
	err    error
}

func (s *stubLegalEntityListService) List(ctx context.Context, query app.ListQuery) (app.ListResult[app.LegalEntityDTO], error) {
	_ = ctx
	s.called = true
	s.query = query
	return s.result, s.err
}

type stubLegalEntityWriteService struct {
	stubLegalEntityListService
	createArg *app.CreateLegalEntityCommand
	createRes app.LegalEntityDTO
	createErr error
	getID     string
	getRes    app.LegalEntityDTO
	getErr    error
	updateID  string
	updateArg *app.PatchLegalEntityCommand
	updateRes app.LegalEntityDTO
	updateErr error
	deleteID  string
	deleteErr error
}

func (s *stubLegalEntityWriteService) Create(ctx context.Context, cmd app.CreateLegalEntityCommand) (app.LegalEntityDTO, error) {
	_ = ctx
	s.createArg = &cmd
	return s.createRes, s.createErr
}

func (s *stubLegalEntityWriteService) Get(ctx context.Context, id string) (app.LegalEntityDTO, error) {
	_ = ctx
	s.getID = id
	return s.getRes, s.getErr
}

func (s *stubLegalEntityWriteService) Update(ctx context.Context, id string, cmd app.PatchLegalEntityCommand) (app.LegalEntityDTO, error) {
	_ = ctx
	s.updateID = id
	s.updateArg = &cmd
	return s.updateRes, s.updateErr
}

func (s *stubLegalEntityWriteService) Delete(ctx context.Context, id string) error {
	_ = ctx
	s.deleteID = id
	return s.deleteErr
}

// Stub services for IssuerProfile
type stubIssuerProfileWriteService struct {
	createArg *app.CreateIssuerProfileCommand
	createRes app.IssuerProfileDTO
	createErr error
	getID     string
	getRes    app.IssuerProfileDTO
	getErr    error
	updateID  string
	updateArg *app.PatchIssuerProfileCommand
	updateRes app.IssuerProfileDTO
	updateErr error
}

func (s *stubIssuerProfileWriteService) Create(ctx context.Context, cmd app.CreateIssuerProfileCommand) (app.IssuerProfileDTO, error) {
	_ = ctx
	s.createArg = &cmd
	return s.createRes, s.createErr
}

func (s *stubIssuerProfileWriteService) Get(ctx context.Context, id string) (app.IssuerProfileDTO, error) {
	_ = ctx
	s.getID = id
	return s.getRes, s.getErr
}

func (s *stubIssuerProfileWriteService) Update(ctx context.Context, id string, cmd app.PatchIssuerProfileCommand) (app.IssuerProfileDTO, error) {
	_ = ctx
	s.updateID = id
	s.updateArg = &cmd
	return s.updateRes, s.updateErr
}

func TestCommandRunWritesHealthOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "health command defaults to text",
			args: []string{"health"},
			want: "Billar Health\n─────────────\nStatus: ok\n",
		},
		{
			name: "status alias defaults to text",
			args: []string{"status"},
			want: "Billar Health\n─────────────\nStatus: ok\n",
		},
		{
			name: "writes json when requested",
			args: []string{"health", "--format", "json"},
			want: "{\"name\":\"billar\",\"status\":\"ok\"}\n",
		},
		{
			name: "writes toon when requested",
			args: []string{"health", "--format", "toon"},
			want: "name: billar\nstatus: ok\n",
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var out bytes.Buffer
			cmd := NewCommand(stubHealthService{
				status: app.HealthDTO{Name: "billar", Status: "ok"},
			}, nil, nil, nil, nil, nil, nil, false)

			if err := cmd.Run(context.Background(), tc.args, &out); err != nil {
				t.Fatalf("Run() error = %v", err)
			}

			if out.String() != tc.want {
				t.Fatalf("Run() output = %q, want %q", out.String(), tc.want)
			}
		})
	}
}

func TestCommandRunWritesCustomerProfileListOutput(t *testing.T) {
	t.Parallel()

	baseResult := app.ListResult[app.CustomerProfileDTO]{
		Items: []app.CustomerProfileDTO{{
			ID:              "cus_123",
			LegalEntityID:   "le_abc",
			Status:          "active",
			DefaultCurrency: "USD",
			CreatedAt:       "2026-04-03T10:00:00Z",
			UpdatedAt:       "2026-04-03T10:05:00Z",
		}},
		Total:    1,
		Page:     2,
		PageSize: 1,
	}

	tests := []struct {
		name        string
		args        []string
		wantText    string
		wantJSON    string
		wantContain string
		wantQuery   app.ListQuery
	}{
		{
			name:      "text output",
			args:      []string{"customer", "list", "--search", "  Acme  ", "--sort", "created_at:desc", "--page", "2", "--page-size", "1"},
			wantText:  "Customer Profiles\n───────────────\nPage: 2\nPage size: 1\nTotal: 1\n\n1. cus_123\n   Legal entity ID: le_abc\n   Status: active\n   Default currency: USD\n   Created at: 2026-04-03T10:00:00Z\n   Updated at: 2026-04-03T10:05:00Z\n",
			wantQuery: app.ListQuery{Search: "Acme", SortField: "created_at", SortDir: "desc", Page: 2, PageSize: 1},
		},
		{
			name:      "json output",
			args:      []string{"customer", "list", "--format", "json"},
			wantJSON:  "{\"items\":[{\"id\":\"cus_123\",\"legal_entity_id\":\"le_abc\",\"status\":\"active\",\"default_currency\":\"USD\",\"notes\":\"\",\"created_at\":\"2026-04-03T10:00:00Z\",\"updated_at\":\"2026-04-03T10:05:00Z\"}],\"total\":1,\"page\":2,\"page_size\":1}\n",
			wantQuery: app.ListQuery{Page: 1, PageSize: 20, SortField: "created_at", SortDir: "asc"},
		},
		{
			name:        "toon output",
			args:        []string{"customer", "list", "--format", "toon"},
			wantContain: "cus_123",
			wantQuery:   app.ListQuery{Page: 1, PageSize: 20, SortField: "created_at", SortDir: "asc"},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			customerService := &stubCustomerProfileWriteService{stubCustomerProfileListService: stubCustomerProfileListService{result: baseResult}}
			var out bytes.Buffer
			cmd := NewCommand(stubHealthService{status: app.HealthDTO{Name: "billar", Status: "ok"}}, nil, nil, customerService, nil, nil, nil, false)

			if err := cmd.Run(context.Background(), tc.args, &out); err != nil {
				t.Fatalf("Run() error = %v", err)
			}

			if !customerService.stubCustomerProfileListService.called {
				t.Fatal("customer service was not called")
			}
			if customerService.stubCustomerProfileListService.query != tc.wantQuery {
				t.Fatalf("Run() query = %+v, want %+v", customerService.stubCustomerProfileListService.query, tc.wantQuery)
			}

			switch {
			case tc.wantText != "":
				if out.String() != tc.wantText {
					t.Fatalf("Run() output = %q, want %q", out.String(), tc.wantText)
				}
			case tc.wantJSON != "":
				if out.String() != tc.wantJSON {
					t.Fatalf("Run() output = %q, want %q", out.String(), tc.wantJSON)
				}
			default:
				if !strings.Contains(out.String(), tc.wantContain) {
					t.Fatalf("Run() output = %q, want contains %q", out.String(), tc.wantContain)
				}
			}
		})
	}
}

func TestCommandRunRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "missing command",
			args:    nil,
			wantErr: commandUsage,
		},
		{
			name:    "unknown command",
			args:    []string{"unknown-command"},
			wantErr: "unknown command",
		},
		{
			name:    "invalid format",
			args:    []string{"health", "--format", "yaml"},
			wantErr: "unsupported output format",
		},
		{
			name:    "extra positional args",
			args:    []string{"health", "extra"},
			wantErr: commandUsage,
		},
		{
			name:    "missing customer service",
			args:    []string{"customer", "list"},
			wantErr: "customer service is required",
		},
		{
			name:    "unknown customer subcommand",
			args:    []string{"customer", "foo"},
			wantErr: "unknown command",
		},
		{
			name:    "missing legal entity service",
			args:    []string{"legal-entity", "list"},
			wantErr: "legal entity service is required",
		},
		{
			name:    "unknown legal-entity subcommand",
			args:    []string{"legal-entity", "foo"},
			wantErr: "unknown command",
		},
		{
			name:    "missing issuer service",
			args:    []string{"issuer", "create"},
			wantErr: "issuer service is required",
		},
		{
			name:    "unknown issuer subcommand",
			args:    []string{"issuer", "foo"},
			wantErr: "unknown command",
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var customerService CustomerProfileServiceProvider
			var legalEntityService LegalEntityServiceProvider
			var issuerService IssuerProfileServiceProvider

			// Provide services only for commands that need them
			if tc.name == "unknown customer subcommand" {
				customerService = &stubCustomerProfileWriteService{}
			}
			if tc.name == "unknown legal-entity subcommand" {
				legalEntityService = &stubLegalEntityWriteService{}
			}
			if tc.name == "unknown issuer subcommand" {
				issuerService = &stubIssuerProfileWriteService{}
			}

			cmd := NewCommand(stubHealthService{status: app.HealthDTO{Name: "billar", Status: "ok"}}, legalEntityService, issuerService, customerService, nil, nil, nil, false)
			err := cmd.Run(context.Background(), tc.args, &bytes.Buffer{})
			if err == nil {
				t.Fatal("Run() error = nil, want non-nil")
			}

			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("Run() error = %q, want substring %q", err.Error(), tc.wantErr)
			}
		})
	}
}

func TestCommandCustomerProfileCreateWithJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		service       *stubCustomerProfileWriteService
		args          []string
		wantErr       bool
		wantErrSubstr string
		wantCreateArg *app.CreateCustomerProfileCommand
		wantContain   string
	}{
		{
			name: "creates customer profile with valid JSON",
			service: &stubCustomerProfileWriteService{
				createRes: app.CustomerProfileDTO{
					ID:              "cus_123",
					LegalEntityID:   "le_abc",
					Status:          "active",
					DefaultCurrency: "USD",
				},
			},
			args: []string{"customer", "create", "--json", `{"type":"company","legal_name":"Acme SRL","default_currency":"USD"}`},
			wantCreateArg: &app.CreateCustomerProfileCommand{
				LegalEntityType: "company",
				LegalName:       "Acme SRL",
				DefaultCurrency: "USD",
			},
			wantContain: "cus_123",
		},
		{
			name:          "returns error for malformed JSON",
			service:       &stubCustomerProfileWriteService{},
			args:          []string{"customer", "create", "--json", `{invalid json}`},
			wantErr:       true,
			wantErrSubstr: "json:",
		},
		{
			name: "returns error for missing required field",
			service: &stubCustomerProfileWriteService{
				createErr: errors.New("legal entity id is required"),
			},
			args:          []string{"customer", "create", "--json", `{}`},
			wantErr:       true,
			wantErrSubstr: "legal entity id is required",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var out bytes.Buffer
			cmd := NewCommand(stubHealthService{status: app.HealthDTO{Name: "billar", Status: "ok"}}, nil, nil, tc.service, nil, nil, nil, false)
			err := cmd.Run(context.Background(), tc.args, &out)

			if tc.wantErr {
				if err == nil {
					t.Fatal("Run() error = nil, want non-nil")
				}
				if tc.wantErrSubstr != "" && !strings.Contains(err.Error(), tc.wantErrSubstr) {
					t.Fatalf("Run() error = %q, want substring %q", err.Error(), tc.wantErrSubstr)
				}
				return
			}

			if err != nil {
				t.Fatalf("Run() error = %v", err)
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

			if tc.wantContain != "" && !strings.Contains(out.String(), tc.wantContain) {
				t.Errorf("Run() output = %q, want contains %q", out.String(), tc.wantContain)
			}
		})
	}
}

func TestCommandCustomerProfileCreateWithFormatFlag(t *testing.T) {
	t.Parallel()

	service := &stubCustomerProfileWriteService{
		createRes: app.CustomerProfileDTO{
			ID:              "cus_123",
			LegalEntityID:   "le_abc",
			DefaultCurrency: "USD",
		},
	}

	tests := []struct {
		name        string
		args        []string
		wantContain string
	}{
		{
			name:        "text format default",
			args:        []string{"customer", "create", "--json", `{"type":"company","legal_name":"Acme SRL","default_currency":"USD"}`},
			wantContain: "cus_123",
		},
		{
			name:        "json format",
			args:        []string{"customer", "create", "--json", `{"type":"company","legal_name":"Acme SRL","default_currency":"USD"}`, "--format", "json"},
			wantContain: `"id":"cus_123"`,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var out bytes.Buffer
			cmd := NewCommand(stubHealthService{status: app.HealthDTO{Name: "billar", Status: "ok"}}, nil, nil, service, nil, nil, nil, false)
			if err := cmd.Run(context.Background(), tc.args, &out); err != nil {
				t.Fatalf("Run() error = %v", err)
			}

			if !strings.Contains(out.String(), tc.wantContain) {
				t.Fatalf("Run() output = %q, want contains %q", out.String(), tc.wantContain)
			}
		})
	}
}

func TestCommandCustomerProfileUpdateWithJSON(t *testing.T) {
	t.Parallel()

	status := "inactive"

	tests := []struct {
		name          string
		service       *stubCustomerProfileWriteService
		args          []string
		wantErr       bool
		wantErrSubstr string
		wantUpdateID  string
		wantUpdateArg *app.PatchCustomerProfileCommand
		wantContain   string
	}{
		{
			name: "updates customer profile with partial patch",
			service: &stubCustomerProfileWriteService{
				updateRes: app.CustomerProfileDTO{
					ID:              "cus_123",
					LegalEntityID:   "le_abc",
					Status:          "inactive",
					DefaultCurrency: "USD",
				},
			},
			args:         []string{"customer", "update", "--id", "cus_123", "--json", `{"status":"inactive"}`},
			wantUpdateID: "cus_123",
			wantUpdateArg: &app.PatchCustomerProfileCommand{
				Status: &status,
			},
			wantContain: "cus_123",
		},
		{
			name:          "returns error for malformed JSON",
			service:       &stubCustomerProfileWriteService{},
			args:          []string{"customer", "update", "--id", "cus_123", "--json", `{invalid json}`},
			wantErr:       true,
			wantErrSubstr: "json:",
		},
		{
			name:          "returns error for missing id",
			service:       &stubCustomerProfileWriteService{},
			args:          []string{"customer", "update", "--json", `{}`},
			wantErr:       true,
			wantErrSubstr: "id",
		},
		{
			name:          "returns error for not found",
			service:       &stubCustomerProfileWriteService{updateErr: app.ErrCustomerProfileNotFound},
			args:          []string{"customer", "update", "--id", "cus_nonexistent", "--json", `{}`},
			wantErr:       true,
			wantErrSubstr: "not found",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var out bytes.Buffer
			cmd := NewCommand(stubHealthService{status: app.HealthDTO{Name: "billar", Status: "ok"}}, nil, nil, tc.service, nil, nil, nil, false)
			err := cmd.Run(context.Background(), tc.args, &out)

			if tc.wantErr {
				if err == nil {
					t.Fatal("Run() error = nil, want non-nil")
				}
				if tc.wantErrSubstr != "" && !strings.Contains(err.Error(), tc.wantErrSubstr) {
					t.Fatalf("Run() error = %q, want substring %q", err.Error(), tc.wantErrSubstr)
				}
				return
			}

			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}

			if tc.wantUpdateID != "" && tc.service.updateID != tc.wantUpdateID {
				t.Errorf("Update() id = %q, want %q", tc.service.updateID, tc.wantUpdateID)
			}

			if tc.wantUpdateArg != nil && tc.service.updateArg != nil {
				if tc.wantUpdateArg.Status != nil {
					if tc.service.updateArg.Status == nil || *tc.service.updateArg.Status != *tc.wantUpdateArg.Status {
						t.Errorf("Update() status = %v, want %v", tc.service.updateArg.Status, tc.wantUpdateArg.Status)
					}
				}
			}

			if tc.wantContain != "" && !strings.Contains(out.String(), tc.wantContain) {
				t.Errorf("Run() output = %q, want contains %q", out.String(), tc.wantContain)
			}
		})
	}
}

func TestCommandCustomerProfileDelete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		service       *stubCustomerProfileWriteService
		args          []string
		wantErr       bool
		wantErrSubstr string
		wantDeleteID  string
		wantContain   string
	}{
		{
			name:         "deletes customer profile successfully",
			service:      &stubCustomerProfileWriteService{},
			args:         []string{"customer", "delete", "--id", "cus_123"},
			wantDeleteID: "cus_123",
			wantContain:  "Deleted",
		},
		{
			name:          "returns error for missing id",
			service:       &stubCustomerProfileWriteService{},
			args:          []string{"customer", "delete"},
			wantErr:       true,
			wantErrSubstr: "id",
		},
		{
			name:          "returns error for not found",
			service:       &stubCustomerProfileWriteService{deleteErr: app.ErrCustomerProfileNotFound},
			args:          []string{"customer", "delete", "--id", "cus_nonexistent"},
			wantErr:       true,
			wantErrSubstr: "not found",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var out bytes.Buffer
			cmd := NewCommand(stubHealthService{status: app.HealthDTO{Name: "billar", Status: "ok"}}, nil, nil, tc.service, nil, nil, nil, false)
			err := cmd.Run(context.Background(), tc.args, &out)

			if tc.wantErr {
				if err == nil {
					t.Fatal("Run() error = nil, want non-nil")
				}
				if tc.wantErrSubstr != "" && !strings.Contains(err.Error(), tc.wantErrSubstr) {
					t.Fatalf("Run() error = %q, want substring %q", err.Error(), tc.wantErrSubstr)
				}
				return
			}

			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}

			if tc.wantDeleteID != "" && tc.service.deleteID != tc.wantDeleteID {
				t.Errorf("Delete() id = %q, want %q", tc.service.deleteID, tc.wantDeleteID)
			}

			if tc.wantContain != "" && !strings.Contains(out.String(), tc.wantContain) {
				t.Errorf("Run() output = %q, want contains %q", out.String(), tc.wantContain)
			}
		})
	}
}
