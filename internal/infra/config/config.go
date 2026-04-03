package config

import (
	"os"
	"strings"
)

type Config struct {
	AppName      string
	ColorEnabled bool
	AccessPolicy AccessPolicy
}

type AccessPolicy struct {
	AllowedEmails  []string
	AllowedDomains []string
	AllowedIPs     []string
	OAuthProvider  string
}

func Load() Config {
	_ = loadEnvFile(".env")

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
		AccessPolicy: AccessPolicy{
			AllowedEmails:  splitAndTrimCSV(os.Getenv("ALLOWED_EMAILS")),
			AllowedDomains: splitAndTrimCSV(os.Getenv("ALLOWED_DOMAINS")),
			AllowedIPs:     splitAndTrimCSV(os.Getenv("ALLOWED_IPS")),
			OAuthProvider:  strings.TrimSpace(os.Getenv("OAUTH_PROVIDER")),
		},
	}
}
