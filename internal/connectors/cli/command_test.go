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

type stubCustomerListService struct {
	called bool
	query  app.ListQuery
	result app.ListResult[app.CustomerDTO]
	err    error
}

func (s *stubCustomerListService) List(ctx context.Context, query app.ListQuery) (app.ListResult[app.CustomerDTO], error) {
	_ = ctx
	s.called = true
	s.query = query
	return s.result, s.err
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
			}, nil, false)

			if err := cmd.Run(context.Background(), tc.args, &out); err != nil {
				t.Fatalf("Run() error = %v", err)
			}

			if out.String() != tc.want {
				t.Fatalf("Run() output = %q, want %q", out.String(), tc.want)
			}
		})
	}
}

func TestCommandRunWritesCustomerListOutput(t *testing.T) {
	t.Parallel()

	baseResult := app.ListResult[app.CustomerDTO]{
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
			wantText:  "Billar Customers\n───────────────\nPage: 2\nPage size: 1\nTotal: 1\n\n1. Acme SRL\n   Trade name: Acme\n   Type: company\n   Status: active\n   Email: billing@acme.example\n   Default currency: USD\n   Created at: 2026-04-03T10:00:00Z\n   Updated at: 2026-04-03T10:05:00Z\n",
			wantQuery: app.ListQuery{Search: "Acme", SortField: "created_at", SortDir: "desc", Page: 2, PageSize: 1},
		},
		{
			name:      "json output",
			args:      []string{"customer", "list", "--format", "json"},
			wantJSON:  "{\"items\":[{\"id\":\"cus_123\",\"type\":\"company\",\"legal_name\":\"Acme SRL\",\"trade_name\":\"Acme\",\"tax_id\":\"\",\"email\":\"billing@acme.example\",\"phone\":\"\",\"website\":\"\",\"billing_address\":{\"street\":\"\",\"city\":\"\",\"state\":\"\",\"postal_code\":\"\",\"country\":\"\"},\"status\":\"active\",\"default_currency\":\"USD\",\"notes\":\"\",\"created_at\":\"2026-04-03T10:00:00Z\",\"updated_at\":\"2026-04-03T10:05:00Z\"}],\"total\":1,\"page\":2,\"page_size\":1}\n",
			wantQuery: app.ListQuery{Page: 1, PageSize: 20, SortField: "created_at", SortDir: "asc"},
		},
		{
			name:        "toon output",
			args:        []string{"customer", "list", "--format", "toon"},
			wantContain: "Acme SRL",
			wantQuery:   app.ListQuery{Page: 1, PageSize: 20, SortField: "created_at", SortDir: "asc"},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			service := &stubCustomerWriteService{stubCustomerListService: stubCustomerListService{result: baseResult}}
			var out bytes.Buffer
			cmd := NewCommand(stubHealthService{status: app.HealthDTO{Name: "billar", Status: "ok"}}, service, false)

			if err := cmd.Run(context.Background(), tc.args, &out); err != nil {
				t.Fatalf("Run() error = %v", err)
			}

			if !service.stubCustomerListService.called {
				t.Fatal("customer service was not called")
			}
			if service.stubCustomerListService.query != tc.wantQuery {
				t.Fatalf("Run() query = %+v, want %+v", service.stubCustomerListService.query, tc.wantQuery)
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
		service HealthStatusProvider
		wantErr string
	}{
		{
			name:    "missing command",
			args:    nil,
			service: stubHealthService{},
			wantErr: commandUsage,
		},
		{
			name:    "unknown command",
			args:    []string{"invoice"},
			service: stubHealthService{},
			wantErr: "unknown command",
		},
		{
			name:    "invalid format",
			args:    []string{"health", "--format", "yaml"},
			service: stubHealthService{},
			wantErr: "unsupported output format",
		},
		{
			name:    "extra positional args",
			args:    []string{"health", "extra"},
			service: stubHealthService{},
			wantErr: commandUsage,
		},
		{
			name: "service failure",
			args: []string{"health"},
			service: stubHealthService{
				err: errors.New("boom"),
			},
			wantErr: "run health command: boom",
		},
		{
			name:    "missing service",
			args:    []string{"health"},
			service: nil,
			wantErr: "health service is required",
		},
		{
			name:    "missing customer service",
			args:    []string{"customer", "list"},
			service: stubHealthService{},
			wantErr: "customer service is required",
		},
		{
			name:    "unknown customer subcommand",
			args:    []string{"customer", "foo"},
			service: stubHealthService{},
			wantErr: "unknown command",
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var customerService CustomerServiceProvider
			if tc.name == "unknown customer subcommand" {
				customerService = &stubCustomerWriteService{}
			}
			cmd := NewCommand(tc.service, customerService, false)
			if tc.name == "missing customer service" {
				cmd = NewCommand(stubHealthService{status: app.HealthDTO{Name: "billar", Status: "ok"}}, nil, false)
			}
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

// CustomerWriteServiceStub for testing CLI write operations
type stubCustomerWriteService struct {
	stubCustomerListService
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

func (s *stubCustomerWriteService) Create(ctx context.Context, cmd app.CreateCustomerCommand) (app.CustomerDTO, error) {
	_ = ctx
	s.createArg = &cmd
	return s.createRes, s.createErr
}

func (s *stubCustomerWriteService) Update(ctx context.Context, id string, cmd app.PatchCustomerCommand) (app.CustomerDTO, error) {
	_ = ctx
	s.updateID = id
	s.updateArg = &cmd
	return s.updateRes, s.updateErr
}

func (s *stubCustomerWriteService) Delete(ctx context.Context, id string) error {
	_ = ctx
	s.deleteID = id
	return s.deleteErr
}

func TestCommandCustomerCreateWithJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		service       *stubCustomerWriteService
		args          []string
		wantErr       bool
		wantErrSubstr string
		wantCreateArg *app.CreateCustomerCommand
		wantContain   string
	}{
		{
			name: "creates customer with valid JSON",
			service: &stubCustomerWriteService{
				createRes: app.CustomerDTO{
					ID:        "cus_123",
					Type:      "company",
					LegalName: "Acme SRL",
					Email:     "billing@acme.example",
					Status:    "active",
				},
			},
			args: []string{"customer", "create", "--json", `{"type":"company","legal_name":"Acme SRL","email":"billing@acme.example"}`},
			wantCreateArg: &app.CreateCustomerCommand{
				Type:      "company",
				LegalName: "Acme SRL",
				Email:     "billing@acme.example",
			},
			wantContain: "cus_123",
		},
		{
			name:          "returns error for malformed JSON",
			service:       &stubCustomerWriteService{},
			args:          []string{"customer", "create", "--json", `{invalid json}`},
			wantErr:       true,
			wantErrSubstr: "json:",
		},
		{
			name: "returns error for missing required field",
			service: &stubCustomerWriteService{
				createErr: errors.New("legal name is required"),
			},
			args:          []string{"customer", "create", "--json", `{"type":"company"}`},
			wantErr:       true,
			wantErrSubstr: "legal name is required",
		},
		{
			name: "returns error for unauthenticated request",
			service: &stubCustomerWriteService{
				createErr: app.ErrCustomerCreateAccessDenied,
			},
			args:          []string{"customer", "create", "--json", `{"type":"company","legal_name":"Test"}`},
			wantErr:       true,
			wantErrSubstr: "authenticated",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var out bytes.Buffer
			cmd := NewCommand(stubHealthService{status: app.HealthDTO{Name: "billar", Status: "ok"}}, tc.service, false)
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
				if tc.service.createArg.Type != tc.wantCreateArg.Type {
					t.Errorf("Create() type = %q, want %q", tc.service.createArg.Type, tc.wantCreateArg.Type)
				}
				if tc.service.createArg.LegalName != tc.wantCreateArg.LegalName {
					t.Errorf("Create() legal_name = %q, want %q", tc.service.createArg.LegalName, tc.wantCreateArg.LegalName)
				}
			}

			if tc.wantContain != "" && !strings.Contains(out.String(), tc.wantContain) {
				t.Errorf("Run() output = %q, want contains %q", out.String(), tc.wantContain)
			}
		})
	}
}

func TestCommandCustomerCreateWithFormatFlag(t *testing.T) {
	t.Parallel()

	service := &stubCustomerWriteService{
		createRes: app.CustomerDTO{
			ID:        "cus_123",
			Type:      "company",
			LegalName: "Acme SRL",
		},
	}

	tests := []struct {
		name        string
		args        []string
		wantContain string
	}{
		{
			name:        "text format default",
			args:        []string{"customer", "create", "--json", `{"type":"company","legal_name":"Acme"}`},
			wantContain: "cus_123",
		},
		{
			name:        "json format",
			args:        []string{"customer", "create", "--json", `{"type":"company","legal_name":"Acme"}`, "--format", "json"},
			wantContain: `"id":"cus_123"`,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var out bytes.Buffer
			cmd := NewCommand(stubHealthService{status: app.HealthDTO{Name: "billar", Status: "ok"}}, service, false)
			if err := cmd.Run(context.Background(), tc.args, &out); err != nil {
				t.Fatalf("Run() error = %v", err)
			}

			if !strings.Contains(out.String(), tc.wantContain) {
				t.Fatalf("Run() output = %q, want contains %q", out.String(), tc.wantContain)
			}
		})
	}
}

