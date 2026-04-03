package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Carlos0934/billar/internal/app"
	mcpconnector "github.com/Carlos0934/billar/internal/connectors/mcp"
	"github.com/Carlos0934/billar/internal/infra/config"
)

type noopSessionService struct {
	provider string
}

func (s noopSessionService) StartLogin(context.Context) (app.LoginIntentDTO, error) {
	provider := strings.TrimSpace(s.provider)
	if provider == "" {
		provider = "openai"
	}

	return app.LoginIntentDTO{LoginURL: "https://login.example/" + provider}, nil
}

func (noopSessionService) Status(context.Context) (app.SessionStatusDTO, error) {
	return app.SessionStatusDTO{Status: "unauthenticated"}, nil
}

func (noopSessionService) Logout(context.Context) (app.LogoutDTO, error) {
	return app.LogoutDTO{Message: "Logged out"}, nil
}

func main() {
	cfg := config.Load()
	server := mcpconnector.NewServer(noopSessionService{provider: cfg.AccessPolicy.OAuthProvider}, mcpconnector.NewIngressGuard(cfg.AccessPolicy))

	if err := server.ServeStdio(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
