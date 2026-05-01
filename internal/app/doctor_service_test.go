package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type doctorStoreStub struct {
	pingErr       error
	schemaVersion int
	schemaErr     error
}

type doctorDBProbeStub struct {
	schemaVersion int
	err           error
	paths         []string
}

func (s doctorStoreStub) Ping(context.Context) error { return s.pingErr }

func (s doctorStoreStub) SchemaVersion(context.Context) (int, error) {
	return s.schemaVersion, s.schemaErr
}

func (s *doctorDBProbeStub) Probe(_ context.Context, dbPath string) (int, error) {
	s.paths = append(s.paths, dbPath)
	return s.schemaVersion, s.err
}

func TestDoctorServiceReportHealthyEnvironment(t *testing.T) {
	t.Parallel()

	svc := NewDoctorService(doctorStoreStub{schemaVersion: 7}, DoctorConfig{
		Project:      "billar",
		DBPath:       "/runtime/billar.db",
		ExportDir:    t.TempDir(),
		PDFAvailable: true,
	})

	report, err := svc.Report(context.Background())
	if err != nil {
		t.Fatalf("Report() error = %v", err)
	}
	if report.Project != "billar" || report.DBPath != "/runtime/billar.db" || report.SchemaVersion != 7 {
		t.Fatalf("identity/schema = (%q,%q,%d), want configured runtime values", report.Project, report.DBPath, report.SchemaVersion)
	}
	if !report.DBReachable || !report.ExportDirSet || !report.ExportDirWritable || !report.PDFAvailable {
		t.Fatalf("health booleans = %+v, want all configured checks true", report)
	}
	if len(report.CommandHealth) != 5 {
		t.Fatalf("len(CommandHealth) = %d, want db/db_parent/export_dir/backup_dir/pdf entries", len(report.CommandHealth))
	}
	rendered := fmt.Sprintf("%+v", report)
	if strings.Contains(strings.ToLower(rendered), "secret") || strings.Contains(rendered, "super-secret-token") {
		t.Fatalf("Report leaked secret material: %q", rendered)
	}
}

func TestDoctorServiceReportDegradedEnvironment(t *testing.T) {
	t.Parallel()

	svc := NewDoctorService(doctorStoreStub{pingErr: errors.New("database locked"), schemaErr: errors.New("no schema")}, DoctorConfig{Project: "billar", DBPath: "/runtime/billar.db"})

	report, err := svc.Report(context.Background())
	if err != nil {
		t.Fatalf("Report() error = %v", err)
	}
	if report.DBReachable || report.ExportDirSet || report.ExportDirWritable || report.PDFAvailable {
		t.Fatalf("degraded booleans = %+v, want all optional/runtime checks false", report)
	}
	if report.SchemaVersion != 0 {
		t.Fatalf("SchemaVersion = %d, want 0 when schema probe fails", report.SchemaVersion)
	}
	if len(report.CommandHealth) != 5 || report.CommandHealth[0].Status != "fail" || !strings.Contains(report.CommandHealth[0].Message, "database locked") {
		t.Fatalf("db command health = %+v, want failing db entry", report.CommandHealth)
	}
}

func TestDoctorServiceReportRuntimePathReadiness(t *testing.T) {
	t.Parallel()

	runtimeRoot := t.TempDir()
	exportDir := filepath.Join(runtimeRoot, "exports")
	backupDir := filepath.Join(runtimeRoot, "backups")
	if err := os.Mkdir(exportDir, 0o700); err != nil {
		t.Fatalf("mkdir export dir: %v", err)
	}

	svc := NewDoctorService(doctorStoreStub{schemaVersion: 7}, DoctorConfig{
		Project:         "billar",
		DBPath:          filepath.Join(runtimeRoot, "billar.db"),
		DBPathSource:    "configured",
		ExportDir:       exportDir,
		ExportDirSource: "configured",
		BackupDir:       backupDir,
		BackupDirSource: "default",
		PDFAvailable:    true,
	})

	report, err := svc.Report(context.Background())
	if err != nil {
		t.Fatalf("Report() error = %v", err)
	}
	if report.DBPathSource != "configured" || report.ExportDirSource != "configured" || report.BackupDirSource != "default" {
		t.Fatalf("path sources = db:%q export:%q backup:%q, want configured/configured/default", report.DBPathSource, report.ExportDirSource, report.BackupDirSource)
	}
	if !report.DBParentDirExists || !report.DBParentDirWritable || !report.ExportDirExists || !report.ExportDirWritable {
		t.Fatalf("existing path readiness = %+v, want db parent and export dir present+writable", report)
	}
	if report.BackupDir != backupDir || report.BackupDirExists || report.BackupDirWritable {
		t.Fatalf("backup readiness = (%q,%t,%t), want missing backup dir reported", report.BackupDir, report.BackupDirExists, report.BackupDirWritable)
	}
	if !containsString(report.NextSteps, "billar setup") {
		t.Fatalf("NextSteps = %#v, want billar setup when required dir is missing", report.NextSteps)
	}
	if !commandHealthHas(report.CommandHealth, "backup_dir", "fail") || !commandHealthHas(report.CommandHealth, "db_parent_dir", "ok") {
		t.Fatalf("CommandHealth = %#v, want backup_dir failure and db_parent_dir ok", report.CommandHealth)
	}
}

