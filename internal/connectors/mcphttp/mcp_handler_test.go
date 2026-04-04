package mcphttp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Carlos0934/billar/internal/app"
	mcpconnector "github.com/Carlos0934/billar/internal/connectors/mcp"
	"github.com/mark3labs/mcp-go/client"
	transport "github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

func TestV1MCPRoute(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.Handle("/v1/mcp", mcpconnector.NewServer(routeSessionServiceStub{}, routeCustomerServiceStub{result: app.ListResult[app.CustomerDTO]{
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
	}}, mcpconnector.NewIngressGuard(nil), nil).HTTPHandler())

	httpServer := httptest.NewServer(mux)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	httpTransport, err := transport.NewStreamableHTTP(httpServer.URL + "/v1/mcp")
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

	_, err = mcpClient.Initialize(ctx, mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo: mcp.Implementation{
				Name:    "billar-mcphttp-route-test",
				Version: "1.0.0",
			},
			Capabilities: mcp.ClientCapabilities{},
		},
	})
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	result, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{Params: mcp.CallToolParams{Name: "customer.list"}})
	if err != nil {
		t.Fatalf("CallTool(customer.list) error = %v", err)
	}
	if result.IsError {
		t.Fatalf("CallTool(customer.list) returned tool error: %+v", result)
	}
	if got := mcp.GetTextFromContent(result.Content[0]); got != "Billar Customers\n───────────────\nPage: 1\nPage size: 20\nTotal: 1\n\n1. Acme SRL\n   Type: company\n   Status: active\n   Default currency: USD\n   Created at: 2026-04-03T10:00:00Z\n   Updated at: 2026-04-03T10:05:00Z\n" {
		t.Fatalf("CallTool(customer.list) text = %q", got)
	}
}

type routeSessionServiceStub struct{}

func (routeSessionServiceStub) StartLogin(context.Context) (app.LoginIntentDTO, error) {
	return app.LoginIntentDTO{LoginURL: "https://login.example/test-provider"}, nil
}

func (routeSessionServiceStub) Status(context.Context) (app.SessionStatusDTO, error) {
	return app.SessionStatusDTO{Status: "unauthenticated"}, nil
}

func (routeSessionServiceStub) Logout(context.Context) (app.LogoutDTO, error) {
	return app.LogoutDTO{Message: "Logged out"}, nil
}

type routeCustomerServiceStub struct {
	result app.ListResult[app.CustomerDTO]
}

func (s routeCustomerServiceStub) List(context.Context, app.ListQuery) (app.ListResult[app.CustomerDTO], error) {
	return s.result, nil
}
