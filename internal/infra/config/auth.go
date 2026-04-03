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
	RedirectURL       string
	ListenAddr        string
	AllowedEmails     []string
	AllowedDomains    []string
	ResourceServerURI string
}

func LoadAuthConfig() (AuthConfig, error) {
	if err := loadEnvFile(".env"); err != nil {
		return AuthConfig{}, err
	}

	cfg := AuthConfig{
		ClientID:          strings.TrimSpace(os.Getenv("OAUTH_CLIENT_ID")),
		ClientSecret:      strings.TrimSpace(os.Getenv("OAUTH_CLIENT_SECRET")),
		IssuerURL:         strings.TrimSpace(os.Getenv("OAUTH_ISSUER_URL")),
		RedirectURL:       strings.TrimSpace(os.Getenv("OAUTH_REDIRECT_URL")),
		ListenAddr:        strings.TrimSpace(os.Getenv("MCP_HTTP_LISTEN_ADDR")),
		AllowedEmails:     splitAndTrimCSV(os.Getenv("AUTH_ALLOWED_EMAILS")),
		AllowedDomains:    splitAndTrimCSV(os.Getenv("AUTH_ALLOWED_DOMAINS")),
		ResourceServerURI: strings.TrimSpace(os.Getenv("AUTH_RESOURCE_SERVER_URI")),
	}

	if cfg.IssuerURL == "" {
		cfg.IssuerURL = "https://accounts.google.com"
	}
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = "127.0.0.1:8080"
	}
	if cfg.ResourceServerURI == "" {
		cfg.ResourceServerURI = "http://127.0.0.1:8080"
	}
	if cfg.RedirectURL == "" {
		cfg.RedirectURL = cfg.ResourceServerURI + "/auth/callback"
	}

	if len(cfg.AllowedEmails) == 0 && len(cfg.AllowedDomains) == 0 {
		return AuthConfig{}, errors.New("auth access policy requires at least one allowed email or allowed domain")
	}

	return cfg, nil
}
