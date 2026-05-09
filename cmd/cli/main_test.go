package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Carlos0934/billar/internal/app"
	"github.com/Carlos0934/billar/internal/infra/backup"
	"github.com/Carlos0934/billar/internal/infra/config"
	infrasqlite "github.com/Carlos0934/billar/internal/infra/sqlite"
)

func TestNewCommandWiresTimeEntryService(t *testing.T) {
	t.Parallel()

	store := mustOpenCLIStore(t)
	seedCLIWiringFixture(t, store.DB())

	cmd := newCommand(config.Config{AppName: "billar", ColorEnabled: false}, store)

	var out bytes.Buffer
	if err := cmd.Run(context.Background(), []string{"time-entry", "record", `--json={"customer_profile_id":"cus_cli_wiring","service_agreement_id":"sa_cli_wiring","description":"wiring check","hours":60,"billable":true,"date":"2026-04-10T00:00:00Z"}`, "--format", "json"}, &out); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !strings.Contains(out.String(), "wiring check") {
		t.Fatalf("Run() output = %q, want wiring check payload", out.String())
	}

	timeEntryStore := infrasqlite.NewTimeEntryStore(store)
	entries, err := timeEntryStore.ListByCustomerProfile(context.Background(), "cus_cli_wiring")
	if err != nil {
		t.Fatalf("ListByCustomerProfile() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("ListByCustomerProfile() len = %d, want 1", len(entries))
	}
	got := entries[0]
	if got.CustomerProfileID != "cus_cli_wiring" || got.ServiceAgreementID != "sa_cli_wiring" {
		t.Fatalf("ListByCustomerProfile() = %+v, want seeded wiring entry", got)
	}
	if got.Description != "wiring check" {
		t.Fatalf("ListByCustomerProfile() description = %q, want %q", got.Description, "wiring check")
	}
}

func TestMainRunsHealthCommand(t *testing.T) {
	storePath := t.TempDir() + "/cli-main.db"
	t.Setenv("BILLAR_DB_PATH", storePath)

	oldArgs := os.Args
	os.Args = []string{"billar", "health"}
	t.Cleanup(func() { os.Args = oldArgs })

	main()
}

func TestOpenConfiguredStoreReportsDBPathAndOverrideHint(t *testing.T) {
	parentFile := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(parentFile, []byte("file"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	dbPath := filepath.Join(parentFile, "billar.db")

	store, err := openConfiguredStore(config.Config{DBPath: dbPath})
	if err == nil {
		if store != nil {
			_ = store.Close()
		}
		t.Fatal("openConfiguredStore() error = nil, want directory creation error")
	}
	if !strings.Contains(err.Error(), dbPath) {
		t.Fatalf("openConfiguredStore() error = %q, want attempted path %q", err.Error(), dbPath)
	}
	if !strings.Contains(err.Error(), "BILLAR_DB_PATH") {
		t.Fatalf("openConfiguredStore() error = %q, want BILLAR_DB_PATH hint", err.Error())
	}
}

func TestCommandNeedsStoreAllowsSetupAndHelpBeforeDBOpen(t *testing.T) {
	t.Parallel()

	for _, args := range [][]string{{"setup"}, {"setup", "--format", "json"}, {"doctor"}, {"doctor", "--format", "json"}, {"backup", "list"}, {"backup", "create"}, {"--help"}, {"help"}} {
		if commandNeedsStore(args) {
			t.Fatalf("commandNeedsStore(%v) = true, want false for pre-store command", args)
		}
	}
	for _, args := range [][]string{{"invoice", "list"}, {"health"}} {
		if !commandNeedsStore(args) {
			t.Fatalf("commandNeedsStore(%v) = false, want true", args)
		}
	}
}

func TestNewPreStoreCommandRunsSetupWithoutDoctorStore(t *testing.T) {
	t.Parallel()

	runtimeRoot := t.TempDir()
	cfg := config.Config{AppName: "billar", ColorEnabled: false, DBPath: filepath.Join(runtimeRoot, "data", "billar.db"), DBPathSource: "configured", ExportDir: filepath.Join(runtimeRoot, "exports"), ExportDirSource: "configured", BackupDir: filepath.Join(runtimeRoot, "backups"), BackupDirSource: "configured"}
	cmd := newPreStoreCommand(cfg)
	var out bytes.Buffer
	if err := cmd.Run(context.Background(), []string{"setup", "--format", "json"}, &out); err != nil {
		t.Fatalf("setup Run() error = %v", err)
	}
	if !strings.Contains(out.String(), cfg.BackupDir) {
		t.Fatalf("setup output = %q, want backup dir", out.String())
	}
}

func TestNewPreStoreCommandRunsBackupRestoreWithoutOpeningStore(t *testing.T) {
	runtimeRoot := t.TempDir()
	cfg := config.Config{AppName: "billar", ColorEnabled: false, DBPath: filepath.Join(runtimeRoot, "data", "billar.db"), DBPathSource: "configured", ExportDir: filepath.Join(runtimeRoot, "exports"), ExportDirSource: "configured", BackupDir: filepath.Join(runtimeRoot, "backups"), BackupDirSource: "configured"}
	if err := os.MkdirAll(filepath.Dir(cfg.DBPath), 0o700); err != nil {
		t.Fatalf("mkdir db dir: %v", err)
	}
	store, err := infrasqlite.Open(cfg.DBPath)
	if err != nil {
		t.Fatalf("seed store: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close seed store: %v", err)
	}
	record, err := backup.Snapshotter{Now: func() time.Time { return time.Date(2026, 5, 1, 18, 0, 0, 0, time.UTC) }}.Create(context.Background(), cfg.DBPath, cfg.BackupDir)
	if err != nil {
		t.Fatalf("create backup fixture: %v", err)
	}
	if err := os.Remove(cfg.DBPath); err != nil {
		t.Fatalf("remove live db: %v", err)
	}

	cmd := newPreStoreCommand(cfg)
	var dryRunOut bytes.Buffer
	if err := cmd.Run(context.Background(), []string{"backup", "restore", "--id", record.ID, "--dry-run", "--format", "json"}, &dryRunOut); err != nil {
		t.Fatalf("backup restore dry-run Run() error = %v, output = %q", err, dryRunOut.String())
	}
	if _, statErr := os.Stat(cfg.DBPath); !os.IsNotExist(statErr) {
		t.Fatalf("db path stat after dry-run = %v, want no store open or restore mutation", statErr)
	}

	var restoreOut bytes.Buffer
	if err := cmd.Run(context.Background(), []string{"backup", "restore", "--id", record.ID, "--force", "--format", "json"}, &restoreOut); err != nil {
		t.Fatalf("backup restore force Run() error = %v, output = %q", err, restoreOut.String())
	}
	if _, err := os.Stat(cfg.DBPath); err != nil {
		t.Fatalf("db path after destructive restore: %v", err)
	}
}

func TestNewPreStoreCommandRunsDoctorReadinessWithoutCreatingDBParent(t *testing.T) {
	t.Parallel()

	runtimeRoot := filepath.Join(t.TempDir(), "missing-runtime")
	cfg := config.Config{AppName: "billar", ColorEnabled: false, DBPath: filepath.Join(runtimeRoot, "data", "billar.db"), DBPathSource: "configured", ExportDir: filepath.Join(runtimeRoot, "exports"), ExportDirSource: "configured", BackupDir: filepath.Join(runtimeRoot, "backups"), BackupDirSource: "configured"}
	cmd := newPreStoreCommand(cfg)

	var out bytes.Buffer
	err := cmd.Run(context.Background(), []string{"doctor", "--format", "json"}, &out)
	if err == nil || !strings.Contains(err.Error(), "doctor readiness failed") {
		t.Fatalf("doctor Run() error = %v, want readiness failure", err)
	}
	if _, statErr := os.Stat(filepath.Dir(cfg.DBPath)); !os.IsNotExist(statErr) {
		t.Fatalf("db parent stat error = %v, want doctor to remain read-only", statErr)
	}
	if _, statErr := os.Stat(cfg.DBPath); !os.IsNotExist(statErr) {
		t.Fatalf("db path stat error = %v, want doctor not to create or migrate SQLite", statErr)
	}
	if !strings.Contains(out.String(), "billar setup") || !strings.Contains(out.String(), "db_parent_dir") {
		t.Fatalf("doctor output = %q, want setup guidance and DB parent readiness", out.String())
	}
}

func TestNewPreStoreCommandRunsDoctorReadinessForInitializedDB(t *testing.T) {
	t.Parallel()

	runtimeRoot := t.TempDir()
	cfg := config.Config{AppName: "billar", ColorEnabled: false, DBPath: filepath.Join(runtimeRoot, "data", "billar.db"), DBPathSource: "configured", ExportDir: filepath.Join(runtimeRoot, "exports"), ExportDirSource: "configured", BackupDir: filepath.Join(runtimeRoot, "backups"), BackupDirSource: "configured"}
	if err := os.MkdirAll(cfg.ExportDir, 0o700); err != nil {
		t.Fatalf("mkdir export dir: %v", err)
	}
	if err := os.MkdirAll(cfg.BackupDir, 0o700); err != nil {
		t.Fatalf("mkdir backup dir: %v", err)
	}
	store, err := infrasqlite.Open(cfg.DBPath)
	if err != nil {
		t.Fatalf("initialize sqlite store: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close initialized sqlite store: %v", err)
	}
	before, err := os.Stat(cfg.DBPath)
	if err != nil {
		t.Fatalf("stat initialized db: %v", err)
	}

	cmd := newPreStoreCommand(cfg)
	var out bytes.Buffer
	if err := cmd.Run(context.Background(), []string{"doctor", "--format", "json"}, &out); err != nil {
		t.Fatalf("doctor Run() error = %v, output = %q; want healthy initialized DB readiness", err, out.String())
	}
	var report struct {
		DBReachable   bool `json:"db_reachable"`
		CommandHealth []struct {
			Name   string `json:"name"`
			Status string `json:"status"`
		} `json:"command_health"`
	}
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal doctor output: %v; output = %q", err, out.String())
	}
	if !report.DBReachable || !cliCommandHealthHas(report.CommandHealth, "db", "ok") {
		t.Fatalf("doctor report = %+v, want reachable DB and ok db command health", report)
	}
	after, err := os.Stat(cfg.DBPath)
	if err != nil {
		t.Fatalf("stat db after doctor: %v", err)
	}
	if !after.ModTime().Equal(before.ModTime()) || after.Size() != before.Size() {
		t.Fatalf("db changed after doctor: before(size=%d, mod=%s) after(size=%d, mod=%s), want read-only probe", before.Size(), before.ModTime(), after.Size(), after.ModTime())
	}
}

func cliCommandHealthHas(values []struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}, name, status string) bool {
	for _, value := range values {
		if value.Name == name && value.Status == status {
			return true
		}
	}
	return false
}

func mustOpenCLIStore(t *testing.T) *infrasqlite.Store {
	t.Helper()

	store, err := infrasqlite.Open(t.TempDir() + "/cli-entrypoint.db")
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	})
	return store
}

