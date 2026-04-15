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

func newCommand(cfg config.Config, store *infrasqlite.Store) connectorcli.Command {
	legalEntityStore := infrasqlite.NewLegalEntityStore(store)
	issuerProfileStore := infrasqlite.NewIssuerProfileStore(store)
	customerProfileStore := infrasqlite.NewCustomerProfileStore(store)
	agreementStore := infrasqlite.NewServiceAgreementStore(store)
	timeEntryStore := infrasqlite.NewTimeEntryStore(store)
	invoiceStore := infrasqlite.NewInvoiceStore(store)
	invoiceSequenceStore := infrasqlite.NewInvoiceSequenceStore(store)

	return connectorcli.NewCommand(
		app.NewHealthService(cfg.AppName),
		app.NewLegalEntityService(legalEntityStore),
		app.NewIssuerProfileService(legalEntityStore, issuerProfileStore),
		app.NewCustomerProfileService(legalEntityStore, customerProfileStore),
		app.NewAgreementService(agreementStore, customerProfileStore),
		app.NewTimeEntryService(timeEntryStore, customerProfileStore, agreementStore),
		app.NewInvoiceService(invoiceStore, timeEntryStore, agreementStore, customerProfileStore, invoiceSequenceStore),
		cfg.ColorEnabled,
	)
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

	cmd := newCommand(cfg, store)

	if err := cmd.Run(ctx, os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
