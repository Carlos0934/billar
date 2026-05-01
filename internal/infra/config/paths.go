package config

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	pathSourceConfigured = "configured"
	pathSourceDefault    = "default"
	exportDirName        = "exports"
	backupDirName        = "backups"
)

type PathValue struct {
	Path   string
	Source string
}

type Paths struct {
	DBPath    PathValue
	ExportDir PathValue
	BackupDir PathValue
}

func Resolve(getenv func(string) string) (Paths, error) {
	return resolvePathsWith(getenv, os.UserHomeDir, os.UserConfigDir)
}

func ResolveBackupDir(getenv func(string) string) (PathValue, error) {
	return resolveBackupDirWith(getenv, os.UserHomeDir, os.UserConfigDir)
}

func resolvePathsWith(getenv func(string) string, homeDir, userConfigDir func() (string, error)) (Paths, error) {
	dbPath, err := resolveDBPathWith(getenv, homeDir, userConfigDir)
	if err != nil {
		return Paths{}, err
	}

	dbSource := pathSourceDefault
	if strings.TrimSpace(getenv("BILLAR_DB_PATH")) != "" {
		dbSource = pathSourceConfigured
	}

	exportDir := PathValue{
		Path:   filepath.Join(filepath.Dir(dbPath), exportDirName),
		Source: pathSourceDefault,
	}
	if path := strings.TrimSpace(getenv("BILLAR_EXPORT_DIR")); path != "" {
		exportDir = PathValue{Path: path, Source: pathSourceConfigured}
	}

	backupDir := PathValue{
		Path:   filepath.Join(filepath.Dir(dbPath), backupDirName),
		Source: pathSourceDefault,
	}
	if path := strings.TrimSpace(getenv("BILLAR_BACKUP_DIR")); path != "" {
		backupDir = PathValue{Path: path, Source: pathSourceConfigured}
	}

	return Paths{
		DBPath:    PathValue{Path: dbPath, Source: dbSource},
		ExportDir: exportDir,
		BackupDir: backupDir,
	}, nil
}

func resolveBackupDirWith(getenv func(string) string, homeDir, userConfigDir func() (string, error)) (PathValue, error) {
	paths, err := resolvePathsWith(getenv, homeDir, userConfigDir)
	if err != nil {
		return PathValue{}, err
	}
	return paths.BackupDir, nil
}
