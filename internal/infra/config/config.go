package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	AppName      string
	ColorEnabled bool
	ExportDir    string
	DBPath       string
}

func Load() (Config, error) {
	_ = loadEnvFile(".env")

	appName := strings.TrimSpace(os.Getenv("BILLAR_APP_NAME"))
	if appName == "" {
		appName = "billar"
	}

	colorEnabled := true
	if os.Getenv("NO_COLOR") != "" || strings.EqualFold(os.Getenv("TERM"), "dumb") {
		colorEnabled = false
	}

	dbPath, err := resolveDBPath()
	if err != nil {
		return Config{}, err
	}
	if strings.TrimSpace(os.Getenv("BILLAR_DB_PATH")) == "" {
		if err := ensureDefaultDBParentDir(dbPath); err != nil {
			return Config{}, err
		}
	}

	return Config{
		AppName:      appName,
		ColorEnabled: colorEnabled,
		ExportDir:    strings.TrimSpace(os.Getenv("BILLAR_EXPORT_DIR")),
		DBPath:       dbPath,
	}, nil
}

func ensureDefaultDBParentDir(dbPath string) error {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o700); err != nil {
		return fmt.Errorf("create database directory for %q: %w; set BILLAR_DB_PATH to a writable SQLite database path", dbPath, err)
	}
	return nil
}
