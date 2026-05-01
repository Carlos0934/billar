package docs_test

import (
	"os"
	"strings"
	"testing"
)

func TestREADMEGlobalSetupBackupContract(t *testing.T) {
	t.Parallel()

	readme := readLower(t, "../README.md")
	for _, want := range []string{"make install", "bindir", "billar_backup_dir", "billar setup", "billar doctor", "billar backup create", "billar backup list", "billar backup restore", "--dry-run", "--confirm", "--force", "exit code", "safety snapshot", "concurrent_processes", "sensitive", "unencrypted"} {
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
	for _, forbidden := range []string{"restore is intentionally deferred", "not available/not implemented"} {
		if strings.Contains(readme, forbidden) {
			t.Fatalf("README still contains obsolete restore wording %q", forbidden)
		}
	}
}

func TestTechnicalBlueprintGlobalSetupBackupContract(t *testing.T) {
	t.Parallel()

	blueprint := readLower(t, "technical_blueprint.md")
	for _, want := range []string{"make install", "bindir", "billar setup", "billar doctor", "billar backup create", "billar backup list", ".db", ".db.json", "sensitive", "unencrypted", "restore", "deferred"} {
		if !strings.Contains(blueprint, want) {
			t.Fatalf("technical blueprint missing %q", want)
		}
	}
	requireNoBackupSQLiteArtifact(t, "technical blueprint", blueprint)
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
