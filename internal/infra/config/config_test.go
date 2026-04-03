package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadParsesAccessPolicyFromEnvironment(t *testing.T) {
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
		"ALLOWED_EMAILS",
		"ALLOWED_DOMAINS",
		"ALLOWED_IPS",
		"OAUTH_PROVIDER",
	} {
		t.Setenv(key, "")
	}

	t.Setenv("BILLAR_APP_NAME", " session-surface ")
	t.Setenv("ALLOWED_EMAILS", " user@example.com , admin@example.com ")
	t.Setenv("ALLOWED_DOMAINS", " company.com , example.org ")
	t.Setenv("ALLOWED_IPS", " 127.0.0.1 , 10.0.0.1 ")
	t.Setenv("OAUTH_PROVIDER", " openai ")

	got := Load()
	want := Config{
		AppName:      "session-surface",
		ColorEnabled: true,
		AccessPolicy: AccessPolicy{
			AllowedEmails:  []string{"user@example.com", "admin@example.com"},
			AllowedDomains: []string{"company.com", "example.org"},
			AllowedIPs:     []string{"127.0.0.1", "10.0.0.1"},
			OAuthProvider:  "openai",
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Load() = %+v, want %+v", got, want)
	}
}

func TestLoadReadsDotEnvFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	content := []byte("BILLAR_APP_NAME=from-file\nALLOWED_EMAILS=file@example.com\nALLOWED_DOMAINS=file.example.com\nALLOWED_IPS=10.0.0.10\nOAUTH_PROVIDER=file-provider\n")
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

	for _, key := range []string{"BILLAR_APP_NAME", "ALLOWED_EMAILS", "ALLOWED_DOMAINS", "ALLOWED_IPS", "OAUTH_PROVIDER"} {
		t.Setenv(key, "")
	}

	got := Load()
	want := Config{
		AppName:      "from-file",
		ColorEnabled: true,
		AccessPolicy: AccessPolicy{
			AllowedEmails:  []string{"file@example.com"},
			AllowedDomains: []string{"file.example.com"},
			AllowedIPs:     []string{"10.0.0.10"},
			OAuthProvider:  "file-provider",
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Load() = %+v, want %+v", got, want)
	}
}