func TestNewCommandWiresInvoiceService(t *testing.T) {
	t.Parallel()

	store := mustOpenCLIStore(t)
	seedCLIWiringFixture(t, store.DB())
	cmd := newCommand(config.Config{AppName: "billar", ColorEnabled: false}, store)

	// First record a time entry via CLI so it has unbilled data.
	var recOut bytes.Buffer
	if err := cmd.Run(context.Background(), []string{"time-entry", "record", `--json={"customer_profile_id":"cus_cli_wiring","service_agreement_id":"sa_cli_wiring","description":"invoice wiring","hours":60,"billable":true,"date":"2026-04-10T00:00:00Z"}`, "--format", "json"}, &recOut); err != nil {
		t.Fatalf("time-entry record Run() error = %v", err)
	}

	var out bytes.Buffer
	// Running "invoice draft" proves the invoice service is wired —
	// without wiring, the command would return "invoice service is required".
	err := cmd.Run(context.Background(), []string{"invoice", "draft", "--customer-id", "cus_cli_wiring", "--format", "json"}, &out)
	if err != nil {
		t.Fatalf("invoice draft Run() error = %v", err)
	}
	if !strings.Contains(out.String(), "cus_cli_wiring") {
		t.Fatalf("invoice draft output = %q, want customer ID payload", out.String())
	}
}

