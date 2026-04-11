package mcp

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/client"
	transport "github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

func TestMCPServerOverStdio(t *testing.T) {
	repoRoot := repoRoot(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)

	stdio := transport.NewStdioWithOptions(
		"go",
		[]string{"OAUTH_PROVIDER=test-provider", "BILLAR_LOCAL_AUTH_EMAIL=integration@example.com"},
		[]string{"run", "./cmd/mcp"},
		transport.WithCommandFunc(func(ctx context.Context, command string, env []string, args []string) (*exec.Cmd, error) {
			cmd := exec.CommandContext(ctx, command, args...)
			cmd.Env = append(os.Environ(), env...)
			cmd.Dir = repoRoot
			return cmd, nil
		}),
	)

	mcpClient := client.NewClient(stdio)
	if err := mcpClient.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	t.Cleanup(func() {
		if err := mcpClient.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
		cancel()
	})

	initResult, err := mcpClient.Initialize(ctx, mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo: mcp.Implementation{
				Name:    "billar-stdio-integration-test",
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
	wantToolNames := []string{
		"customer_profile.create",
		"customer_profile.delete",
		"customer_profile.get",
		"customer_profile.list",
		"customer_profile.update",
		"issuer_profile.create",
		"issuer_profile.delete",
		"issuer_profile.get",
		"issuer_profile.update",
		"service_agreement.activate",
		"service_agreement.create",
		"service_agreement.deactivate",
		"service_agreement.get",
		"service_agreement.list_by_customer_profile",
		"service_agreement.update_rate",
		"session.status",
	}
	if !reflect.DeepEqual(gotToolNames, wantToolNames) {
		t.Fatalf("ListTools() names = %v, want %v", gotToolNames, wantToolNames)
	}

	customerResult, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{Name: "customer_profile.list"},
	})
	if err != nil {
		t.Fatalf("CallTool(customer_profile.list) error = %v", err)
	}
	if customerResult.IsError {
		t.Fatalf("CallTool(customer_profile.list) returned tool error: %+v", customerResult)
	}
	if len(customerResult.Content) != 1 {
		t.Fatalf("CallTool(customer_profile.list) content = %d items, want 1", len(customerResult.Content))
	}
	if got := mcp.GetTextFromContent(customerResult.Content[0]); got != "Billar Customer Profiles\n───────────────\nPage: 1\nPage size: 20\nTotal: 0\nNo customer profiles found\n" {
		t.Fatalf("CallTool(customer_profile.list) text = %q", got)
	}

	statusResult, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{Name: "session.status"},
	})
	if err != nil {
		t.Fatalf("CallTool(session.status) error = %v", err)
	}
	if statusResult.IsError {
		t.Fatalf("CallTool(session.status) returned tool error: %+v", statusResult)
	}
	if len(statusResult.Content) != 1 {
		t.Fatalf("CallTool(session.status) content = %d items, want 1", len(statusResult.Content))
	}
	if got := mcp.GetTextFromContent(statusResult.Content[0]); got != "Status: active\nEmail: integration@example.com\nEmail verified: true\nSubject: local-bypass\nIssuer: billar://local\n" {
		t.Fatalf("CallTool(session.status) text = %q, want %q", got, "Status: active\nEmail: integration@example.com\nEmail verified: true\nSubject: local-bypass\nIssuer: billar://local\n")
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}

	return filepath.Clean(filepath.Join(wd, "..", "..", ".."))
}
