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
		[]string{"OAUTH_PROVIDER=test-provider", "BILLAR_SESSION_EMAIL=integration@example.com"},
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
	if got := mcp.GetTextFromContent(customerResult.Content[0]); got != "Billar Customers\n───────────────\nPage: 1\nPage size: 20\nTotal: 0\nNo customers found\n" {
		t.Fatalf("CallTool(customer.list) text = %q", got)
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
	if got := mcp.GetTextFromContent(statusResult.Content[0]); got != "Status: active\nEmail: integration@example.com\nEmail verified: true\n" {
		t.Fatalf("CallTool(session.status) text = %q, want %q", got, "Status: active\nEmail: integration@example.com\nEmail verified: true\n")
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