func TestNewCommandWiresQuoteService(t *testing.T) {
	t.Parallel()

	store := mustOpenCLIStore(t)
	seedCLIWiringFixture(t, store.DB())
	cmd := newCommand(config.Config{AppName: "billar", ColorEnabled: false}, store)

	var out bytes.Buffer
	err := cmd.Run(context.Background(), []string{"quote", "create", "--customer-id", "cus_cli_wiring", "--currency", "USD", "--format", "json"}, &out)
	if err != nil {
		t.Fatalf("quote create Run() error = %v", err)
	}
	if !strings.Contains(out.String(), "cus_cli_wiring") || !strings.Contains(out.String(), "draft") {
		t.Fatalf("quote create output = %q, want quote payload", out.String())
	}
}

func TestNewCommandWiresQuotePDFService(t *testing.T) {
	t.Parallel()

	store := mustOpenCLIStore(t)
	seedCLIWiringFixture(t, store.DB())
	exportDir := t.TempDir()
	cmd := newCommand(config.Config{AppName: "billar", ColorEnabled: false, ExportDir: exportDir}, store)

	var createOut bytes.Buffer
	if err := cmd.Run(context.Background(), []string{"quote", "create", "--customer-id", "cus_cli_wiring", "--currency", "USD", "--format", "json"}, &createOut); err != nil {
		t.Fatalf("quote create Run() error = %v", err)
	}
	var quote app.QuoteDTO
	if err := json.Unmarshal(createOut.Bytes(), &quote); err != nil {
		t.Fatalf("quote create output invalid: %v", err)
	}

	var lineOut bytes.Buffer
	if err := cmd.Run(context.Background(), []string{"quote", "add-line", quote.ID, "--agreement-id", "sa_cli_wiring", "--description", "Proposal wiring", "--minutes", "60", "--format", "json"}, &lineOut); err != nil {
		t.Fatalf("quote add-line Run() error = %v", err)
	}

	outPath := filepath.Join(exportDir, "proposal.pdf")
	var pdfOut bytes.Buffer
	if err := cmd.Run(context.Background(), []string{"quote", "pdf", quote.ID, "--out", outPath, "--format", "json"}, &pdfOut); err != nil {
		t.Fatalf("quote pdf Run() error = %v", err)
	}
	var exported app.QuoteRenderedFileDTO
	if err := json.Unmarshal(pdfOut.Bytes(), &exported); err != nil {
		t.Fatalf("quote pdf output invalid: %v", err)
	}
	if exported.QuoteID != quote.ID || exported.Path != outPath || exported.MimeType != "application/pdf" || exported.SizeBytes == 0 {
		t.Fatalf("quote pdf output = %+v, want wired export metadata", exported)
	}
}

