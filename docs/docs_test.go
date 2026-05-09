package docs_test

import (
	"os"
	"strings"
	"testing"
)

func TestREADMEGlobalSetupBackupContract(t *testing.T) {
	t.Parallel()

	readme := readLower(t, "../README.md")
	for _, want := range []string{"make install", "bindir", "billar_backup_dir", "billar setup", "billar doctor", "billar health", "billar invoice import --file", "billar invoice pdf", "billar backup create", "billar backup list", "billar backup restore", "time-entry", "agreement update-rate", "--customer-id", "--dry-run", "--confirm", "--force", "exit code", "safety snapshot", "concurrent_processes", "sensitive", "unencrypted"} {
		if !strings.Contains(readme, want) {
			t.Fatalf("README missing %q", want)
		}
	}
	for _, want := range []string{".db", ".db.json"} {
		if !strings.Contains(readme, want) {
			t.Fatalf("README must document backup artifact %q", want)
		}
	}
	requireNoBackupSQLiteArtifact(t, "README", readme)
	if !strings.Contains(readme, ".env") || !strings.Contains(readme, "current working directory") {
		t.Fatalf("README must document cwd-only .env behavior")
	}
	for _, forbidden := range []string{"restore is intentionally deferred", "restore is deferred", "not available", "not implemented"} {
		if strings.Contains(readme, forbidden) {
			t.Fatalf("README still contains obsolete restore wording %q", forbidden)
		}
	}
}

func TestTechnicalBlueprintArchitectureContract(t *testing.T) {
	t.Parallel()

	blueprint := readLower(t, "technical_blueprint.md")
	for _, want := range []string{"cli", "billar health", "billar doctor", "billar invoice import --file", "billar invoice pdf", "internal/core", "internal/app", "internal/connectors/cli", "internal/infra", "storage", "auth", "rendering", "restore implemented", "docs/operations.md", "docs/invoices.md"} {
		if !strings.Contains(blueprint, want) {
			t.Fatalf("technical blueprint missing %q", want)
		}
	}
	requireNoBackupSQLiteArtifact(t, "technical blueprint", blueprint)
	for _, forbidden := range []string{"restore is intentionally deferred", "restore is deferred", "not available", "not implemented"} {
		if strings.Contains(blueprint, forbidden) {
			t.Fatalf("technical blueprint still contains obsolete wording %q", forbidden)
		}
	}
}

func TestOperationsRunbookContract(t *testing.T) {
	t.Parallel()

	operations := readLower(t, "operations.md")
	for _, want := range []string{"billar setup", "billar doctor", "billar backup restore", "billar_db_path", "billar_export_dir", "billar_backup_dir", "--dry-run", "--confirm", "--force", "exit code", "safety snapshot", "concurrent_processes", "sensitive", "unencrypted"} {
		if !strings.Contains(operations, want) {
			t.Fatalf("operations doc missing %q", want)
		}
	}
}

func TestEnvExampleContract(t *testing.T) {
	t.Parallel()

	envExample := readLower(t, "../.env.example")
	for _, want := range []string{"optional override", "cwd-only `.env`", "existing non-empty environment variables win", "billar setup", "does not write `.env`", "billar_db_path", "billar_export_dir", "billar_backup_dir", "<db-parent>/exports", "<db-parent>/backups"} {
		if !strings.Contains(envExample, want) {
			t.Fatalf(".env.example missing %q", want)
		}
	}
}

func readLower(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return strings.ToLower(string(data))
}

func requireNoBackupSQLiteArtifact(t *testing.T, name, doc string) {
	t.Helper()

	for _, line := range strings.Split(doc, "\n") {
		if strings.Contains(line, "backup") && strings.Contains(line, ".sqlite") {
			if strings.Contains(line, "not .sqlite") || strings.Contains(line, "not `.sqlite`") || strings.Contains(line, "does not create .sqlite") || strings.Contains(line, "does not create `.sqlite`") || strings.Contains(line, ".sqlite` backup filenames are not") || strings.Contains(line, "no .sqlite") {
				continue
			}
			t.Fatalf("%s must not document .sqlite backup artifacts; backups use .db plus .db.json sidecars", name)
		}
	}
}
