package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Carlos0934/billar/internal/app"
	connectorcli "github.com/Carlos0934/billar/internal/connectors/cli"
	"github.com/Carlos0934/billar/internal/infra/config"
	"github.com/Carlos0934/billar/internal/infra/exportfs"
	"github.com/Carlos0934/billar/internal/infra/pdf"
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
	invoiceService := app.NewInvoiceService(invoiceStore, timeEntryStore, agreementStore, customerProfileStore, invoiceSequenceStore, issuerProfileStore)
	invoicePDFService := app.NewInvoicePDFService(invoiceStore, timeEntryStore, customerProfileStore, issuerProfileStore, legalEntityStore, pdf.Renderer{}, exportfs.DirectWriter{})
	invoiceProvider := app.NewInvoiceProvider(invoiceService, invoicePDFService)

	return connectorcli.NewCommand(
		app.NewHealthService(cfg.AppName),
		app.NewLegalEntityService(legalEntityStore),
		app.NewIssuerProfileService(legalEntityStore, issuerProfileStore),
		app.NewCustomerProfileService(legalEntityStore, customerProfileStore),
		app.NewAgreementService(agreementStore, customerProfileStore),
		app.NewTimeEntryService(timeEntryStore, customerProfileStore, agreementStore),
		invoiceProvider,
		cfg.ColorEnabled,
	)
}

func main() {
	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	store, err := openConfiguredStore(cfg)
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

func openConfiguredStore(cfg config.Config) (*infrasqlite.Store, error) {
	store, err := infrasqlite.Open(cfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database at %q: %w; set BILLAR_DB_PATH to choose a writable database path", cfg.DBPath, err)
	}
	return store, nil
}
