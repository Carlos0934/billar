package config

import (
	"os"
	"strings"
)

type Config struct {
	AppName      string
	ColorEnabled bool
}

func Load() Config {
	appName := strings.TrimSpace(os.Getenv("BILLAR_APP_NAME"))
	if appName == "" {
		appName = "billar"
	}

	colorEnabled := true
	if os.Getenv("NO_COLOR") != "" || strings.EqualFold(os.Getenv("TERM"), "dumb") {
		colorEnabled = false
	}

	return Config{
		AppName:      appName,
		ColorEnabled: colorEnabled,
	}
}
