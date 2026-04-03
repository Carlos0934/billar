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

			service := &stubCustomerListService{result: baseResult}
			var out bytes.Buffer
			cmd := NewCommand(stubHealthService{status: app.HealthDTO{Name: "billar", Status: "ok"}}, service, false)

			if err := cmd.Run(context.Background(), tc.args, &out); err != nil {
				t.Fatalf("Run() error = %v", err)
			}

			if !service.called {
				t.Fatal("customer service was not called")
			}
			if service.query != tc.wantQuery {
				t.Fatalf("Run() query = %+v, want %+v", service.query, tc.wantQuery)
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
			args:    []string{"customer", "create"},
			service: stubHealthService{},
			wantErr: "unknown command",
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cmd := NewCommand(tc.service, nil, false)
			if tc.name == "missing customer service" || tc.name == "unknown customer subcommand" {
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
