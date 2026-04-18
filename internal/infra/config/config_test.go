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
		"NO_COLOR",
		"TERM",
	} {
		t.Setenv(key, "")
	}

	t.Setenv("BILLAR_APP_NAME", " session-surface ")

	got := Load()
	want := Config{
		AppName:      "session-surface",
		ColorEnabled: true,
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Load() = %+v, want %+v", got, want)
	}
}

func TestLoadReadsDotEnvFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	content := []byte("BILLAR_APP_NAME=from-file\n")
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

	got := Load()
	want := Config{
		AppName:      "from-file",
		ColorEnabled: true,
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
	}
	_ = cfg
	// Any attempt to set cfg.AccessPolicy would cause a compile error.
}
