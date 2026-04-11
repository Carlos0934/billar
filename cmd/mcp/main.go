package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/Carlos0934/billar/internal/app"
	mcpconnector "github.com/Carlos0934/billar/internal/connectors/mcp"
	"github.com/Carlos0934/billar/internal/infra/config"
	"github.com/Carlos0934/billar/internal/infra/logging"
	infrasqlite "github.com/Carlos0934/billar/internal/infra/sqlite"
)

var serveStdio = func(server *mcpconnector.Server) error {
	return server.ServeStdio()
}

func newServer(cfg config.Config, localAuthEmail string, store *infrasqlite.Store, logger *slog.Logger) (*mcpconnector.Server, error) {
	legalEntityStore := infrasqlite.NewLegalEntityStore(store)
	issuerProfileStore := infrasqlite.NewIssuerProfileStore(store)
	customerProfileStore := infrasqlite.NewCustomerProfileStore(store)
	agreementStore := infrasqlite.NewServiceAgreementStore(store)
	timeEntryStore := infrasqlite.NewTimeEntryStore(store)

	identities, err := app.NewLocalBypassIdentitySource(localAuthEmail, app.IdentityPolicy{
		AllowedEmails:  cfg.AccessPolicy.AllowedEmails,
		AllowedDomains: cfg.AccessPolicy.AllowedDomains,
	})
	if err != nil {
		return nil, err
	}

	return mcpconnector.NewServer(
		app.NewRequestSessionService(identities),
		app.NewIssuerProfileService(legalEntityStore, issuerProfileStore),
		app.NewCustomerProfileService(legalEntityStore, customerProfileStore),
		app.NewAgreementService(agreementStore, customerProfileStore),
		app.NewTimeEntryService(timeEntryStore, customerProfileStore, agreementStore),
		mcpconnector.NewIngressGuardFromConfig(cfg.AccessPolicy),
		logger,
	), nil
}

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

	server, err := newServer(cfg, os.Getenv("BILLAR_LOCAL_AUTH_EMAIL"), store, logger)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := serveStdio(server); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
