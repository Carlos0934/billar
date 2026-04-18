package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEnvFileParsesValues(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	content := []byte("# comment\nBILLAR_APP_NAME=env-parsed\nOAUTH_PROVIDER=openai\n")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	for _, key := range []string{"BILLAR_APP_NAME", "OAUTH_PROVIDER"} {
		t.Setenv(key, "")
	}

	if err := loadEnvFile(path); err != nil {
		t.Fatalf("loadEnvFile() error = %v", err)
	}

	if got := os.Getenv("BILLAR_APP_NAME"); got != "env-parsed" {
		t.Fatalf("BILLAR_APP_NAME = %q, want %q", got, "env-parsed")
	}
	if got := os.Getenv("OAUTH_PROVIDER"); got != "openai" {
		t.Fatalf("OAUTH_PROVIDER = %q, want %q", got, "openai")
	}
}

func TestLoadEnvFileMissingFileReturnsNoError(t *testing.T) {
	if err := loadEnvFile(filepath.Join(t.TempDir(), "missing.env")); err != nil {
		t.Fatalf("loadEnvFile() error = %v, want nil", err)
	}
}
