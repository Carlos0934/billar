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
			name: "parses single api key",
			env: map[string]string{
				"MCP_API_KEYS":         "key-abc123",
				"MCP_HTTP_LISTEN_ADDR": "127.0.0.1:8080",
			},
			want: AuthConfig{
				APIKeys:    []string{"key-abc123"},
				ListenAddr: "127.0.0.1:8080",
			},
		},
		{
			name: "parses multiple comma-separated api keys",
			env: map[string]string{
				"MCP_API_KEYS":         " key-one , key-two , key-three ",
				"MCP_HTTP_LISTEN_ADDR": "127.0.0.1:9090",
			},
			want: AuthConfig{
				APIKeys:    []string{"key-one", "key-two", "key-three"},
				ListenAddr: "127.0.0.1:9090",
			},
		},
		{
			name: "defaults listen addr when missing",
			env: map[string]string{
				"MCP_API_KEYS":         "key-abc",
				"MCP_HTTP_LISTEN_ADDR": "",
			},
			want: AuthConfig{
				APIKeys:    []string{"key-abc"},
				ListenAddr: "127.0.0.1:8080",
			},
		},
		{
			name: "fails when MCP_API_KEYS is empty",
			env: map[string]string{
				"MCP_API_KEYS":         "",
				"MCP_HTTP_LISTEN_ADDR": "127.0.0.1:8080",
			},
			wantErr: "MCP_API_KEYS",
		},
		{
			name: "fails when MCP_API_KEYS is whitespace only",
			env: map[string]string{
				"MCP_API_KEYS":         "   ",
				"MCP_HTTP_LISTEN_ADDR": "127.0.0.1:8080",
			},
			wantErr: "MCP_API_KEYS",
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			for _, key := range []string{
				"MCP_API_KEYS",
				"MCP_HTTP_LISTEN_ADDR",
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

// TestLoadAuthConfigFailsWhenMCPAPIKeysIsUnset verifies that LoadAuthConfig returns an error
// when MCP_API_KEYS is not present in the environment at all (os.LookupEnv returns false).
// This is distinct from the empty/whitespace cases tested in TestLoadAuthConfig.
func TestLoadAuthConfigFailsWhenMCPAPIKeysIsUnset(t *testing.T) {
	// Use a temp dir so there is no .env file that could set MCP_API_KEYS.
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })

	// Ensure MCP_API_KEYS is fully absent from the environment.
	t.Setenv("MCP_API_KEYS", "placeholder") // registers cleanup via t.Cleanup
	if err := os.Unsetenv("MCP_API_KEYS"); err != nil {
		t.Fatalf("Unsetenv() error = %v", err)
	}
	t.Setenv("MCP_HTTP_LISTEN_ADDR", "127.0.0.1:8080")

	_, err = LoadAuthConfig()
	if err == nil {
		t.Fatal("LoadAuthConfig() error = nil, want non-nil when MCP_API_KEYS is unset")
	}
	if !strings.Contains(err.Error(), "MCP_API_KEYS") {
		t.Fatalf("LoadAuthConfig() error = %q, want it to mention MCP_API_KEYS", err.Error())
	}
}

func TestLoadAuthConfigLoadsDotEnv(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}

	workdir := t.TempDir()
	content := []byte("MCP_API_KEYS=from-file-key\nMCP_HTTP_LISTEN_ADDR=127.0.0.1:8080\n")
	if err := os.WriteFile(filepath.Join(workdir, ".env"), content, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	for _, key := range []string{
		"MCP_API_KEYS",
		"MCP_HTTP_LISTEN_ADDR",
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
		APIKeys:    []string{"from-file-key"},
		ListenAddr: "127.0.0.1:8080",
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
		"MCP_API_KEYS=",
		"MCP_HTTP_LISTEN_ADDR=",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf(".env.example missing %q", want)
		}
	}
}
