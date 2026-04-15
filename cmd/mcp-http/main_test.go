package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Carlos0934/billar/internal/app"
	"github.com/Carlos0934/billar/internal/infra/config"
	infrasqlite "github.com/Carlos0934/billar/internal/infra/sqlite"
	"github.com/mark3labs/mcp-go/client"
	transport "github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

type stubMCPHTTPAuthenticator struct{}

func (stubMCPHTTPAuthenticator) Authenticate(ctx context.Context, bearerToken string) (app.AuthenticatedIdentity, error) {
	_ = ctx
	_ = bearerToken
	return app.AuthenticatedIdentity{Email: "integration@example.com", EmailVerified: true, Subject: "local-bypass"}, nil
}

func TestNewServerWiresHTTPRoutesAndTimeEntryTools(t *testing.T) {
	t.Parallel()

	store := mustOpenMCPHTTPStore(t)
	server, err := newServer(
		config.AuthConfig{ListenAddr: "127.0.0.1:0", IssuerURL: "https://issuer.example", ResourceServerURI: "https://resource.example"},
		config.Config{AppName: "billar"},
		stubMCPHTTPAuthenticator{},
		store,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	if err != nil {
		t.Fatalf("newServer() error = %v", err)
	}

	if server == nil || server.Handler == nil {
		t.Fatal("newServer() returned incomplete server")
	}

	httpServer := httptest.NewServer(server.Handler)
	t.Cleanup(httpServer.Close)

	resp, err := http.Get(httpServer.URL + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /healthz status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	var health app.HealthDTO
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		t.Fatalf("decode /healthz body: %v", err)
	}
	if health.Name != "billar" || health.Status != "ok" {
		t.Fatalf("/healthz payload = %+v, want billar ok", health)
	}

	resp, err = http.Get(httpServer.URL + "/.well-known/oauth-protected-resource")
	if err != nil {
		t.Fatalf("GET metadata error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET metadata status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if got := resp.Header.Get("Content-Type"); !strings.Contains(got, "application/json") {
		t.Fatalf("metadata content-type = %q, want json", got)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	httpTransport, err := transport.NewStreamableHTTP(httpServer.URL+"/v1/mcp", transport.WithHTTPHeaders(map[string]string{"Authorization": "Bearer test-token"}))
	if err != nil {
		t.Fatalf("NewStreamableHTTP() error = %v", err)
	}
	mcpClient := client.NewClient(httpTransport)
	if err := mcpClient.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	t.Cleanup(func() {
		if err := mcpClient.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	})

	if _, err := mcpClient.Initialize(ctx, mcp.InitializeRequest{Params: mcp.InitializeParams{ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION, ClientInfo: mcp.Implementation{Name: "billar-entrypoint-test", Version: "1.0.0"}, Capabilities: mcp.ClientCapabilities{}}}); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	toolsResult, err := mcpClient.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}
	var toolNames []string
	for _, tool := range toolsResult.Tools {
		toolNames = append(toolNames, tool.Name)
	}
	tools := strings.Join(toolNames, ",")
	for _, want := range []string{"session.status", "time_entry.record", "time_entry.list_unbilled", "invoice.draft", "invoice.issue", "invoice.discard"} {
		if !strings.Contains(tools, want) {
			t.Fatalf("ListTools() names = %q, want %q", tools, want)
		}
	}
}

func TestMainWiresHTTPServer(t *testing.T) {
	t.Setenv("OAUTH_CLIENT_ID", "test-client")
	t.Setenv("OAUTH_ISSUER_URL", "https://issuer.example")
	t.Setenv("MCP_HTTP_LISTEN_ADDR", "127.0.0.1:0")
	t.Setenv("AUTH_ALLOWED_EMAILS", "integration@example.com")
	t.Setenv("AUTH_RESOURCE_SERVER_URI", "https://resource.example")
	t.Setenv("BILLAR_DB_PATH", t.TempDir()+"/mcp-http-main.db")

	oldListenAndServe := listenAndServe
	listenAndServe = func(server *http.Server) error {
		if server == nil {
			t.Fatal("ListenAndServe received nil server")
		}
		return http.ErrServerClosed
	}
	t.Cleanup(func() { listenAndServe = oldListenAndServe })

	main()
}

func mustOpenMCPHTTPStore(t *testing.T) *infrasqlite.Store {
	t.Helper()

	store, err := infrasqlite.Open(t.TempDir() + "/mcp-http-entrypoint.db")
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	})
	return store
}
