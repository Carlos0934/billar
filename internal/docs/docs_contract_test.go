package docs

import (
	"os"
	"strings"
	"testing"
)

func TestCLISoleInterfaceDocumentationContract(t *testing.T) {
	t.Parallel()

	docs := map[string]string{
		"README.md":                   "../../README.md",
		"docs/technical_blueprint.md": "../../docs/technical_blueprint.md",
	}

	for name, path := range docs {
		name, path := name, path
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			contentBytes, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read documentation contract source: %v", err)
			}
			content := strings.ToLower(string(contentBytes))

			assertContainsAll(t, content, []string{
				"cli",
				"billar health",
				"billar doctor",
				"billar invoice import --file",
				"billar invoice pdf",
			})
			assertContainsNone(t, content, []string{
				"opencode.json",
				"mcp_api_keys",
				"mcp_http_listen_addr",
				"run-mcp-http",
				"migration: mcp tools",
				"/v1/mcp",
				"authorization: bearer",
			})
		})
	}
}

func TestCLIFirstBillingPolicyDocumentationContract(t *testing.T) {
	t.Parallel()

	docs := map[string]string{
		"README.md":        "../../README.md",
		"docs/invoices.md": "../../docs/invoices.md",
	}

	for name, path := range docs {
		name, path := name, path
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			contentBytes, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read documentation contract source: %v", err)
			}
			content := strings.ToLower(string(contentBytes))

			assertContainsAll(t, content, []string{
				"cli",
				"direct sqlite",
				"emergency repair-only",
				"explicit operator approval",
			})
		})
	}
}

func TestStaleOperationalDocumentationIsForbidden(t *testing.T) {
	t.Parallel()

	docs := map[string]string{
		"README.md":                   "../../README.md",
		"docs/technical_blueprint.md": "../../docs/technical_blueprint.md",
		"docs/operations.md":          "../../docs/operations.md",
		"docs/invoices.md":            "../../docs/invoices.md",
		"docs/import.md":              "../../docs/import.md",
		"AGENTS.md":                   "../../AGENTS.md",
	}

	for name, path := range docs {
		name, path := name, path
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			contentBytes, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read documentation contract source %s: %v", name, err)
			}
			content := strings.ToLower(string(contentBytes))

			assertContainsNone(t, content, []string{
				"opencode.json",
				"mcp_api_keys",
				"mcp_http_listen_addr",
				"run-mcp-http",
				"migration: mcp tools",
				"/v1/mcp",
				"authorization: bearer",
				"restore is intentionally deferred",
				"restore is deferred",
			})
		})
	}
}

func TestAgentGuidanceContract(t *testing.T) {
	t.Parallel()

	contentBytes, err := os.ReadFile("../../AGENTS.md")
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	content := strings.ToLower(string(contentBytes))

	assertContainsAll(t, content, []string{
		"cli-only",
		"makefile",
		"cmd/cli/main.go",
		"internal/infra/config",
		"internal/core",
		"internal/app",
		"sqlite",
		"do not access sqlite from `internal/core`",
		"do not let cli code bypass `internal/app`",
		".atl/",
		".env",
		"go.work*",
		"skills-lock.json",
	})
}

func assertContainsNone(t *testing.T, content string, forbidden []string) {
	t.Helper()

	for _, phrase := range forbidden {
		if strings.Contains(content, phrase) {
			t.Fatalf("documentation contract contains forbidden %q", phrase)
		}
	}
}

func assertContainsAll(t *testing.T, content string, required []string) {
	t.Helper()

	for _, phrase := range required {
		if !strings.Contains(content, phrase) {
			t.Fatalf("documentation contract missing %q", phrase)
		}
	}
}
