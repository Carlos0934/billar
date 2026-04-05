package main

import (
	"fmt"
	"os"

	"github.com/Carlos0934/billar/internal/app"
	mcpconnector "github.com/Carlos0934/billar/internal/connectors/mcp"
	"github.com/Carlos0934/billar/internal/infra/config"
	"github.com/Carlos0934/billar/internal/infra/logging"
	infrasqlite "github.com/Carlos0934/billar/internal/infra/sqlite"
)

func main() {
	logger := logging.New()
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

	issuerProfileService := app.NewIssuerProfileService(infrasqlite.NewLegalEntityStore(store), infrasqlite.NewIssuerProfileStore(store))
	customerProfileService := app.NewCustomerProfileService(infrasqlite.NewLegalEntityStore(store), infrasqlite.NewCustomerProfileStore(store))

	identities, err := app.NewLocalBypassIdentitySource(os.Getenv("BILLAR_LOCAL_AUTH_EMAIL"), app.IdentityPolicy{
		AllowedEmails:  cfg.AccessPolicy.AllowedEmails,
		AllowedDomains: cfg.AccessPolicy.AllowedDomains,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	server := mcpconnector.NewServer(
		app.NewRequestSessionService(identities),
		issuerProfileService,
		customerProfileService,
		mcpconnector.NewIngressGuardFromConfig(cfg.AccessPolicy),
		logger,
	)

	if err := server.ServeStdio(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
