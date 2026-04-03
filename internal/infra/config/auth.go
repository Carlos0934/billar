package config

import (
	"errors"
	"os"
	"strings"
)

type AuthConfig struct {
	ClientID          string
	ClientSecret      string
	IssuerURL         string
	AllowedEmails     []string
	AllowedDomains    []string
	ResourceServerURI string
}

func LoadAuthConfig() (AuthConfig, error) {
	if err := loadEnvFile(".env"); err != nil {
		return AuthConfig{}, err
	}

	cfg := AuthConfig{
		ClientID:          strings.TrimSpace(os.Getenv("BILLAR_OAUTH_CLIENT_ID")),
		ClientSecret:      strings.TrimSpace(os.Getenv("BILLAR_OAUTH_CLIENT_SECRET")),
		IssuerURL:         strings.TrimSpace(os.Getenv("BILLAR_OAUTH_ISSUER_URL")),
		AllowedEmails:     splitAndTrimCSV(os.Getenv("BILLAR_AUTH_ALLOWED_EMAILS")),
		AllowedDomains:    splitAndTrimCSV(os.Getenv("BILLAR_AUTH_ALLOWED_DOMAINS")),
		ResourceServerURI: strings.TrimSpace(os.Getenv("BILLAR_AUTH_RESOURCE_SERVER_URI")),
	}

	if len(cfg.AllowedEmails) == 0 && len(cfg.AllowedDomains) == 0 {
		return AuthConfig{}, errors.New("auth access policy requires at least one allowed email or allowed domain")
	}

	return cfg, nil
}
