package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEnvFileParsesValues(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	content := []byte("# comment\nALLOWED_EMAILS=user@example.com, admin@example.com\n\nALLOWED_DOMAINS='company.com'\nALLOWED_IPS=\"127.0.0.1, 10.0.0.1\"\nOAUTH_PROVIDER=openai\n")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	for _, key := range []string{"ALLOWED_EMAILS", "ALLOWED_DOMAINS", "ALLOWED_IPS", "OAUTH_PROVIDER"} {
		t.Setenv(key, "")
	}

	if err := loadEnvFile(path); err != nil {
		t.Fatalf("loadEnvFile() error = %v", err)
	}

	if got := os.Getenv("ALLOWED_EMAILS"); got != "user@example.com, admin@example.com" {
		t.Fatalf("ALLOWED_EMAILS = %q, want %q", got, "user@example.com, admin@example.com")
	}
	if got := os.Getenv("ALLOWED_DOMAINS"); got != "company.com" {
		t.Fatalf("ALLOWED_DOMAINS = %q, want %q", got, "company.com")
	}
	if got := os.Getenv("ALLOWED_IPS"); got != "127.0.0.1, 10.0.0.1" {
		t.Fatalf("ALLOWED_IPS = %q, want %q", got, "127.0.0.1, 10.0.0.1")
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
