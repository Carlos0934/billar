package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Carlos0934/billar/internal/app"
)

type stubDoctorService struct {
	report app.DoctorReportDTO
	err    error
}

func (s stubDoctorService) Report(ctx context.Context) (app.DoctorReportDTO, error) {
	_ = ctx
	return s.report, s.err
}

func TestDoctorCommandFormatsAndPosture(t *testing.T) {
	t.Parallel()

	report := app.DoctorReportDTO{
		Project: "billar", DBPath: "/runtime/billar.db", SchemaVersion: 7, DBReachable: true,
		DBPathSource: "configured", DBParentDir: "/runtime", DBParentDirSource: "configured", DBParentDirExists: true, DBParentDirWritable: true,
		ExportDir: "/runtime/exports", ExportDirSource: "default", ExportDirSet: false, ExportDirExists: false, ExportDirWritable: false,
		BackupDir: "/runtime/backups", BackupDirSource: "default", BackupDirExists: true, BackupDirWritable: true,
		PDFAvailable:  true,
		CommandHealth: []app.DoctorCommandHealthDTO{{Name: "db", Status: "ok", Message: "reachable"}, {Name: "export_dir", Status: "fail", Message: "BILLAR_EXPORT_DIR is not configured"}},
		NextSteps:     []string{"billar setup"},
	}
	for _, format := range []string{"text", "json", "toon"} {
		format := format
		t.Run(format, func(t *testing.T) {
			t.Parallel()
			cmd := NewCommand(stubHealthService{status: app.HealthDTO{Name: "billar", Status: "ok"}}, nil, nil, nil, nil, nil, nil, false)
			cmd.doctor = stubDoctorService{report: report}
			var out bytes.Buffer
			err := cmd.Run(context.Background(), []string{"doctor", "--format", format}, &out)
			if err == nil || !strings.Contains(err.Error(), "doctor readiness failed") {
				t.Fatalf("Run() error = %v, want readiness failure after output", err)
			}
			got := out.String()
			for _, want := range []string{"billar", "backup_dir", "/runtime/backups", "source", "writable", "billar setup"} {
				if !strings.Contains(got, want) {
					t.Fatalf("%s output missing %q:\n%s", format, want, got)
				}
			}
			if strings.Contains(strings.ToLower(got), "mcp") {
				t.Fatalf("%s output contains removed MCP readiness fields:\n%s", format, got)
			}
			if strings.Contains(got, "token") || strings.Contains(got, "api-key") || strings.Contains(got, "super-secret") {
				t.Fatalf("doctor output leaked secret-looking material: %q", got)
			}
			if format == "json" {
				var dto app.DoctorReportDTO
				if err := json.Unmarshal(out.Bytes(), &dto); err != nil {
					t.Fatalf("json output invalid: %v", err)
				}
				if dto.ExportDirSet {
					t.Fatalf("json dto = %+v, want stable doctor booleans", dto)
				}
				var fields map[string]any
				if err := json.Unmarshal(out.Bytes(), &fields); err != nil {
					t.Fatalf("json output invalid for field map: %v", err)
				}
				for _, removed := range []string{"mcp_configured", "mcp_trusted_writes", "mcp_listen_addr"} {
					if _, ok := fields[removed]; ok {
						t.Fatalf("json output contains removed field %q in %#v", removed, fields)
					}
				}
			}
		})
	}
}

func TestDoctorCommandReturnsErrorAfterWritingFailingReadiness(t *testing.T) {
	t.Parallel()

	report := app.DoctorReportDTO{
		Project: "billar",
		DBPath:  "/missing/billar.db",
		CommandHealth: []app.DoctorCommandHealthDTO{
			{Name: "db_parent_dir", Status: "fail", Message: "database parent directory does not exist"},
		},
		NextSteps: []string{"billar setup"},
	}
	cmd := NewCommand(stubHealthService{status: app.HealthDTO{Name: "billar", Status: "ok"}}, nil, nil, nil, nil, nil, nil, false)
	cmd.doctor = stubDoctorService{report: report}

	var out bytes.Buffer
	err := cmd.Run(context.Background(), []string{"doctor", "--format", "json"}, &out)
	if err == nil || !strings.Contains(err.Error(), "doctor readiness failed") {
		t.Fatalf("Run() error = %v, want readiness failure", err)
	}
	if !strings.Contains(out.String(), "billar setup") || !strings.Contains(out.String(), "db_parent_dir") {
		t.Fatalf("doctor output = %q, want readiness report written before error", out.String())
	}
}
