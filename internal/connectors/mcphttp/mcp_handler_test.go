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
		mcpconnector.NewServer(
			app.NewRequestSessionService(app.ContextIdentitySource{}),
			issuerProfileProviderStub{},
			customerProfileProviderStub{},
			nil,
			nil,
			nil,
			mcpconnector.NewIngressGuard(nil),
			nil,
		).HTTPHandler(),
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

func TestV1MCPRouteChallengesUnauthenticatedNonDiscoveryRequest(t *testing.T) {
	t.Parallel()

	challenge := app.OAuthChallengeDTO{ResourceURI: "https://resource.example", AuthorizationServers: []string{"https://issuer.example"}}
	mux := http.NewServeMux()
	mux.Handle("/v1/mcp", NewMCPHTTPAuthMiddleware(&requestAuthenticatorStub{err: app.ErrMissingBearerToken}, challenge, nil).Wrap(
		mcpconnector.NewServer(
			app.NewRequestSessionService(app.ContextIdentitySource{}),
			issuerProfileProviderStub{},
			customerProfileProviderStub{},
			nil,
			nil,
			nil,
			mcpconnector.NewIngressGuard(nil),
			nil,
		).HTTPHandler(),
	))

	httpServer := httptest.NewServer(mux)
	defer httpServer.Close()

	// Use a non-allowlisted method (tools/call) to test authentication challenge
	resp, err := http.Post(httpServer.URL+"/v1/mcp", "application/json", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"session.status"}}`))
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

func TestV1MCPRouteChallengesUnauthenticatedInitialize(t *testing.T) {
	t.Parallel()

	challenge := app.OAuthChallengeDTO{ResourceURI: "https://resource.example", AuthorizationServers: []string{"https://issuer.example"}}
	mux := http.NewServeMux()
	mux.Handle("/v1/mcp", NewMCPHTTPAuthMiddleware(&requestAuthenticatorStub{err: app.ErrMissingBearerToken}, challenge, nil).Wrap(
		mcpconnector.NewServer(
			app.NewRequestSessionService(app.ContextIdentitySource{}),
			issuerProfileProviderStub{},
			customerProfileProviderStub{},
			nil,
			nil,
			nil,
			mcpconnector.NewIngressGuard(nil),
			nil,
		).HTTPHandler(),
	))

	httpServer := httptest.NewServer(mux)
	defer httpServer.Close()

	// All unauthenticated requests including initialize MUST return 401 with WWW-Authenticate
	resp, err := http.Post(httpServer.URL+"/v1/mcp", "application/json", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","clientInfo":{"name":"test","version":"1.0.0"},"capabilities":{}}}`))
	if err != nil {
		t.Fatalf("http.Post() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
	if got := resp.Header.Get("WWW-Authenticate"); !strings.Contains(got, "Bearer") || !strings.Contains(got, "oauth-protected-resource") {
		t.Fatalf("WWW-Authenticate = %q, want bearer challenge with resource_metadata", got)
	}
}

func TestV1MCPRouteChallengesUnauthenticatedToolsList(t *testing.T) {
	t.Parallel()

	challenge := app.OAuthChallengeDTO{ResourceURI: "https://resource.example", AuthorizationServers: []string{"https://issuer.example"}}
	mux := http.NewServeMux()
	mux.Handle("/v1/mcp", NewMCPHTTPAuthMiddleware(&requestAuthenticatorStub{err: app.ErrMissingBearerToken}, challenge, nil).Wrap(
		mcpconnector.NewServer(
			app.NewRequestSessionService(app.ContextIdentitySource{}),
			issuerProfileProviderStub{},
			customerProfileProviderStub{},
			nil,
			nil,
			nil,
			mcpconnector.NewIngressGuard(nil),
			nil,
		).HTTPHandler(),
	))

	httpServer := httptest.NewServer(mux)
	defer httpServer.Close()

	resp, err := http.Post(httpServer.URL+"/v1/mcp", "application/json", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`))
	if err != nil {
		t.Fatalf("http.Post() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d (all unauthenticated requests must return 401)", resp.StatusCode, http.StatusUnauthorized)
	}
	if got := resp.Header.Get("WWW-Authenticate"); !strings.Contains(got, "Bearer") {
		t.Fatalf("WWW-Authenticate = %q, want bearer challenge", got)
	}
}

func TestV1MCPRouteChallengesUnauthenticatedNotificationsInitialized(t *testing.T) {
	t.Parallel()

	challenge := app.OAuthChallengeDTO{ResourceURI: "https://resource.example", AuthorizationServers: []string{"https://issuer.example"}}
	mux := http.NewServeMux()
	mux.Handle("/v1/mcp", NewMCPHTTPAuthMiddleware(&requestAuthenticatorStub{err: app.ErrMissingBearerToken}, challenge, nil).Wrap(
		mcpconnector.NewServer(
			app.NewRequestSessionService(app.ContextIdentitySource{}),
			issuerProfileProviderStub{},
			customerProfileProviderStub{},
			nil,
			nil,
			nil,
			mcpconnector.NewIngressGuard(nil),
			nil,
		).HTTPHandler(),
	))

	httpServer := httptest.NewServer(mux)
	defer httpServer.Close()

	resp, err := http.Post(httpServer.URL+"/v1/mcp", "application/json", strings.NewReader(`{"jsonrpc":"2.0","method":"notifications/initialized","params":{}}`))
	if err != nil {
		t.Fatalf("http.Post() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d (all unauthenticated requests must return 401)", resp.StatusCode, http.StatusUnauthorized)
	}
	if got := resp.Header.Get("WWW-Authenticate"); !strings.Contains(got, "Bearer") {
		t.Fatalf("WWW-Authenticate = %q, want bearer challenge", got)
	}
}

// Stub implementations

type issuerProfileProviderStub struct{}

func (s issuerProfileProviderStub) Create(ctx context.Context, cmd app.CreateIssuerProfileCommand) (app.IssuerProfileDTO, error) {
	return app.IssuerProfileDTO{}, nil
}

func (s issuerProfileProviderStub) Get(ctx context.Context, id string) (app.IssuerProfileDTO, error) {
	return app.IssuerProfileDTO{}, nil
}

func (s issuerProfileProviderStub) Update(ctx context.Context, id string, cmd app.PatchIssuerProfileCommand) (app.IssuerProfileDTO, error) {
	return app.IssuerProfileDTO{}, nil
}

func (s issuerProfileProviderStub) Delete(ctx context.Context, id string) error {
	return nil
}

type customerProfileProviderStub struct{}

func (s customerProfileProviderStub) List(ctx context.Context, query app.ListQuery) (app.ListResult[app.CustomerProfileDTO], error) {
	return app.ListResult[app.CustomerProfileDTO]{}, nil
}

func (s customerProfileProviderStub) Create(ctx context.Context, cmd app.CreateCustomerProfileCommand) (app.CustomerProfileDTO, error) {
	return app.CustomerProfileDTO{}, nil
}

func (s customerProfileProviderStub) Get(ctx context.Context, id string) (app.CustomerProfileDTO, error) {
	return app.CustomerProfileDTO{}, nil
}

func (s customerProfileProviderStub) Update(ctx context.Context, id string, cmd app.PatchCustomerProfileCommand) (app.CustomerProfileDTO, error) {
	return app.CustomerProfileDTO{}, nil
}

func (s customerProfileProviderStub) Delete(ctx context.Context, id string) error {
	return nil
}
