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
				"BILLAR_OAUTH_CLIENT_ID":          " client-id ",
				"BILLAR_OAUTH_CLIENT_SECRET":      " secret-value ",
				"BILLAR_OAUTH_ISSUER_URL":         " https://issuer.example ",
				"BILLAR_AUTH_ALLOWED_EMAILS":      " admin@example.com , user@example.com ",
				"BILLAR_AUTH_ALLOWED_DOMAINS":     " allowed.com , company.com ",
				"BILLAR_AUTH_RESOURCE_SERVER_URI": " https://resource.example ",
			},
			want: AuthConfig{
				ClientID:          "client-id",
				ClientSecret:      "secret-value",
				IssuerURL:         "https://issuer.example",
				AllowedEmails:     []string{"admin@example.com", "user@example.com"},
				AllowedDomains:    []string{"allowed.com", "company.com"},
				ResourceServerURI: "https://resource.example",
			},
		},
		{
			name: "defaults missing oauth strings to empty values",
			env: map[string]string{
				"BILLAR_OAUTH_CLIENT_ID":          "",
				"BILLAR_OAUTH_CLIENT_SECRET":      "",
				"BILLAR_OAUTH_ISSUER_URL":         "",
				"BILLAR_AUTH_ALLOWED_EMAILS":      "admin@example.com",
				"BILLAR_AUTH_ALLOWED_DOMAINS":     "",
				"BILLAR_AUTH_RESOURCE_SERVER_URI": "https://resource.example",
			},
			want: AuthConfig{
				AllowedEmails:     []string{"admin@example.com"},
				AllowedDomains:    []string{},
				ResourceServerURI: "https://resource.example",
			},
		},
		{
			name: "rejects empty policy after trimming",
			env: map[string]string{
				"BILLAR_OAUTH_CLIENT_ID":          "client-id",
				"BILLAR_OAUTH_CLIENT_SECRET":      "secret-value",
				"BILLAR_OAUTH_ISSUER_URL":         "https://issuer.example",
				"BILLAR_AUTH_ALLOWED_EMAILS":      "   ",
				"BILLAR_AUTH_ALLOWED_DOMAINS":     "",
				"BILLAR_AUTH_RESOURCE_SERVER_URI": "https://resource.example",
			},
			wantErr: "access policy",
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			for _, key := range []string{
				"BILLAR_OAUTH_CLIENT_ID",
				"BILLAR_OAUTH_CLIENT_SECRET",
				"BILLAR_OAUTH_ISSUER_URL",
				"BILLAR_AUTH_ALLOWED_EMAILS",
				"BILLAR_AUTH_ALLOWED_DOMAINS",
				"BILLAR_AUTH_RESOURCE_SERVER_URI",
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
	content := []byte("BILLAR_OAUTH_CLIENT_ID=from-file\nBILLAR_OAUTH_CLIENT_SECRET=from-file-secret\nBILLAR_OAUTH_ISSUER_URL=https://issuer.from.file\nBILLAR_AUTH_ALLOWED_EMAILS= file-user@example.com \nBILLAR_AUTH_ALLOWED_DOMAINS=file.example.com\nBILLAR_AUTH_RESOURCE_SERVER_URI=https://resource.from.file\n")
	if err := os.WriteFile(filepath.Join(workdir, ".env"), content, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	for _, key := range []string{
		"BILLAR_OAUTH_CLIENT_ID",
		"BILLAR_OAUTH_CLIENT_SECRET",
		"BILLAR_OAUTH_ISSUER_URL",
		"BILLAR_AUTH_ALLOWED_EMAILS",
		"BILLAR_AUTH_ALLOWED_DOMAINS",
		"BILLAR_AUTH_RESOURCE_SERVER_URI",
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
		ClientSecret:      "from-file-secret",
		IssuerURL:         "https://issuer.from.file",
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
	content := []byte("BILLAR_OAUTH_CLIENT_ID='from-file'\nBILLAR_OAUTH_CLIENT_SECRET=\"from-file-secret\"\nBILLAR_OAUTH_ISSUER_URL='https://issuer.from.file'\nBILLAR_AUTH_ALLOWED_EMAILS=' file-user@example.com '\nBILLAR_AUTH_ALLOWED_DOMAINS=\"file.example.com\"\nBILLAR_AUTH_RESOURCE_SERVER_URI='https://resource.from.file'\n")
	if err := os.WriteFile(filepath.Join(workdir, ".env"), content, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	t.Setenv("BILLAR_OAUTH_CLIENT_ID", "preset-client-id")
	t.Setenv("BILLAR_OAUTH_CLIENT_SECRET", "")
	t.Setenv("BILLAR_OAUTH_ISSUER_URL", "")
	t.Setenv("BILLAR_AUTH_ALLOWED_EMAILS", "")
	t.Setenv("BILLAR_AUTH_ALLOWED_DOMAINS", "")
	t.Setenv("BILLAR_AUTH_RESOURCE_SERVER_URI", "")

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
		ClientSecret:      "from-file-secret",
		IssuerURL:         "https://issuer.from.file",
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
		"BILLAR_OAUTH_CLIENT_ID=",
		"BILLAR_OAUTH_CLIENT_SECRET=",
		"BILLAR_OAUTH_ISSUER_URL=",
		"BILLAR_AUTH_RESOURCE_SERVER_URI=",
		"BILLAR_AUTH_ALLOWED_EMAILS=",
		"BILLAR_AUTH_ALLOWED_DOMAINS=",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf(".env.example missing %q", want)
		}
	}
}
