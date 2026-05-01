package config

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestResolveBackupDir(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		want    PathValue
		wantErr bool
	}{
		{
			name: "configured backup dir wins over db default",
			env: map[string]string{
				"BILLAR_BACKUP_DIR": " /var/backups/billar ",
				"BILLAR_DB_PATH":    "/ignored/billar.db",
			},
			want: PathValue{Path: "/var/backups/billar", Source: "configured"},
		},
		{
			name: "default backup dir is beside configured database",
			env: map[string]string{
				"BILLAR_DB_PATH": "/data/billar/live.db",
			},
			want: PathValue{Path: filepath.Join("/data", "billar", "backups"), Source: "default"},
		},
		{
			name: "default backup dir follows resolved xdg database parent",
			env: map[string]string{
				"XDG_DATA_HOME": "/xdg-data",
			},
			want: PathValue{Path: filepath.Join("/xdg-data", "billar", "backups"), Source: "default"},
		},
		{
			name:    "missing database base reports error",
			env:     map[string]string{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getenv := func(key string) string { return tt.env[key] }
			got, err := resolveBackupDirWith(getenv, errHomeDir, errUserConfigDir)
			if tt.wantErr {
				if err == nil {
					t.Fatal("resolveBackupDirWith() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveBackupDirWith() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("resolveBackupDirWith() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestResolvePaths(t *testing.T) {
	tests := []struct {
		name          string
		env           map[string]string
		homeDir       func() (string, error)
		userConfigDir func() (string, error)
		want          Paths
	}{
		{
			name: "configured paths are reported as configured",
			env: map[string]string{
				"BILLAR_DB_PATH":    "/custom/billar.db",
				"BILLAR_EXPORT_DIR": "/custom/exports",
				"BILLAR_BACKUP_DIR": "/custom/backups",
			},
			homeDir:       errHomeDir,
			userConfigDir: errUserConfigDir,
			want: Paths{
				DBPath:    PathValue{Path: "/custom/billar.db", Source: "configured"},
				ExportDir: PathValue{Path: "/custom/exports", Source: "configured"},
				BackupDir: PathValue{Path: "/custom/backups", Source: "configured"},
			},
		},
		{
			name:          "default paths are derived from database parent without cwd",
			env:           map[string]string{},
			homeDir:       func() (string, error) { return "/home/alice", nil },
			userConfigDir: func() (string, error) { return "/config", nil },
			want: Paths{
				DBPath:    PathValue{Path: filepath.Join("/config", "billar", "billar.db"), Source: "default"},
				ExportDir: PathValue{Path: filepath.Join("/config", "billar", "exports"), Source: "default"},
				BackupDir: PathValue{Path: filepath.Join("/config", "billar", "backups"), Source: "default"},
			},
		},
		{
			name: "whitespace overrides fall back to defaults",
			env: map[string]string{
				"BILLAR_DB_PATH":    "  ",
				"BILLAR_EXPORT_DIR": "\t",
				"BILLAR_BACKUP_DIR": " ",
			},
			homeDir:       func() (string, error) { return "/home/alice", nil },
			userConfigDir: func() (string, error) { return "", errors.New("config unavailable") },
			want: Paths{
				DBPath:    PathValue{Path: filepath.Join("/home/alice", ".local", "share", "billar", "billar.db"), Source: "default"},
				ExportDir: PathValue{Path: filepath.Join("/home/alice", ".local", "share", "billar", "exports"), Source: "default"},
				BackupDir: PathValue{Path: filepath.Join("/home/alice", ".local", "share", "billar", "backups"), Source: "default"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getenv := func(key string) string { return tt.env[key] }
			got, err := resolvePathsWith(getenv, tt.homeDir, tt.userConfigDir)
			if err != nil {
				t.Fatalf("resolvePathsWith() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("resolvePathsWith() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