func TestNewCommandWiresSetupBackupAndDoctorReadiness(t *testing.T) {
	t.Parallel()

	store := mustOpenCLIStore(t)
	runtimeRoot := t.TempDir()
	cfg := config.Config{
		AppName:         "billar",
		ColorEnabled:    false,
		DBPath:          filepath.Join(runtimeRoot, "data", "billar.db"),
		DBPathSource:    "configured",
		ExportDir:       filepath.Join(runtimeRoot, "exports"),
		ExportDirSource: "configured",
		BackupDir:       filepath.Join(runtimeRoot, "backups"),
		BackupDirSource: "configured",
	}
	cmd := newCommand(cfg, store)

	var setupOut bytes.Buffer
	if err := cmd.Run(context.Background(), []string{"setup", "--format", "json"}, &setupOut); err != nil {
		t.Fatalf("setup Run() error = %v", err)
	}
	if !strings.Contains(setupOut.String(), cfg.BackupDir) {
		t.Fatalf("setup output = %q, want wired backup dir", setupOut.String())
	}

	var doctorOut bytes.Buffer
	if err := cmd.Run(context.Background(), []string{"doctor", "--format", "json"}, &doctorOut); err != nil {
		t.Fatalf("doctor Run() error = %v", err)
	}
	if !strings.Contains(doctorOut.String(), "backup_dir_writable") || !strings.Contains(doctorOut.String(), "configured") {
		t.Fatalf("doctor output = %q, want readiness fields and sources", doctorOut.String())
	}

	var backupOut bytes.Buffer
	if err := cmd.Run(context.Background(), []string{"backup", "list", "--format", "json"}, &backupOut); err != nil {
		t.Fatalf("backup list Run() error = %v", err)
	}
	if !strings.Contains(backupOut.String(), cfg.BackupDir) {
		t.Fatalf("backup list output = %q, want wired backup dir", backupOut.String())
	}
}

func seedCLIWiringFixture(t *testing.T, db *sql.DB) {
	t.Helper()

	now := time.Date(2026, 4, 10, 10, 0, 0, 0, time.UTC).UnixNano()
	if _, err := db.ExecContext(context.Background(), `
INSERT INTO legal_entities (id, type, legal_name, trade_name, tax_id, email, phone, website, billing_address, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"le_cli_wiring", "company", "CLI Wiring Co", "", "", "", "", "", "{}", now, now); err != nil {
		t.Fatalf("insert legal entity: %v", err)
	}
	if _, err := db.ExecContext(context.Background(), `
INSERT INTO legal_entities (id, type, legal_name, trade_name, tax_id, email, phone, website, billing_address, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"le_cli_issuer", "company", "CLI Issuer Co", "", "", "", "", "", "{}", now, now); err != nil {
		t.Fatalf("insert issuer legal entity: %v", err)
	}
	if _, err := db.ExecContext(context.Background(), `
INSERT INTO issuer_profiles (id, legal_entity_id, default_currency, default_notes, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?)`,
		"iss_cli_wiring", "le_cli_issuer", "USD", "", now, now); err != nil {
		t.Fatalf("insert issuer profile: %v", err)
	}
	if _, err := db.ExecContext(context.Background(), `
INSERT INTO customer_profiles (id, legal_entity_id, status, default_currency, notes, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"cus_cli_wiring", "le_cli_wiring", "active", "USD", "", now, now); err != nil {
		t.Fatalf("insert customer profile: %v", err)
	}
	if _, err := db.ExecContext(context.Background(), `
INSERT INTO service_agreements (id, customer_profile_id, name, description, billing_mode, hourly_rate, currency, active, valid_from, valid_until, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"sa_cli_wiring", "cus_cli_wiring", "CLI Wiring Agreement", "", "hourly", 5000, "USD", 1, nil, nil, now, now); err != nil {
		t.Fatalf("insert service agreement: %v", err)
	}
}
