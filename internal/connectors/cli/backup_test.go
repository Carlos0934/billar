package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/Carlos0934/billar/internal/app"
)

type stubBackupService struct {
	created app.BackupRecordDTO
	list    app.BackupListDTO
	err     error
}

func (s stubBackupService) Create(ctx context.Context) (app.BackupRecordDTO, error) {
	_ = ctx
	return s.created, s.err
}

func (s stubBackupService) List(ctx context.Context) (app.BackupListDTO, error) {
	_ = ctx
	return s.list, s.err
}

func TestBackupCommandCreateAndListFormats(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2026, 5, 1, 15, 0, 0, 0, time.UTC)
	created := app.BackupRecordDTO{ID: "billar-20260501T150000Z-schema7", File: "/backups/billar-20260501T150000Z-schema7.db", SidecarFile: "/backups/billar-20260501T150000Z-schema7.db.json", CreatedAt: createdAt, SchemaVersion: 7, SizeBytes: 4096, SHA256: "abc123", Metadata: true, Warnings: []string{"Backups contain sensitive billing data; protect them like the live database."}}
	list := app.BackupListDTO{Dir: "/backups", Backups: []app.BackupRecordDTO{created, {ID: "orphan", File: "/backups/orphan.db", SizeBytes: 128, Metadata: false}}}

	for _, tc := range []struct {
		name string
		args []string
	}{
		{name: "create", args: []string{"backup", "create"}},
		{name: "list", args: []string{"backup", "list"}},
	} {
		tc := tc
		for _, format := range []string{"text", "json", "toon"} {
			format := format
			t.Run(tc.name+"_"+format, func(t *testing.T) {
				t.Parallel()
				cmd := NewCommand(stubHealthService{status: app.HealthDTO{Name: "billar", Status: "ok"}}, nil, nil, nil, nil, nil, nil, false)
				cmd.backup = stubBackupService{created: created, list: list}
				var out bytes.Buffer
				args := append(append([]string{}, tc.args...), "--format", format)
				if err := cmd.Run(context.Background(), args, &out); err != nil {
					t.Fatalf("Run() error = %v", err)
				}
				got := out.String()
				for _, want := range []string{"billar-20260501T150000Z-schema7", ".db", "metadata"} {
					if !strings.Contains(got, want) {
						t.Fatalf("%s %s output missing %q:\n%s", tc.name, format, want, got)
					}
				}
				if tc.name == "create" && !strings.Contains(strings.ToLower(got), "sensitive") {
					t.Fatalf("backup create %s output missing sensitive-data warning:\n%s", format, got)
				}
				if format == "json" {
					if tc.name == "create" {
						var dto app.BackupRecordDTO
						if err := json.Unmarshal(out.Bytes(), &dto); err != nil || dto.SHA256 != "abc123" || dto.SizeBytes != 4096 {
							t.Fatalf("json create dto/error = %+v/%v, want canonical backup record", dto, err)
						}
					} else {
						var raw map[string]json.RawMessage
						if err := json.Unmarshal(out.Bytes(), &raw); err != nil {
							t.Fatalf("json list output invalid: %v", err)
						}
						if _, hasRecords := raw["records"]; hasRecords {
							t.Fatalf("json list fields = %#v, want backups field instead of records", raw)
						}
						var dto app.BackupListDTO
						if err := json.Unmarshal(out.Bytes(), &dto); err != nil || len(dto.Backups) != 2 || dto.Backups[1].Metadata {
							t.Fatalf("json list dto/error = %+v/%v, want mixed metadata records", dto, err)
						}
					}
				}
			})
		}
	}
}

func TestBackupCommandRestoreDeferred(t *testing.T) {
	t.Parallel()

	cmd := NewCommand(stubHealthService{status: app.HealthDTO{Name: "billar", Status: "ok"}}, nil, nil, nil, nil, nil, nil, false)
	cmd.backup = stubBackupService{}
	var out bytes.Buffer
	err := cmd.Run(context.Background(), []string{"backup", "restore"}, &out)
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "deferred") {
		t.Fatalf("backup restore error = %v, want deferred restore message", err)
	}
}

func TestBackupCommandListJSONUsesBackupsFieldForEmptyList(t *testing.T) {
	t.Parallel()

	cmd := NewCommand(stubHealthService{status: app.HealthDTO{Name: "billar", Status: "ok"}}, nil, nil, nil, nil, nil, nil, false)
	cmd.backup = stubBackupService{list: app.BackupListDTO{Dir: "/backups", Backups: []app.BackupRecordDTO{}}}

	var out bytes.Buffer
	if err := cmd.Run(context.Background(), []string{"backup", "list", "--format", "json"}, &out); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(out.Bytes(), &raw); err != nil {
		t.Fatalf("json output invalid: %v", err)
	}
	if _, ok := raw["backups"]; !ok {
		t.Fatalf("json fields = %#v, want backups field", raw)
	}
	if _, ok := raw["records"]; ok {
		t.Fatalf("json fields = %#v, did not want records compatibility field", raw)
	}
	var dto app.BackupListDTO
	if err := json.Unmarshal(out.Bytes(), &dto); err != nil || dto.Backups == nil || len(dto.Backups) != 0 {
		t.Fatalf("json dto/error = %+v/%v, want non-nil empty backups", dto, err)
	}
}
