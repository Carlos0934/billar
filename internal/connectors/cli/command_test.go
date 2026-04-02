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
			}, false)

			if err := cmd.Run(context.Background(), tc.args, &out); err != nil {
				t.Fatalf("Run() error = %v", err)
			}

			if out.String() != tc.want {
				t.Fatalf("Run() output = %q, want %q", out.String(), tc.want)
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
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cmd := NewCommand(tc.service, false)
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
