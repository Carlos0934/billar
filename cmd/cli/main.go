package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Carlos0934/billar/internal/app"
	connectorcli "github.com/Carlos0934/billar/internal/connectors/cli"
	"github.com/Carlos0934/billar/internal/core"
	infraauth "github.com/Carlos0934/billar/internal/infra/auth"
	"github.com/Carlos0934/billar/internal/infra/config"
	infrasqlite "github.com/Carlos0934/billar/internal/infra/sqlite"
)

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

	cmd := connectorcli.NewCommand(app.NewHealthService(cfg.AppName), app.NewCustomerService(sessionStore, infrasqlite.NewCustomerStore(store)), cfg.ColorEnabled)

	if err := cmd.Run(ctx, os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
