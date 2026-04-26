package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	dbDirName  = "billar"
	dbFileName = "billar.db"
)

func resolveDBPath() (string, error) {
	return resolveDBPathWith(os.Getenv, os.UserHomeDir, os.UserConfigDir)
}

func resolveDBPathWith(getenv func(string) string, homeDir, userConfigDir func() (string, error)) (string, error) {
	if path := strings.TrimSpace(getenv("BILLAR_DB_PATH")); path != "" {
		return path, nil
	}

	if dataHome := strings.TrimSpace(getenv("XDG_DATA_HOME")); dataHome != "" {
		return filepath.Join(dataHome, dbDirName, dbFileName), nil
	}

	if configDir, err := userConfigDir(); err == nil && strings.TrimSpace(configDir) != "" {
		return filepath.Join(configDir, dbDirName, dbFileName), nil
	}

	if home, err := homeDir(); err == nil && strings.TrimSpace(home) != "" {
		return filepath.Join(home, ".local", "share", dbDirName, dbFileName), nil
	}

	return "", fmt.Errorf("resolve database path: set BILLAR_DB_PATH")
}
