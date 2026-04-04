package mcphttp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Carlos0934/billar/internal/app"
	mcpconnector "github.com/Carlos0934/billar/internal/connectors/mcp"
	"github.com/mark3labs/mcp-go/client"
	transport "github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

func TestV1MCPRouteUsesConnectorAuthenticatedIdentity(t *testing.T) {
	t.Parallel()

	challenge := app.OAuthChallengeDTO{ResourceURI: "https://resource.example", AuthorizationServers: []string{"https://issuer.example"}}
	mux := http.NewServeMux()
	mux.Handle("/v1/mcp", NewMCPHTTPAuthMiddleware(&requestAuthenticatorStub{identity: app.AuthenticatedIdentity{Email: "person@example.com", EmailVerified: true}}, challenge, nil).Wrap(
		mcpconnector.NewServer(app.NewRequestSessionService(app.ContextIdentitySource{}), routeCustomerServiceStub{result: app.ListResult[app.CustomerDTO]{
			Items: []app.CustomerDTO{{
				ID:              "cus_123",
				Type:            "company",
				LegalName:       "Acme SRL",
				Status:          "active",
				DefaultCurrency: "USD",
				CreatedAt:       "2026-04-03T10:00:00Z",
				UpdatedAt:       "2026-04-03T10:05:00Z",
			}},
			Total:    1,
			Page:     1,
			PageSize: 20,
		}}, mcpconnector.NewIngressGuard(nil), nil).HTTPHandler(),
	))

	httpServer := httptest.NewServer(mux)
	defer httpServer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	httpTransport, err := transport.NewStreamableHTTP(httpServer.URL+"/v1/mcp", transport.WithHTTPHeaders(map[string]string{"Authorization": "Bearer token-123"}))
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

	_, err = mcpClient.Initialize(ctx, mcp.InitializeRequest{Params: mcp.InitializeParams{ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION, ClientInfo: mcp.Implementation{Name: "billar-mcphttp-route-test", Version: "1.0.0"}, Capabilities: mcp.ClientCapabilities{}}})
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	result, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{Params: mcp.CallToolParams{Name: "session.status"}})
	if err != nil {
		t.Fatalf("CallTool(session.status) error = %v", err)
	}
	if result.IsError {
		t.Fatalf("CallTool(session.status) returned tool error: %+v", result)
	}
	if got := mcp.GetTextFromContent(result.Content[0]); got != "Status: active\nEmail: person@example.com\nEmail verified: true\n" {
		t.Fatalf("CallTool(session.status) text = %q", got)
	}
}

func TestV1MCPRouteChallengesUnauthenticatedRequest(t *testing.T) {
	t.Parallel()

	challenge := app.OAuthChallengeDTO{ResourceURI: "https://resource.example", AuthorizationServers: []string{"https://issuer.example"}}
	mux := http.NewServeMux()
	mux.Handle("/v1/mcp", NewMCPHTTPAuthMiddleware(&requestAuthenticatorStub{err: app.ErrMissingBearerToken}, challenge, nil).Wrap(
		mcpconnector.NewServer(app.NewRequestSessionService(app.ContextIdentitySource{}), routeCustomerServiceStub{}, mcpconnector.NewIngressGuard(nil), nil).HTTPHandler(),
	))

	httpServer := httptest.NewServer(mux)
	defer httpServer.Close()

	resp, err := http.Post(httpServer.URL+"/v1/mcp", "application/json", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","clientInfo":{"name":"test","version":"1.0.0"},"capabilities":{}}}`))
	if err != nil {
		t.Fatalf("http.Post() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
	if got := resp.Header.Get("WWW-Authenticate"); !strings.Contains(got, "Bearer") || !strings.Contains(got, "oauth-protected-resource") {
		t.Fatalf("WWW-Authenticate = %q, want bearer challenge with metadata", got)
	}
}

type routeCustomerServiceStub struct {
	result app.ListResult[app.CustomerDTO]
}

func (s routeCustomerServiceStub) List(context.Context, app.ListQuery) (app.ListResult[app.CustomerDTO], error) {
	return s.result, nil
}
