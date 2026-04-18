package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Carlos0934/billar/internal/app"
	mcpconnector "github.com/Carlos0934/billar/internal/connectors/mcp"
	mcphttpconnector "github.com/Carlos0934/billar/internal/connectors/mcphttp"
	"github.com/Carlos0934/billar/internal/infra/config"
	"github.com/Carlos0934/billar/internal/infra/logging"
	infrasqlite "github.com/Carlos0934/billar/internal/infra/sqlite"
)

var listenAndServe = func(server *http.Server) error {
	return server.ListenAndServe()
}

func newServer(authCfg config.AuthConfig, appCfg config.Config, store *infrasqlite.Store, logger *slog.Logger) (*http.Server, error) {
	legalEntityStore := infrasqlite.NewLegalEntityStore(store)
	issuerProfileStore := infrasqlite.NewIssuerProfileStore(store)
	customerProfileStore := infrasqlite.NewCustomerProfileStore(store)
	agreementStore := infrasqlite.NewServiceAgreementStore(store)
	timeEntryStore := infrasqlite.NewTimeEntryStore(store)
	invoiceStore := infrasqlite.NewInvoiceStore(store)
	invoiceSequenceStore := infrasqlite.NewInvoiceSequenceStore(store)

	issuerProfileService := app.NewIssuerProfileService(legalEntityStore, issuerProfileStore)
	customerProfileService := app.NewCustomerProfileService(legalEntityStore, customerProfileStore)
	agreementService := app.NewAgreementService(agreementStore, customerProfileStore)
	timeEntryService := app.NewTimeEntryService(timeEntryStore, customerProfileStore, agreementStore)
	invoiceService := app.NewInvoiceService(invoiceStore, timeEntryStore, agreementStore, customerProfileStore, invoiceSequenceStore)

	mcpSessionService := app.NewRequestSessionService(app.ContextIdentitySource{})
	healthService := app.NewHealthService(appCfg.AppName)

	mcpServer := mcpconnector.NewServer(
		mcpSessionService,
		issuerProfileService,
		customerProfileService,
		agreementService,
		timeEntryService,
		invoiceService,
		logger,
	)
	mcpAuthMiddleware := mcphttpconnector.NewAPIKeyAuthMiddleware(authCfg.APIKeys, logger)

	mux := http.NewServeMux()
	mux.Handle("/healthz", mcphttpconnector.HealthHandler(healthService))
	mux.Handle("/v1/mcp", mcpAuthMiddleware.Wrap(mcpServer.HTTPHandler()))

	return &http.Server{
		Addr:              authCfg.ListenAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}, nil
}

func main() {
	logger := logging.New()
	authCfg, err := config.LoadAuthConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	appCfg := config.Load()

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

	server, err := newServer(authCfg, appCfg, store, logger)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	shutdownCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-shutdownCtx.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()

	logger.Info("server listening", slog.String("server", "mcp-http"), slog.String("addr", authCfg.ListenAddr))
	if err := listenAndServe(server); err != nil && err != http.ErrServerClosed {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
