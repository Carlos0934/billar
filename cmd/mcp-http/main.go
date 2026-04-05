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
	infraauth "github.com/Carlos0934/billar/internal/infra/auth"
	"github.com/Carlos0934/billar/internal/infra/config"
	"github.com/Carlos0934/billar/internal/infra/logging"
	infrasqlite "github.com/Carlos0934/billar/internal/infra/sqlite"
)

func main() {
	logger := logging.New()
	authCfg, err := config.LoadAuthConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	appCfg := config.Load()

	googleAccessTokenAuthenticator, err := infraauth.NewGoogleAccessTokenAuthenticator(authCfg.IssuerURL, authCfg.ClientID)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	identityPolicy := app.IdentityPolicy{AllowedEmails: authCfg.AllowedEmails, AllowedDomains: authCfg.AllowedDomains}
	requestAuthService := app.NewRequestAuthService(googleAccessTokenAuthenticator, identityPolicy)
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

	legalEntityStore := infrasqlite.NewLegalEntityStore(store)
	issuerProfileStore := infrasqlite.NewIssuerProfileStore(store)
	customerProfileStore := infrasqlite.NewCustomerProfileStore(store)

	legalEntityService := app.NewLegalEntityService(legalEntityStore)
	issuerProfileService := app.NewIssuerProfileService(legalEntityStore, issuerProfileStore)
	customerProfileService := app.NewCustomerProfileService(legalEntityStore, customerProfileStore)

	mcpSessionService := app.NewRequestSessionService(app.ContextIdentitySource{})
	healthService := app.NewHealthService(appCfg.AppName)
	mcpChallenge := app.OAuthChallengeDTO{
		ResourceURI:          authCfg.ResourceServerURI,
		AuthorizationServers: []string{authCfg.IssuerURL},
	}
	mcpServer := mcpconnector.NewServer(
		mcpSessionService,
		legalEntityService,
		issuerProfileService,
		customerProfileService,
		mcpconnector.NewIngressGuardFromConfig(appCfg.AccessPolicy),
		logger,
	)
	mcpAuthMiddleware := mcphttpconnector.NewMCPHTTPAuthMiddleware(requestAuthService, mcpChallenge, logger)

	mux := http.NewServeMux()
	mux.Handle("/healthz", mcphttpconnector.HealthHandler(healthService))
	mux.Handle("/v1/mcp", mcpAuthMiddleware.Wrap(mcpServer.HTTPHandler()))
	mux.Handle("/.well-known/oauth-protected-resource", mcphttpconnector.MetadataHandler(mcpChallenge))

	server := &http.Server{
		Addr:              authCfg.ListenAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
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
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