func TestCommandCustomerUpdateWithJSON(t *testing.T) {
	t.Parallel()

	email := "updated@example.com"
	notes := ""

	tests := []struct {
		name          string
		service       *stubCustomerWriteService
		args          []string
		wantErr       bool
		wantErrSubstr string
		wantUpdateID  string
		wantUpdateArg *app.PatchCustomerCommand
		wantContain   string
	}{
		{
			name: "updates customer with partial patch leaving other fields untouched",
			service: &stubCustomerWriteService{
				updateRes: app.CustomerDTO{
					ID:              "cus_123",
					Type:            "company",
					LegalName:       "Acme SRL",
					Email:           "updated@example.com",
					Notes:           "Old notes remain untouched",
					DefaultCurrency: "USD",
					Status:          "active",
				},
			},
			args:         []string{"customer", "update", "--id", "cus_123", "--json", `{"email":"updated@example.com"}`},
			wantUpdateID: "cus_123",
			wantUpdateArg: &app.PatchCustomerCommand{
				Email: &email,
			},
			wantContain: "cus_123",
		},
		{
			name: "patch with one field leaves other pointer fields nil",
			service: &stubCustomerWriteService{
				updateRes: app.CustomerDTO{
					ID:        "cus_123",
					Type:      "company",
					LegalName: "Acme SRL",
					Email:     "updated@example.com",
					Notes:     "Preserved notes",
					Status:    "active",
				},
			},
			args:         []string{"customer", "update", "--id", "cus_123", "--json", `{"email":"updated@example.com"}`},
			wantUpdateID: "cus_123",
			// Critical: only Email should be set, all other pointer fields should remain nil
			// to preserve PATCH semantics where omitted fields remain untouched
			wantUpdateArg: &app.PatchCustomerCommand{
				Email:           &email,
				Type:            nil,
				LegalName:       nil,
				TradeName:       nil,
				TaxID:           nil,
				Phone:           nil,
				Website:         nil,
				Notes:           nil, // This is what proves the "untouched" semantics
				DefaultCurrency: nil,
			},
			wantContain: "cus_123",
		},
		{
			name: "updates customer and clears notes field",
			service: &stubCustomerWriteService{
				updateRes: app.CustomerDTO{
					ID:              "cus_123",
					Type:            "company",
					LegalName:       "Acme SRL",
					Email:           "old@example.com",
					Notes:           "",
					DefaultCurrency: "USD",
					Status:          "active",
				},
			},
			args:         []string{"customer", "update", "--id", "cus_123", "--json", `{"notes":""}`},
			wantUpdateID: "cus_123",
			wantUpdateArg: &app.PatchCustomerCommand{
				Notes: &notes,
			},
			wantContain: "cus_123",
		},
		{
			name:          "returns error for malformed JSON",
			service:       &stubCustomerWriteService{},
			args:          []string{"customer", "update", "--id", "cus_123", "--json", `{invalid json}`},
			wantErr:       true,
			wantErrSubstr: "json:",
		},
		{
			name:          "returns error for missing id",
			service:       &stubCustomerWriteService{},
			args:          []string{"customer", "update", "--json", `{}`},
			wantErr:       true,
			wantErrSubstr: "id",
		},
		{
			name:          "returns error for not found",
			service:       &stubCustomerWriteService{updateErr: app.ErrCustomerNotFound},
			args:          []string{"customer", "update", "--id", "cus_nonexistent", "--json", `{}`},
			wantErr:       true,
			wantErrSubstr: "not found",
		},
		{
			name:          "returns error for unauthenticated request",
			service:       &stubCustomerWriteService{updateErr: app.ErrCustomerUpdateAccessDenied},
			args:          []string{"customer", "update", "--id", "cus_123", "--json", `{}`},
			wantErr:       true,
			wantErrSubstr: "authenticated",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var out bytes.Buffer
			cmd := NewCommand(stubHealthService{status: app.HealthDTO{Name: "billar", Status: "ok"}}, tc.service, false)
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
				// Verify specified fields match
				if tc.wantUpdateArg.Email != nil {
					if tc.service.updateArg.Email == nil || *tc.service.updateArg.Email != *tc.wantUpdateArg.Email {
						t.Errorf("Update() email = %v, want %v", tc.service.updateArg.Email, tc.wantUpdateArg.Email)
					}
				}
				if tc.wantUpdateArg.Notes != nil {
					if tc.service.updateArg.Notes == nil || *tc.service.updateArg.Notes != *tc.wantUpdateArg.Notes {
						t.Errorf("Update() notes = %v, want %v", tc.service.updateArg.Notes, tc.wantUpdateArg.Notes)
					}
				}
				// Verify unspecified fields are nil (PATCH semantics: omitted fields remain untouched)
				if tc.wantUpdateArg.Type == nil && tc.service.updateArg.Type != nil {
					t.Errorf("Update() Type should be nil for PATCH, got %v", *tc.service.updateArg.Type)
				}
				if tc.wantUpdateArg.LegalName == nil && tc.service.updateArg.LegalName != nil {
					t.Errorf("Update() LegalName should be nil for PATCH, got %v", *tc.service.updateArg.LegalName)
				}
				if tc.wantUpdateArg.TradeName == nil && tc.service.updateArg.TradeName != nil {
					t.Errorf("Update() TradeName should be nil for PATCH, got %v", *tc.service.updateArg.TradeName)
				}
				if tc.wantUpdateArg.TaxID == nil && tc.service.updateArg.TaxID != nil {
					t.Errorf("Update() TaxID should be nil for PATCH, got %v", *tc.service.updateArg.TaxID)
				}
				if tc.wantUpdateArg.Phone == nil && tc.service.updateArg.Phone != nil {
					t.Errorf("Update() Phone should be nil for PATCH, got %v", *tc.service.updateArg.Phone)
				}
				if tc.wantUpdateArg.Website == nil && tc.service.updateArg.Website != nil {
					t.Errorf("Update() Website should be nil for PATCH, got %v", *tc.service.updateArg.Website)
				}
				if tc.wantUpdateArg.Notes == nil && tc.service.updateArg.Notes != nil {
					t.Errorf("Update() Notes should be nil for PATCH, got %v", *tc.service.updateArg.Notes)
				}
				if tc.wantUpdateArg.DefaultCurrency == nil && tc.service.updateArg.DefaultCurrency != nil {
					t.Errorf("Update() DefaultCurrency should be nil for PATCH, got %v", *tc.service.updateArg.DefaultCurrency)
				}
			}

			if tc.wantContain != "" && !strings.Contains(out.String(), tc.wantContain) {
				t.Errorf("Run() output = %q, want contains %q", out.String(), tc.wantContain)
			}
		})
	}
}

