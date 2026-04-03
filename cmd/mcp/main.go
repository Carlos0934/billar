package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Carlos0934/billar/internal/app"
	mcpconnector "github.com/Carlos0934/billar/internal/connectors/mcp"
	"github.com/Carlos0934/billar/internal/core"
	infraauth "github.com/Carlos0934/billar/internal/infra/auth"
	"github.com/Carlos0934/billar/internal/infra/config"
	infrasqlite "github.com/Carlos0934/billar/internal/infra/sqlite"
)

type localSessionService struct {
	provider string
	sessions *infraauth.MemorySessionStore
}

func (s localSessionService) StartLogin(context.Context) (app.LoginIntentDTO, error) {
	provider := strings.TrimSpace(s.provider)
	if provider == "" {
		provider = "openai"
	}

	return app.LoginIntentDTO{LoginURL: "https://login.example/" + provider}, nil
}

func (s localSessionService) Status(ctx context.Context) (app.SessionStatusDTO, error) {
	if s.sessions == nil {
		return app.SessionStatusDTO{Status: "unauthenticated"}, nil
	}

	session, err := s.sessions.GetCurrent(ctx)
	if err != nil {
		return app.SessionStatusDTO{}, err
	}
	if session == nil || session.Status != core.SessionStatusActive {
		return app.SessionStatusDTO{Status: "unauthenticated"}, nil
	}

	return app.SessionStatusDTO{
		Status:        session.Status.String(),
		Email:         session.Identity.Email,
		EmailVerified: session.Identity.EmailVerified,
		Subject:       session.Identity.Subject,
		Issuer:        session.Identity.Issuer,
	}, nil
}

func (s localSessionService) Logout(ctx context.Context) (app.LogoutDTO, error) {
	if s.sessions != nil {
		if err := s.sessions.Save(ctx, &core.Session{Status: core.SessionStatusUnauthenticated}); err != nil {
			return app.LogoutDTO{}, err
		}
	}

	return app.LogoutDTO{Message: "Logged out"}, nil
}

func main() {
	ctx := context.Background()
	cfg := config.Load()
	store, err := infrasqlite.Open(os.Getenv("BILLAR_DB_PATH"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer func() {
		if err := store.Close(); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}()

	sessionStore := infraauth.NewMemorySessionStore()
	if email := os.Getenv("BILLAR_SESSION_EMAIL"); email != "" {
		_ = sessionStore.Save(ctx, &core.Session{Status: core.SessionStatusActive, Identity: core.Identity{Email: email, EmailVerified: true}})
	}

	server := mcpconnector.NewServer(localSessionService{provider: cfg.AccessPolicy.OAuthProvider, sessions: sessionStore}, app.NewCustomerService(sessionStore, infrasqlite.NewCustomerStore(store)), mcpconnector.NewIngressGuardFromConfig(cfg.AccessPolicy))

	if err := server.ServeStdio(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
