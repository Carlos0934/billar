package config

import (
	"os"
	"strings"
)

type Config struct {
	AppName         string
	ColorEnabled    bool
	ExportDir       string
	ExportDirSource string
	BackupDir       string
	BackupDirSource string
	DBPath          string
	DBPathSource    string
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

	paths, err := Resolve(os.Getenv)
	if err != nil {
		return Config{}, err
	}

	return Config{
		AppName:         appName,
		ColorEnabled:    colorEnabled,
		ExportDir:       paths.ExportDir.Path,
		ExportDirSource: paths.ExportDir.Source,
		BackupDir:       paths.BackupDir.Path,
		BackupDirSource: paths.BackupDir.Source,
		DBPath:          paths.DBPath.Path,
		DBPathSource:    paths.DBPath.Source,
	}, nil
}
