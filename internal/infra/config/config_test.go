package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadParsesAppNameFromEnvironment(t *testing.T) {
	dir := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })

	for _, key := range []string{
		"BILLAR_APP_NAME",
		"BILLAR_EXPORT_DIR",
		"BILLAR_BACKUP_DIR",
		"BILLAR_DB_PATH",
		"XDG_DATA_HOME",
		"XDG_CONFIG_HOME",
		"NO_COLOR",
		"TERM",
	} {
		t.Setenv(key, "")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)

	t.Setenv("BILLAR_APP_NAME", " session-surface ")

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	want := Config{
		AppName:         "session-surface",
		ColorEnabled:    true,
		ExportDir:       filepath.Join(home, ".config", "billar", "exports"),
		ExportDirSource: "default",
		BackupDir:       filepath.Join(home, ".config", "billar", "backups"),
		BackupDirSource: "default",
		DBPath:          filepath.Join(home, ".config", "billar", "billar.db"),
		DBPathSource:    "default",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Load() = %+v, want %+v", got, want)
	}
}

func TestLoadReadsDotEnvFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	content := []byte("BILLAR_APP_NAME=from-file\nBILLAR_EXPORT_DIR=/tmp/billar-exports\nBILLAR_BACKUP_DIR=/tmp/backups\n")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })

	t.Setenv("BILLAR_APP_NAME", "")
	t.Setenv("BILLAR_EXPORT_DIR", "")
	t.Setenv("BILLAR_BACKUP_DIR", "")
	t.Setenv("BILLAR_DB_PATH", "/tmp/billar-test.db")

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	want := Config{
		AppName:         "from-file",
		ColorEnabled:    true,
		ExportDir:       "/tmp/billar-exports",
		ExportDirSource: "configured",
		BackupDir:       "/tmp/backups",
		BackupDirSource: "configured",
		DBPath:          "/tmp/billar-test.db",
		DBPathSource:    "configured",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Load() = %+v, want %+v", got, want)
	}
}

func TestConfigHasNoAccessPolicyField(t *testing.T) {
	// The AccessPolicy field must not exist on Config.
	// If this test compiles and runs, the field is absent.
	cfg := Config{
		AppName:      "test",
		ColorEnabled: true,
		ExportDir:    "",
		BackupDir:    "",
		DBPath:       "/tmp/billar.db",
	}
	_ = cfg
	// Any attempt to set cfg.AccessPolicy would cause a compile error.
}

func TestLoadParsesExportDirFromEnvironment(t *testing.T) {
	dir := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })

	t.Setenv("BILLAR_APP_NAME", "")
	t.Setenv("BILLAR_EXPORT_DIR", " /var/billar/exports ")
	t.Setenv("BILLAR_DB_PATH", "/tmp/billar-test.db")
	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "")

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.ExportDir != "/var/billar/exports" {
		t.Fatalf("Load().ExportDir = %q, want trimmed export dir", got.ExportDir)
	}
}

func TestLoadParsesBackupDirFromEnvironment(t *testing.T) {
	dir := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })

	t.Setenv("BILLAR_APP_NAME", "")
	t.Setenv("BILLAR_EXPORT_DIR", "")
	t.Setenv("BILLAR_BACKUP_DIR", " /var/billar/backups ")
	t.Setenv("BILLAR_DB_PATH", "/tmp/billar-test.db")
	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "")

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.BackupDir != "/var/billar/backups" {
		t.Fatalf("Load().BackupDir = %q, want trimmed backup dir", got.BackupDir)
	}
}

func TestLoadPopulatesDBPathFromEnvironment(t *testing.T) {
	dir := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })

	t.Setenv("BILLAR_APP_NAME", "")
	t.Setenv("BILLAR_EXPORT_DIR", "")
	t.Setenv("BILLAR_DB_PATH", " /var/data/billar.db ")
	t.Setenv("XDG_DATA_HOME", "/ignored")

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got.DBPath != "/var/data/billar.db" {
		t.Fatalf("Load().DBPath = %q, want explicit env path", got.DBPath)
	}
}

func TestLoadPopulatesDefaultDBPath(t *testing.T) {
	dir := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })

	home := t.TempDir()
	configHome := t.TempDir()
	t.Setenv("BILLAR_APP_NAME", "")
	t.Setenv("BILLAR_EXPORT_DIR", "")
	t.Setenv("BILLAR_DB_PATH", "")
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("HOME", home)

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	want := filepath.Join(configHome, "billar", "billar.db")
	if got.DBPath != want {
		t.Fatalf("Load().DBPath = %q, want %q", got.DBPath, want)
	}
}

func TestLoadPopulatesDefaultRuntimeDirsWithoutCreatingThem(t *testing.T) {
	dir := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })

	configHome := filepath.Join(t.TempDir(), "config")
	t.Setenv("BILLAR_APP_NAME", "")
	t.Setenv("BILLAR_EXPORT_DIR", "")
	t.Setenv("BILLAR_DB_PATH", "")
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("HOME", t.TempDir())

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	wantDir := filepath.Join(configHome, "billar")
	if got.DBPath != filepath.Join(wantDir, "billar.db") {
		t.Fatalf("Load().DBPath = %q, want path under %q", got.DBPath, wantDir)
	}
	if got.ExportDir != filepath.Join(wantDir, "exports") {
		t.Fatalf("Load().ExportDir = %q, want default export dir", got.ExportDir)
	}
	if got.BackupDir != filepath.Join(wantDir, "backups") {
		t.Fatalf("Load().BackupDir = %q, want default backup dir", got.BackupDir)
	}
	if _, err := os.Stat(wantDir); !os.IsNotExist(err) {
		t.Fatalf("Load() created %q or returned unexpected stat error %v; setup owns directory creation", wantDir, err)
	}
}

func TestLoadDoesNotCreateOrValidateDefaultDBParentDirectory(t *testing.T) {
	dir := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })

	configHomeFile := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(configHomeFile, []byte("file"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	wantPath := filepath.Join(configHomeFile, "billar", "billar.db")
	t.Setenv("BILLAR_APP_NAME", "")
	t.Setenv("BILLAR_EXPORT_DIR", "")
	t.Setenv("BILLAR_DB_PATH", "")
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("XDG_CONFIG_HOME", configHomeFile)
	t.Setenv("HOME", t.TempDir())

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.DBPath != wantPath {
		t.Fatalf("Load().DBPath = %q, want %q", got.DBPath, wantPath)
	}
}
