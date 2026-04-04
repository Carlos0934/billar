package mcp

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/Carlos0934/billar/internal/app"
	"github.com/mark3labs/mcp-go/client"
	transport "github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	"net/http/httptest"
)

func TestMCPServerOverHTTP(t *testing.T) {
	t.Parallel()

	server := NewServer(noopHTTPTestSessionService{}, &customerListServiceStub{result: app.ListResult[app.CustomerDTO]{
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
	}}, NewIngressGuard(nil), nil)
	httpServer := httptest.NewServer(server.HTTPHandler())

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
	wantToolNames := []string{"customer.list", "session.status"}
	if !reflect.DeepEqual(gotToolNames, wantToolNames) {
		t.Fatalf("ListTools() names = %v, want %v", gotToolNames, wantToolNames)
	}

	customerResult, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{Name: "customer.list"},
	})
	if err != nil {
		t.Fatalf("CallTool(customer.list) error = %v", err)
	}
	if customerResult.IsError {
		t.Fatalf("CallTool(customer.list) returned tool error: %+v", customerResult)
	}
	if len(customerResult.Content) != 1 {
		t.Fatalf("CallTool(customer.list) content = %d items, want 1", len(customerResult.Content))
	}
	if got := mcp.GetTextFromContent(customerResult.Content[0]); got != "Billar Customers\n───────────────\nPage: 1\nPage size: 20\nTotal: 1\n\n1. Acme SRL\n   Type: company\n   Status: active\n   Default currency: USD\n   Created at: 2026-04-03T10:00:00Z\n   Updated at: 2026-04-03T10:05:00Z\n" {
		t.Fatalf("CallTool(customer.list) text = %q", got)
	}
}

type noopHTTPTestSessionService struct {
}

func (noopHTTPTestSessionService) StartLogin(context.Context) (app.LoginIntentDTO, error) {
	return app.LoginIntentDTO{}, nil
}

func (noopHTTPTestSessionService) Status(context.Context) (app.SessionStatusDTO, error) {
	return app.SessionStatusDTO{Status: "unauthenticated"}, nil
}

func (noopHTTPTestSessionService) Logout(context.Context) (app.LogoutDTO, error) {
	return app.LogoutDTO{Message: "Logged out"}, nil
}
