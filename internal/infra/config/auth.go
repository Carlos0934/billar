package config

import (
	"errors"
	"os"
	"strings"
)

type AuthConfig struct {
	APIKeys    []string
	ListenAddr string
}

func LoadAuthConfig() (AuthConfig, error) {
	if err := loadEnvFile(".env"); err != nil {
		return AuthConfig{}, err
	}

	cfg := AuthConfig{
		APIKeys:    splitAndTrimCSV(os.Getenv("MCP_API_KEYS")),
		ListenAddr: strings.TrimSpace(os.Getenv("MCP_HTTP_LISTEN_ADDR")),
	}

	if cfg.ListenAddr == "" {
		cfg.ListenAddr = "127.0.0.1:8080"
	}

	if len(cfg.APIKeys) == 0 {
		return AuthConfig{}, errors.New("MCP_API_KEYS is required: set one or more comma-separated API keys")
	}

	return cfg, nil
}
