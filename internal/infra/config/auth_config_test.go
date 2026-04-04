package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoadAuthConfig(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		want    AuthConfig
		wantErr string
	}{
		{
			name: "parses values and trims whitespace",
			env: map[string]string{
				"OAUTH_CLIENT_ID":          " client-id ",
				"OAUTH_ISSUER_URL":         " https://issuer.example ",
				"MCP_HTTP_LISTEN_ADDR":     " 127.0.0.1:8080 ",
				"AUTH_ALLOWED_EMAILS":      " admin@example.com , user@example.com ",
				"AUTH_ALLOWED_DOMAINS":     " allowed.com , company.com ",
				"AUTH_RESOURCE_SERVER_URI": " https://resource.example ",
			},
			want: AuthConfig{
				ClientID:          "client-id",
				IssuerURL:         "https://issuer.example",
				ListenAddr:        "127.0.0.1:8080",
				AllowedEmails:     []string{"admin@example.com", "user@example.com"},
				AllowedDomains:    []string{"allowed.com", "company.com"},
				ResourceServerURI: "https://resource.example",
			},
		},
		{
			name: "defaults missing issuer and listen values",
			env: map[string]string{
				"OAUTH_CLIENT_ID":          "client-id",
				"OAUTH_ISSUER_URL":         "",
				"MCP_HTTP_LISTEN_ADDR":     "",
				"AUTH_ALLOWED_EMAILS":      "admin@example.com",
				"AUTH_ALLOWED_DOMAINS":     "",
				"AUTH_RESOURCE_SERVER_URI": "https://resource.example",
			},
			want: AuthConfig{
				ClientID:          "client-id",
				IssuerURL:         "https://accounts.google.com",
				ListenAddr:        "127.0.0.1:8080",
				AllowedEmails:     []string{"admin@example.com"},
				AllowedDomains:    []string{},
				ResourceServerURI: "https://resource.example",
			},
		},
		{
			name: "requires oidc client id",
			env: map[string]string{
				"OAUTH_CLIENT_ID":          "",
				"OAUTH_ISSUER_URL":         "https://issuer.example",
				"MCP_HTTP_LISTEN_ADDR":     "127.0.0.1:8080",
				"AUTH_ALLOWED_EMAILS":      "admin@example.com",
				"AUTH_ALLOWED_DOMAINS":     "",
				"AUTH_RESOURCE_SERVER_URI": "https://resource.example",
			},
			wantErr: "OAUTH_CLIENT_ID",
		},
		{
			name: "rejects empty policy after trimming",
			env: map[string]string{
				"OAUTH_CLIENT_ID":          "client-id",
				"OAUTH_ISSUER_URL":         "https://issuer.example",
				"MCP_HTTP_LISTEN_ADDR":     "127.0.0.1:8080",
				"AUTH_ALLOWED_EMAILS":      "   ",
				"AUTH_ALLOWED_DOMAINS":     "",
				"AUTH_RESOURCE_SERVER_URI": "https://resource.example",
			},
			wantErr: "access policy",
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			for _, key := range []string{
				"OAUTH_CLIENT_ID",
				"OAUTH_ISSUER_URL",
				"MCP_HTTP_LISTEN_ADDR",
				"AUTH_ALLOWED_EMAILS",
				"AUTH_ALLOWED_DOMAINS",
				"AUTH_RESOURCE_SERVER_URI",
			} {
				t.Setenv(key, tc.env[key])
			}

			got, err := LoadAuthConfig()
			if tc.wantErr != "" {
				if err == nil {
					t.Fatal("LoadAuthConfig() error = nil, want non-nil")
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("LoadAuthConfig() error = %q, want substring %q", err.Error(), tc.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("LoadAuthConfig() error = %v", err)
			}

			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("LoadAuthConfig() = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestLoadAuthConfigLoadsDotEnv(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}

	workdir := t.TempDir()
	content := []byte("OAUTH_CLIENT_ID=from-file\nOAUTH_ISSUER_URL=https://issuer.from.file\nMCP_HTTP_LISTEN_ADDR=127.0.0.1:8080\nAUTH_ALLOWED_EMAILS= file-user@example.com \nAUTH_ALLOWED_DOMAINS=file.example.com\nAUTH_RESOURCE_SERVER_URI=https://resource.from.file\n")
	if err := os.WriteFile(filepath.Join(workdir, ".env"), content, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	for _, key := range []string{
		"OAUTH_CLIENT_ID",
		"OAUTH_ISSUER_URL",
		"MCP_HTTP_LISTEN_ADDR",
		"AUTH_ALLOWED_EMAILS",
		"AUTH_ALLOWED_DOMAINS",
		"AUTH_RESOURCE_SERVER_URI",
	} {
		t.Setenv(key, "")
	}

	if err := os.Chdir(workdir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldwd)
	})

	got, err := LoadAuthConfig()
	if err != nil {
		t.Fatalf("LoadAuthConfig() error = %v", err)
	}

	want := AuthConfig{
		ClientID:          "from-file",
		IssuerURL:         "https://issuer.from.file",
		ListenAddr:        "127.0.0.1:8080",
		AllowedEmails:     []string{"file-user@example.com"},
		AllowedDomains:    []string{"file.example.com"},
		ResourceServerURI: "https://resource.from.file",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LoadAuthConfig() = %+v, want %+v", got, want)
	}
}

func TestLoadAuthConfigPreservesExplicitEnvAndTrimsQuotedDotEnvValues(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}

	workdir := t.TempDir()
	content := []byte("OAUTH_CLIENT_ID='from-file'\nOAUTH_ISSUER_URL='https://issuer.from.file'\nMCP_HTTP_LISTEN_ADDR='127.0.0.1:8080'\nAUTH_ALLOWED_EMAILS=' file-user@example.com '\nAUTH_ALLOWED_DOMAINS=\"file.example.com\"\nAUTH_RESOURCE_SERVER_URI='https://resource.from.file'\n")
	if err := os.WriteFile(filepath.Join(workdir, ".env"), content, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	t.Setenv("OAUTH_CLIENT_ID", "preset-client-id")
	t.Setenv("OAUTH_ISSUER_URL", "")
	t.Setenv("MCP_HTTP_LISTEN_ADDR", "")
	t.Setenv("AUTH_ALLOWED_EMAILS", "")
	t.Setenv("AUTH_ALLOWED_DOMAINS", "")
	t.Setenv("AUTH_RESOURCE_SERVER_URI", "")

	if err := os.Chdir(workdir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldwd)
	})

	got, err := LoadAuthConfig()
	if err != nil {
		t.Fatalf("LoadAuthConfig() error = %v", err)
	}

	want := AuthConfig{
		ClientID:          "preset-client-id",
		IssuerURL:         "https://issuer.from.file",
		ListenAddr:        "127.0.0.1:8080",
		AllowedEmails:     []string{"file-user@example.com"},
		AllowedDomains:    []string{"file.example.com"},
		ResourceServerURI: "https://resource.from.file",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LoadAuthConfig() = %+v, want %+v", got, want)
	}
}

func TestEnvExampleDocumentsAuthSetup(t *testing.T) {
	path := filepath.Clean(filepath.Join("..", "..", "..", ".env.example"))
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}

	text := string(content)
	for _, want := range []string{
		"OAUTH_CLIENT_ID=",
		"OAUTH_ISSUER_URL=",
		"MCP_HTTP_LISTEN_ADDR=",
		"AUTH_RESOURCE_SERVER_URI=",
		"AUTH_ALLOWED_EMAILS=",
		"AUTH_ALLOWED_DOMAINS=",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf(".env.example missing %q", want)
		}
	}
}
