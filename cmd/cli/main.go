package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Carlos0934/billar/internal/app"
	connectorcli "github.com/Carlos0934/billar/internal/connectors/cli"
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

	identities, err := app.NewLocalBypassIdentitySource(os.Getenv("BILLAR_LOCAL_AUTH_EMAIL"), app.IdentityPolicy{
		AllowedEmails:  cfg.AccessPolicy.AllowedEmails,
		AllowedDomains: cfg.AccessPolicy.AllowedDomains,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	cmd := connectorcli.NewCommand(app.NewHealthService(cfg.AppName), app.NewCustomerService(identities, infrasqlite.NewCustomerStore(store)), cfg.ColorEnabled)

	if err := cmd.Run(ctx, os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
