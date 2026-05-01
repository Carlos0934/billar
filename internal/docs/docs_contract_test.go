package docs

import (
	"os"
	"strings"
	"testing"
)

func TestCLIFirstBillingPolicyDocumentationContract(t *testing.T) {
	t.Parallel()

	docs := map[string]string{
		"README.md":                   "../../README.md",
		"docs/technical_blueprint.md": "../../docs/technical_blueprint.md",
		"docs/import.md":              "../../docs/import.md",
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
				"app",
				"trusted",
				"direct sqlite",
				"emergency repair-only",
				"explicit operator approval",
				"mcp",
				"not trusted",
			})
		})
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
