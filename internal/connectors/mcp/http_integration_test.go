package mcp

import (
	"context"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Carlos0934/billar/internal/app"
	mcphttpconnector "github.com/Carlos0934/billar/internal/connectors/mcphttp"
	"github.com/mark3labs/mcp-go/client"
	transport "github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

type authenticatedHTTPTestAuthenticator struct {
	token string

	identity app.AuthenticatedIdentity
	err      error
}

func (a authenticatedHTTPTestAuthenticator) Authenticate(_ context.Context, bearerToken string) (app.AuthenticatedIdentity, error) {
	if strings.TrimSpace(bearerToken) != strings.TrimSpace(a.token) {
		return app.AuthenticatedIdentity{}, app.ErrInvalidBearerToken
	}
	if a.err != nil {
		return app.AuthenticatedIdentity{}, a.err
	}
	return a.identity, nil
}

func TestMCPServerOverHTTPUsesRequestAuthenticatedIdentity(t *testing.T) {
	t.Parallel()

	challenge := app.OAuthChallengeDTO{ResourceURI: "https://resource.example", AuthorizationServers: []string{"https://issuer.example"}}
	server := NewServer(
		app.NewRequestSessionService(app.ContextIdentitySource{}),
		&customerWriteServiceStub{
			customerListServiceStub: customerListServiceStub{
				result: app.ListResult[app.CustomerDTO]{
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
				},
			},
		},
		NewIngressGuard(nil),
		nil,
	)
	httpServer := httptest.NewServer(mcphttpconnector.NewMCPHTTPAuthMiddleware(authenticatedHTTPTestAuthenticator{
		token:    "token-123",
		identity: app.AuthenticatedIdentity{Email: "person@example.com", EmailVerified: true, Subject: "subject-123", Issuer: "https://issuer.example"},
	}, challenge, nil).Wrap(server.HTTPHandler()))

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	httpTransport, err := transport.NewStreamableHTTP(httpServer.URL, transport.WithHTTPHeaders(map[string]string{"Authorization": "Bearer token-123"}))
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
	t.Cleanup(httpServer.Close)

	initResult, err := mcpClient.Initialize(ctx, mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo: mcp.Implementation{
				Name:    "billar-http-integration-test",
				Version: "1.0.0",
			},
			Capabilities: mcp.ClientCapabilities{},
		},
	})
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	if initResult.Capabilities.Tools == nil {
		t.Fatal("Initialize() returned no tool capability")
	}

	toolsResult, err := mcpClient.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}

	var gotToolNames []string
	for _, tool := range toolsResult.Tools {
		gotToolNames = append(gotToolNames, tool.Name)
	}
	if wantToolNames := []string{"customer.create", "customer.delete", "customer.list", "customer.update", "session.status"}; !reflect.DeepEqual(gotToolNames, wantToolNames) {
		t.Fatalf("ListTools() names = %v, want %v", gotToolNames, wantToolNames)
	}

	statusResult, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{Params: mcp.CallToolParams{Name: "session.status"}})
	if err != nil {
		t.Fatalf("CallTool(session.status) error = %v", err)
	}
	if statusResult.IsError {
		t.Fatalf("CallTool(session.status) returned tool error: %+v", statusResult)
	}
	if got := mcp.GetTextFromContent(statusResult.Content[0]); got != "Status: active\nEmail: person@example.com\nEmail verified: true\nSubject: subject-123\nIssuer: https://issuer.example\n" {
		t.Fatalf("CallTool(session.status) text = %q", got)
	}

	customerResult, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{Params: mcp.CallToolParams{Name: "customer.list"}})
	if err != nil {
		t.Fatalf("CallTool(customer.list) error = %v", err)
	}
	if customerResult.IsError {
		t.Fatalf("CallTool(customer.list) returned tool error: %+v", customerResult)
	}
	if got := mcp.GetTextFromContent(customerResult.Content[0]); got != "Billar Customers\n───────────────\nPage: 1\nPage size: 20\nTotal: 1\n\n1. Acme SRL\n   Type: company\n   Status: active\n   Default currency: USD\n   Created at: 2026-04-03T10:00:00Z\n   Updated at: 2026-04-03T10:05:00Z\n" {
		t.Fatalf("CallTool(customer.list) text = %q", got)
	}
}

func TestMCPServerOverHTTPRejectsUnauthenticatedRequests(t *testing.T) {
	t.Parallel()

	challenge := app.OAuthChallengeDTO{ResourceURI: "https://resource.example", AuthorizationServers: []string{"https://issuer.example"}}
	server := NewServer(app.NewRequestSessionService(app.ContextIdentitySource{}), &customerWriteServiceStub{}, NewIngressGuard(nil), nil)
	httpServer := httptest.NewServer(mcphttpconnector.NewMCPHTTPAuthMiddleware(authenticatedHTTPTestAuthenticator{token: "token-123"}, challenge, nil).Wrap(server.HTTPHandler()))
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	httpTransport, err := transport.NewStreamableHTTP(httpServer.URL)
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

	_, err = mcpClient.Initialize(ctx, mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo:      mcp.Implementation{Name: "billar-http-unauth-test", Version: "1.0.0"},
			Capabilities:    mcp.ClientCapabilities{},
		},
	})
	if err == nil {
		t.Fatal("Initialize() error = nil, want unauthorized error")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Fatalf("Initialize() error = %v, want 401", err)
	}
}
