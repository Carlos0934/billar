package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetupServiceRunCreatesRuntimeDirsAndIsIdempotent(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, "data", "billar.db")
	exportDir := filepath.Join(root, "exports")
	backupDir := filepath.Join(root, "backups")
	envPath := filepath.Join(root, ".env")

	svc := NewSetupService("billar", RuntimePaths{
		DBPath:    RuntimePath{Path: dbPath, Source: "configured"},
		ExportDir: RuntimePath{Path: exportDir, Source: "configured"},
		BackupDir: RuntimePath{Path: backupDir, Source: "default"},
	})

	first, err := svc.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	wantCreated := []string{filepath.Dir(dbPath), exportDir, backupDir}
	assertSameStrings(t, first.Created, wantCreated)
	if len(first.AlreadyExisted) != 0 {
		t.Fatalf("AlreadyExisted = %v, want empty", first.AlreadyExisted)
	}
	for _, dir := range wantCreated {
		assertDirMode(t, dir, 0o700)
	}
	if _, err := os.Stat(envPath); !os.IsNotExist(err) {
		t.Fatalf("setup touched .env at %q: %v", envPath, err)
	}

	second, err := svc.Run(context.Background())
	if err != nil {
		t.Fatalf("second Run() error = %v", err)
	}
	assertSameStrings(t, second.AlreadyExisted, wantCreated)
	if len(second.Created) != 0 {
		t.Fatalf("second Created = %v, want empty", second.Created)
	}
}

func TestSetupServiceRunReportsPathsNextStepsAndWarningsWithoutSecrets(t *testing.T) {
	root := t.TempDir()
	report, err := NewSetupService(" billar ", RuntimePaths{
		DBPath:    RuntimePath{Path: filepath.Join(root, "custom", "db.sqlite"), Source: "configured"},
		ExportDir: RuntimePath{Path: filepath.Join(root, "custom", "exports"), Source: "configured"},
		BackupDir: RuntimePath{Path: filepath.Join(root, "custom", "backups"), Source: "configured"},
	}).Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if report.Project != "billar" || report.DBPath != filepath.Join(root, "custom", "db.sqlite") || report.ExportDir == "" || report.BackupDir == "" {
		t.Fatalf("report identity/paths = %+v, want configured paths", report)
	}
	assertContains(t, report.NextSteps, "billar issuer create")
	assertContains(t, report.NextSteps, "billar customer create")
	assertContains(t, report.NextSteps, "billar doctor")
	if len(report.Warnings) != 1 || !strings.Contains(strings.ToLower(report.Warnings[0]), "sensitive") {
		t.Fatalf("Warnings = %v, want sensitive-data warning", report.Warnings)
	}
	rendered := fmt.Sprintf("%+v", report)
	for _, forbidden := range []string{"super-secret-token", "api_key", "MCP_API_KEYS", ".env"} {
		if strings.Contains(strings.ToLower(rendered), strings.ToLower(forbidden)) {
			t.Fatalf("SetupReportDTO leaked forbidden token %q in %q", forbidden, rendered)
		}
	}
}

func assertDirMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%q) error = %v", path, err)
	}
	if !info.IsDir() {
		t.Fatalf("%q is not a directory", path)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("%q mode = %o, want %o", path, got, want)
	}
}

func assertSameStrings(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("slice length = %d (%v), want %d (%v)", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("slice[%d] = %q, want %q (got %v want %v)", i, got[i], want[i], got, want)
		}
	}
}

func assertContains(t *testing.T, items []string, want string) {
	t.Helper()
	for _, item := range items {
		if item == want {
			return
		}
	}
	t.Fatalf("%v does not contain %q", items, want)
}
