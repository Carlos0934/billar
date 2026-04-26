package config

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveDBPath(t *testing.T) {
	tests := []struct {
		name          string
		env           map[string]string
		homeDir       func() (string, error)
		userConfigDir func() (string, error)
		want          string
		wantErr       string
	}{
		{
			name: "explicit db path wins",
			env: map[string]string{
				"BILLAR_DB_PATH": "/var/data/billar.db",
				"XDG_DATA_HOME":  "/x",
			},
			homeDir:       errHomeDir,
			userConfigDir: errUserConfigDir,
			want:          "/var/data/billar.db",
		},
		{
			name: "whitespace db path uses xdg data home",
			env: map[string]string{
				"BILLAR_DB_PATH": "   ",
				"XDG_DATA_HOME":  "/x",
			},
			homeDir:       errHomeDir,
			userConfigDir: errUserConfigDir,
			want:          filepath.Join("/x", "billar", "billar.db"),
		},
		{
			name: "xdg data home default",
			env: map[string]string{
				"XDG_DATA_HOME": "/x",
			},
			homeDir:       errHomeDir,
			userConfigDir: errUserConfigDir,
			want:          filepath.Join("/x", "billar", "billar.db"),
		},
		{
			name:          "user config wins over home when xdg data home is unset",
			env:           map[string]string{},
			homeDir:       func() (string, error) { return "/h", nil },
			userConfigDir: func() (string, error) { return "/c", nil },
			want:          filepath.Join("/c", "billar", "billar.db"),
		},
		{
			name:          "home final fallback",
			env:           map[string]string{},
			homeDir:       func() (string, error) { return "/h", nil },
			userConfigDir: errUserConfigDir,
			want:          filepath.Join("/h", ".local", "share", "billar", "billar.db"),
		},
		{
			name:          "user config fallback",
			env:           map[string]string{},
			homeDir:       errHomeDir,
			userConfigDir: func() (string, error) { return "/c", nil },
			want:          filepath.Join("/c", "billar", "billar.db"),
		},
		{
			name:          "all bases unavailable",
			env:           map[string]string{},
			homeDir:       errHomeDir,
			userConfigDir: errUserConfigDir,
			wantErr:       "BILLAR_DB_PATH",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getenv := func(key string) string { return tt.env[key] }
			got, err := resolveDBPathWith(getenv, tt.homeDir, tt.userConfigDir)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("resolveDBPathWith() error = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveDBPathWith() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("resolveDBPathWith() = %q, want %q", got, tt.want)
			}
		})
	}
}

func errHomeDir() (string, error) {
	return "", errors.New("home unavailable")
}

func errUserConfigDir() (string, error) {
	return "", errors.New("config unavailable")
}
