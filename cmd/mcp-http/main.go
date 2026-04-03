package main

import (
	"context"
	"fmt"
	"log"
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
	infrasqlite "github.com/Carlos0934/billar/internal/infra/sqlite"
)

func main() {
	ctx := context.Background()
	authCfg, err := config.LoadAuthConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	appCfg := config.Load()

	googleOIDC, err := infraauth.NewGoogleOIDC(ctx, authCfg.IssuerURL, authCfg.ClientID, authCfg.ClientSecret, authCfg.RedirectURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	stateStore := infraauth.NewMemoryStateStore(10 * time.Minute)
	sessionStore := infraauth.NewMemorySessionStore()
	sessionService := app.NewAuthSessionService(
		googleOIDC,
		googleOIDC,
		googleOIDC,
		app.IdentityPolicy{AllowedEmails: authCfg.AllowedEmails, AllowedDomains: authCfg.AllowedDomains},
		stateStore,
		sessionStore,
	)
	customerStore, err := infrasqlite.Open(os.Getenv("BILLAR_DB_PATH"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer func() {
		if err := customerStore.Close(); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}()
	customerService := app.NewCustomerService(sessionStore, infrasqlite.NewCustomerStore(customerStore))
	healthService := app.NewHealthService(appCfg.AppName)
	mcpServer := mcpconnector.NewServer(sessionService, customerService, mcpconnector.NewIngressGuardFromConfig(appCfg.AccessPolicy))

	mux := http.NewServeMux()
	mux.Handle("/healthz", mcphttpconnector.HealthHandler(healthService))
	mux.Handle("/v1/mcp", mcpServer.HTTPHandler())
	mux.Handle("/.well-known/oauth-protected-resource", mcphttpconnector.MetadataHandler(app.OAuthChallengeDTO{
		ResourceURI:          authCfg.ResourceServerURI,
		AuthorizationServers: []string{authCfg.IssuerURL},
	}))
	mux.Handle("/auth/login/start", mcphttpconnector.LoginHandler(sessionService))
	mux.Handle("/auth/callback", mcphttpconnector.CallbackHandler(sessionService, stateStore))
	mux.Handle("/auth/session", mcphttpconnector.SessionStatusHandler(sessionService))
	mux.Handle("/auth/logout", mcphttpconnector.LogoutHandler(sessionService))

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

	log.Printf("mcp-http server listening on http://%s", authCfg.ListenAddr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