func TestDoctorServiceReportsMissingDBParentWithoutStore(t *testing.T) {
	t.Parallel()

	runtimeRoot := filepath.Join(t.TempDir(), "missing-root")
	dbPath := filepath.Join(runtimeRoot, "data", "billar.db")
	exportDir := filepath.Join(runtimeRoot, "exports")
	backupDir := filepath.Join(runtimeRoot, "backups")
	svc := NewDoctorService(nil, DoctorConfig{
		Project:         "billar",
		DBPath:          dbPath,
		DBPathSource:    "configured",
		ExportDir:       exportDir,
		ExportDirSource: "default",
		BackupDir:       backupDir,
		BackupDirSource: "default",
	})

	report, err := svc.Report(context.Background())
	if err != nil {
		t.Fatalf("Report() error = %v, want read-only readiness report without opening SQLite", err)
	}
	if report.DBParentDirExists || report.DBParentDirWritable || !containsString(report.NextSteps, "billar setup") {
		t.Fatalf("Report() = %+v, want missing DB parent and setup next step", report)
	}
	if _, err := os.Stat(filepath.Dir(dbPath)); !os.IsNotExist(err) {
		t.Fatalf("db parent stat error = %v, want doctor to avoid creating DB parent", err)
	}
	if !commandHealthHas(report.CommandHealth, "db", "fail") || !commandHealthHas(report.CommandHealth, "db_parent_dir", "fail") {
		t.Fatalf("CommandHealth = %#v, want db and db_parent_dir failures", report.CommandHealth)
	}
}

func TestDoctorServiceReportsReadOnlyDBProbeWithoutStore(t *testing.T) {
	t.Parallel()

	runtimeRoot := t.TempDir()
	dbPath := filepath.Join(runtimeRoot, "data", "billar.db")
	exportDir := filepath.Join(runtimeRoot, "exports")
	backupDir := filepath.Join(runtimeRoot, "backups")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o700); err != nil {
		t.Fatalf("mkdir db parent: %v", err)
	}
	if err := os.WriteFile(dbPath, []byte("sqlite bytes supplied by infra probe test double"), 0o600); err != nil {
		t.Fatalf("write db marker: %v", err)
	}
	if err := os.Mkdir(exportDir, 0o700); err != nil {
		t.Fatalf("mkdir export dir: %v", err)
	}
	if err := os.Mkdir(backupDir, 0o700); err != nil {
		t.Fatalf("mkdir backup dir: %v", err)
	}
	probe := &doctorDBProbeStub{schemaVersion: 9}
	svc := NewDoctorService(nil, DoctorConfig{Project: "billar", DBPath: dbPath, ExportDir: exportDir, BackupDir: backupDir, PDFAvailable: true, DBProbe: probe})

	report, err := svc.Report(context.Background())
	if err != nil {
		t.Fatalf("Report() error = %v", err)
	}
	if !report.DBReachable || report.SchemaVersion != 9 || !commandHealthHas(report.CommandHealth, "db", "ok") {
		t.Fatalf("Report() = %+v, want read-only probe to make DB readiness healthy", report)
	}
	if got := probe.paths; len(got) != 1 || got[0] != dbPath {
		t.Fatalf("probe paths = %#v, want single db path %q", got, dbPath)
	}
}

func TestDoctorReportDTOTags(t *testing.T) {
	t.Parallel()

	fields := map[string]string{
		"Project":             "project",
		"DBPath":              "db_path",
		"DBPathSource":        "db_path_source",
		"DBPathExists":        "db_path_exists",
		"DBParentDir":         "db_parent_dir",
		"DBParentDirSource":   "db_parent_dir_source",
		"DBParentDirExists":   "db_parent_dir_exists",
		"DBParentDirWritable": "db_parent_dir_writable",
		"SchemaVersion":       "schema_version",
		"DBReachable":         "db_reachable",
		"ExportDir":           "export_dir",
		"ExportDirSource":     "export_dir_source",
		"ExportDirExists":     "export_dir_exists",
		"ExportDirSet":        "export_dir_set",
		"ExportDirWritable":   "export_dir_writable",
		"BackupDir":           "backup_dir",
		"BackupDirSource":     "backup_dir_source",
		"BackupDirExists":     "backup_dir_exists",
		"BackupDirWritable":   "backup_dir_writable",
		"PDFAvailable":        "pdf_available",
		"CommandHealth":       "command_health",
		"NextSteps":           "next_steps",
	}
	typ := reflect.TypeOf(DoctorReportDTO{})
	for _, fieldName := range []string{"MCPConfigured", "MCPTrustedWrites", "MCPListenAddr"} {
		if _, ok := typ.FieldByName(fieldName); ok {
			t.Fatalf("DoctorReportDTO still exposes removed MCP field %s", fieldName)
		}
	}
	for fieldName, tagName := range fields {
		field, ok := typ.FieldByName(fieldName)
		if !ok {
			t.Fatalf("DoctorReportDTO missing field %s", fieldName)
		}
		if got := field.Tag.Get("json"); got != tagName {
			t.Fatalf("%s json tag = %q, want %q", fieldName, got, tagName)
		}
		if got := field.Tag.Get("toon"); got != tagName {
			t.Fatalf("%s toon tag = %q, want %q", fieldName, got, tagName)
		}
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func commandHealthHas(values []DoctorCommandHealthDTO, name, status string) bool {
	for _, value := range values {
		if value.Name == name && value.Status == status {
			return true
		}
	}
	return false
}
