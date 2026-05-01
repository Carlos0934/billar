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
	restore app.BackupRestoreResultDTO
	err     error
	req     app.BackupRestoreRequest
}

func (s stubBackupService) Create(ctx context.Context) (app.BackupRecordDTO, error) {
	_ = ctx
	return s.created, s.err
}

func (s stubBackupService) List(ctx context.Context) (app.BackupListDTO, error) {
	_ = ctx
	return s.list, s.err
}

func (s *stubBackupService) Restore(ctx context.Context, req app.BackupRestoreRequest) (app.BackupRestoreResultDTO, error) {
	_ = ctx
	s.req = req
	return s.restore, s.err
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
				cmd.backup = &stubBackupService{created: created, list: list}
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

func TestBackupCommandRestoreParsingOutputsAndExitCodes(t *testing.T) {
	t.Parallel()

	result := app.BackupRestoreResultDTO{
		Backup:       app.BackupRecordDTO{ID: "bk_123", File: "/backups/bk_123.db", SidecarFile: "/backups/bk_123.db.json", SchemaVersion: 4, SizeBytes: 128, SHA256: "abc", Metadata: true},
		TargetDBPath: "/data/billar.db",
		DryRun:       true,
		Validation:   app.BackupValidation{OK: true, SidecarOK: true, HashOK: true, SizeOK: true, IntegrityOK: true, TablesOK: true, SchemaOK: true, BackupSchema: 4, BinarySchema: 4},
		Warnings:     []string{"concurrent_processes: stop other clients"},
	}

	t.Run("dry run calls service and writes DTO", func(t *testing.T) {
		svc := &stubBackupService{restore: result}
		cmd := NewCommand(stubHealthService{status: app.HealthDTO{Name: "billar", Status: "ok"}}, nil, nil, nil, nil, nil, nil, false).WithBackupService(svc)
		var out bytes.Buffer
		if err := cmd.Run(context.Background(), []string{"backup", "restore", "--id", "bk_123", "--dry-run", "--format", "json"}, &out); err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		if svc.req.BackupID != "bk_123" || !svc.req.DryRun || svc.req.File != "" {
			t.Fatalf("Restore request = %+v, want id dry-run", svc.req)
		}
		var dto app.BackupRestoreResultDTO
		if err := json.Unmarshal(out.Bytes(), &dto); err != nil || dto.Backup.ID != "bk_123" || !dto.Validation.OK {
			t.Fatalf("restore json dto/error = %+v/%v, want canonical DTO", dto, err)
		}
	})

	t.Run("file selector force and confirm parse", func(t *testing.T) {
		svc := &stubBackupService{restore: result}
		cmd := NewCommand(stubHealthService{status: app.HealthDTO{Name: "billar", Status: "ok"}}, nil, nil, nil, nil, nil, nil, false).WithBackupService(svc)
		var out bytes.Buffer
		if err := cmd.Run(context.Background(), []string{"backup", "restore", "--file", "/backups/bk_123.db", "--confirm", "bk_123.db", "--force"}, &out); err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		if svc.req.File != "/backups/bk_123.db" || svc.req.Confirm != "bk_123.db" || !svc.req.Force {
			t.Fatalf("Restore request = %+v, want file/confirm/force", svc.req)
		}
		if !strings.Contains(out.String(), "Billar Backup Restore") || !strings.Contains(out.String(), "validation.ok: true") {
			t.Fatalf("restore text output = %q, want title and validation", out.String())
		}
	})

	t.Run("mutual exclusion exits usage code 2", func(t *testing.T) {
		cmd := NewCommand(stubHealthService{status: app.HealthDTO{Name: "billar", Status: "ok"}}, nil, nil, nil, nil, nil, nil, false).WithBackupService(&stubBackupService{restore: result})
		var out bytes.Buffer
		err := cmd.Run(context.Background(), []string{"backup", "restore", "--id", "bk_123", "--file", "/backups/bk_123.db"}, &out)
		if err == nil || commandExitCode(err) != 2 || !strings.Contains(err.Error(), "exactly one") {
			t.Fatalf("Run() error/code = %v/%d, want selector usage code 2", err, commandExitCode(err))
		}
	})

	t.Run("service errors map to stable exit codes", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			err  error
			code int
		}{
			{name: "validation", err: errString("validate backup: hash_mismatch"), code: 3},
			{name: "runtime", err: errString("replace target database: rename failed"), code: 4},
			{name: "internal", err: errString("unexpected boom"), code: 5},
		} {
			t.Run(tc.name, func(t *testing.T) {
				cmd := NewCommand(stubHealthService{status: app.HealthDTO{Name: "billar", Status: "ok"}}, nil, nil, nil, nil, nil, nil, false).WithBackupService(&stubBackupService{err: tc.err})
				var out bytes.Buffer
				err := cmd.Run(context.Background(), []string{"backup", "restore", "--id", "bk_123", "--dry-run"}, &out)
				if err == nil || commandExitCode(err) != tc.code {
					t.Fatalf("Run() error/code = %v/%d, want code %d", err, commandExitCode(err), tc.code)
				}
			})
		}
	})

	t.Run("machine error output includes canonical error code and destructive warning", func(t *testing.T) {
		failureResult := result
		failureResult.DryRun = false
		failureResult.Validation.OK = false
		failureResult.Validation.HashOK = false
		failureResult.Warnings = []string{"concurrent_processes: stop other clients"}
		cmd := NewCommand(stubHealthService{status: app.HealthDTO{Name: "billar", Status: "ok"}}, nil, nil, nil, nil, nil, nil, false).WithBackupService(&stubBackupService{restore: failureResult, err: errString("validate backup: hash_mismatch")})
		var out bytes.Buffer
		err := cmd.Run(context.Background(), []string{"backup", "restore", "--id", "bk_123", "--force", "--format", "json"}, &out)
		if err == nil || commandExitCode(err) != 3 {
			t.Fatalf("Run() error/code = %v/%d, want validation code 3", err, commandExitCode(err))
		}
		var dto struct {
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
			Warnings []string `json:"warnings"`
		}
		if err := json.Unmarshal(out.Bytes(), &dto); err != nil {
			t.Fatalf("json error output invalid: %v\n%s", err, out.String())
		}
		if dto.Error.Code != "hash_mismatch" || !strings.Contains(dto.Error.Message, "hash_mismatch") || !containsText(dto.Warnings, "concurrent_processes") {
			t.Fatalf("json error dto = %+v, want hash_mismatch code/message and concurrent warning", dto)
		}
	})

	t.Run("usage error with machine format emits error code", func(t *testing.T) {
		cmd := NewCommand(stubHealthService{status: app.HealthDTO{Name: "billar", Status: "ok"}}, nil, nil, nil, nil, nil, nil, false).WithBackupService(&stubBackupService{restore: result})
		var out bytes.Buffer
		err := cmd.Run(context.Background(), []string{"backup", "restore", "--id", "bk_123", "--file", "/backups/bk_123.db", "--format", "json"}, &out)
		if err == nil || commandExitCode(err) != 2 {
			t.Fatalf("Run() error/code = %v/%d, want usage code 2", err, commandExitCode(err))
		}
		var dto struct {
			Error struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		if err := json.Unmarshal(out.Bytes(), &dto); err != nil || dto.Error.Code != "exactly_one_selector" {
			t.Fatalf("json usage dto/error = %+v/%v, want exactly_one_selector", dto, err)
		}
	})

	t.Run("json and toon share canonical fields", func(t *testing.T) {
		for _, format := range []string{"json", "toon"} {
			cmd := NewCommand(stubHealthService{status: app.HealthDTO{Name: "billar", Status: "ok"}}, nil, nil, nil, nil, nil, nil, false).WithBackupService(&stubBackupService{restore: result})
			var out bytes.Buffer
			if err := cmd.Run(context.Background(), []string{"backup", "restore", "--id", "bk_123", "--dry-run", "--format", format}, &out); err != nil {
				t.Fatalf("Run(%s) error = %v", format, err)
			}
			for _, want := range []string{"target_db_path", "validation", "bk_123"} {
				if !strings.Contains(out.String(), want) {
					t.Fatalf("%s output missing %q:\n%s", format, want, out.String())
				}
			}
		}
	})
}

type errString string

func (e errString) Error() string { return string(e) }

func containsText(values []string, want string) bool {
	for _, value := range values {
		if strings.Contains(value, want) {
			return true
		}
	}
	return false
}

func TestBackupCommandListJSONUsesBackupsFieldForEmptyList(t *testing.T) {
	t.Parallel()

	cmd := NewCommand(stubHealthService{status: app.HealthDTO{Name: "billar", Status: "ok"}}, nil, nil, nil, nil, nil, nil, false)
	cmd.backup = &stubBackupService{list: app.BackupListDTO{Dir: "/backups", Backups: []app.BackupRecordDTO{}}}

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
