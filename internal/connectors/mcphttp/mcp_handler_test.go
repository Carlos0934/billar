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

const testAPIKey = "test-api-key-for-mcp-handler"

func newTestMCPServer() *mcpconnector.Server {
	return mcpconnector.NewServer(
		app.NewRequestSessionService(app.ContextIdentitySource{}),
		issuerProfileProviderStub{},
		customerProfileProviderStub{},
		nil,
		nil,
		nil,
		nil,
	)
}

func TestV1MCPRouteUsesConnectorAuthenticatedIdentity(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.Handle("/v1/mcp", NewAPIKeyAuthMiddleware([]string{testAPIKey}, nil).Wrap(newTestMCPServer().HTTPHandler()))

	httpServer := httptest.NewServer(mux)
	defer httpServer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	httpTransport, err := transport.NewStreamableHTTP(httpServer.URL+"/v1/mcp", transport.WithHTTPHeaders(map[string]string{"Authorization": "Bearer " + testAPIKey}))
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
	// With API key auth, synthetic identity fields are hidden; only status is shown.
	got := mcp.GetTextFromContent(result.Content[0])
	if !strings.Contains(got, "Status: active") {
		t.Fatalf("CallTool(session.status) text = %q, want to contain \"Status: active\"", got)
	}
	if strings.Contains(got, "mcp@local") || strings.Contains(got, "mcp-api-key") || strings.Contains(got, "billar://local") {
		t.Fatalf("CallTool(session.status) text = %q, synthetic identity fields must be hidden", got)
	}
}

func TestV1MCPRouteChallengesUnauthenticatedNonDiscoveryRequest(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.Handle("/v1/mcp", NewAPIKeyAuthMiddleware([]string{testAPIKey}, nil).Wrap(newTestMCPServer().HTTPHandler()))

	httpServer := httptest.NewServer(mux)
	defer httpServer.Close()

	resp, err := http.Post(httpServer.URL+"/v1/mcp", "application/json", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"session.status"}}`))
	if err != nil {
		t.Fatalf("http.Post() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestV1MCPRouteChallengesUnauthenticatedInitialize(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.Handle("/v1/mcp", NewAPIKeyAuthMiddleware([]string{testAPIKey}, nil).Wrap(newTestMCPServer().HTTPHandler()))

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
}

func TestV1MCPRouteChallengesUnauthenticatedToolsList(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.Handle("/v1/mcp", NewAPIKeyAuthMiddleware([]string{testAPIKey}, nil).Wrap(newTestMCPServer().HTTPHandler()))

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
}

func TestV1MCPRouteChallengesUnauthenticatedNotificationsInitialized(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.Handle("/v1/mcp", NewAPIKeyAuthMiddleware([]string{testAPIKey}, nil).Wrap(newTestMCPServer().HTTPHandler()))

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
