package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Carlos0934/billar/internal/app"
)

type stubSetupService struct {
	report app.SetupReportDTO
	err    error
}

func (s stubSetupService) Run(ctx context.Context) (app.SetupReportDTO, error) {
	_ = ctx
	return s.report, s.err
}

func TestSetupCommandFormats(t *testing.T) {
	t.Parallel()

	report := app.SetupReportDTO{
		Project:        "billar",
		DBPath:         "/runtime/billar.db",
		ExportDir:      "/runtime/exports",
		BackupDir:      "/runtime/backups",
		Created:        []string{"/runtime/backups"},
		AlreadyExisted: []string{"/runtime", "/runtime/exports"},
		NextSteps:      []string{"billar issuer create", "billar doctor"},
		Warnings:       []string{"Backups contain sensitive billing data; protect them like the live database."},
	}

	for _, format := range []string{"text", "json", "toon"} {
		format := format
		t.Run(format, func(t *testing.T) {
			t.Parallel()

			cmd := NewCommand(stubHealthService{status: app.HealthDTO{Name: "billar", Status: "ok"}}, nil, nil, nil, nil, nil, nil, false)
			cmd.setup = stubSetupService{report: report}
			var out bytes.Buffer
			if err := cmd.Run(context.Background(), []string{"setup", "--format", format}, &out); err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			got := out.String()
			for _, want := range []string{"billar", "db_path", "/runtime/billar.db", "backup", "sensitive"} {
				if !strings.Contains(got, want) {
					t.Fatalf("%s output missing %q:\n%s", format, want, got)
				}
			}
			if strings.Contains(got, "super-secret") || strings.Contains(got, "api-key") || strings.Contains(got, "token") {
				t.Fatalf("setup output leaked secret-looking material: %q", got)
			}
			if format == "json" {
				var dto app.SetupReportDTO
				if err := json.Unmarshal(out.Bytes(), &dto); err != nil {
					t.Fatalf("json output invalid: %v", err)
				}
				if dto.BackupDir != report.BackupDir || len(dto.Created) != 1 || len(dto.AlreadyExisted) != 2 {
					t.Fatalf("json dto = %+v, want canonical setup payload", dto)
				}
			}
		})
	}
}