func TestCommandCustomerDelete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		service       *stubCustomerWriteService
		args          []string
		wantErr       bool
		wantErrSubstr string
		wantDeleteID  string
		wantContain   string
	}{
		{
			name:         "deletes customer successfully",
			service:      &stubCustomerWriteService{},
			args:         []string{"customer", "delete", "--id", "cus_123"},
			wantDeleteID: "cus_123",
			wantContain:  "Deleted",
		},
		{
			name:          "returns error for missing id",
			service:       &stubCustomerWriteService{},
			args:          []string{"customer", "delete"},
			wantErr:       true,
			wantErrSubstr: "id",
		},
		{
			name:          "returns error for not found",
			service:       &stubCustomerWriteService{deleteErr: app.ErrCustomerNotFound},
			args:          []string{"customer", "delete", "--id", "cus_nonexistent"},
			wantErr:       true,
			wantErrSubstr: "not found",
		},
		{
			name:          "returns error for unauthenticated request",
			service:       &stubCustomerWriteService{deleteErr: app.ErrCustomerDeleteAccessDenied},
			args:          []string{"customer", "delete", "--id", "cus_123"},
			wantErr:       true,
			wantErrSubstr: "authenticated",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var out bytes.Buffer
			cmd := NewCommand(stubHealthService{status: app.HealthDTO{Name: "billar", Status: "ok"}}, tc.service, false)
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
