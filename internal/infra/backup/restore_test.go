package backup

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	infrasqlite "github.com/Carlos0934/billar/internal/infra/sqlite"

	_ "modernc.org/sqlite"
)

func TestRestorerResolveAndValidateRejectsInvalidSources(t *testing.T) {
	ctx := context.Background()
	restorer := Restorer{}

	t.Run("orphan db", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "orphan.db")
		writeFile(t, path, "not sqlite")
		_, err := restorer.Resolve(ctx, RestoreRequest{File: path}, t.TempDir())
		if err == nil || !strings.Contains(err.Error(), "sidecar_missing") {
			t.Fatalf("Resolve(orphan) error = %v, want sidecar_missing", err)
		}
	})

	t.Run("sidecar parse error", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "bad.db")
		writeFile(t, path, "not sqlite")
		writeFile(t, path+".json", "{")
		_, err := restorer.Resolve(ctx, RestoreRequest{File: path}, t.TempDir())
		if err == nil || !strings.Contains(err.Error(), "sidecar_parse_error") {
			t.Fatalf("Resolve(bad sidecar) error = %v, want sidecar_parse_error", err)
		}
	})

	t.Run("id basename mismatch", func(t *testing.T) {
		path, record := createBillarBackupFixture(t, 1)
		record.ID = "different-id"
		writeJSON(t, path+".json", record)
		_, err := restorer.Resolve(ctx, RestoreRequest{File: path}, t.TempDir())
		if err == nil || !strings.Contains(err.Error(), "id_basename_mismatch") {
			t.Fatalf("Resolve(id mismatch) error = %v, want id_basename_mismatch", err)
		}
	})

	t.Run("size mismatch", func(t *testing.T) {
		path, record := createBillarBackupFixture(t, 1)
		record.SizeBytes++
		writeJSON(t, path+".json", record)
		record, err := restorer.Resolve(ctx, RestoreRequest{File: path}, t.TempDir())
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}
		_, err = restorer.Validate(ctx, record, filepath.Join(t.TempDir(), "live.db"), 4)
		if err == nil || !strings.Contains(err.Error(), "size_mismatch") {
			t.Fatalf("Validate(size mismatch) error = %v, want size_mismatch", err)
		}
	})

	t.Run("sha256 mismatch", func(t *testing.T) {
		path, record := createBillarBackupFixture(t, 1)
		record.SHA256 = strings.Repeat("0", 64)
		writeJSON(t, path+".json", record)
		record, err := restorer.Resolve(ctx, RestoreRequest{File: path}, t.TempDir())
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}
		_, err = restorer.Validate(ctx, record, filepath.Join(t.TempDir(), "live.db"), 4)
		if err == nil || !strings.Contains(err.Error(), "hash_mismatch") {
			t.Fatalf("Validate(hash mismatch) error = %v, want hash_mismatch", err)
		}
	})

	t.Run("schema newer than binary", func(t *testing.T) {
		path, record := createBillarBackupFixture(t, 5)
		record, err := restorer.Resolve(ctx, RestoreRequest{File: path}, t.TempDir())
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}
		_, err = restorer.Validate(ctx, record, filepath.Join(t.TempDir(), "live.db"), 4)
		if err == nil || !strings.Contains(err.Error(), "schema_newer_than_binary") {
			t.Fatalf("Validate(newer schema) error = %v, want schema_newer_than_binary", err)
		}
	})

	t.Run("missing required Billar tables", func(t *testing.T) {
		path := createSQLiteWithTables(t, 1, []string{"invoices"})
		record := writeFixtureSidecar(t, path, 1)
		_, err := restorer.Validate(ctx, record, filepath.Join(t.TempDir(), "live.db"), 4)
		if err == nil || !strings.Contains(err.Error(), "missing_required_tables") {
			t.Fatalf("Validate(non-Billar DB) error = %v, want missing_required_tables", err)
		}
	})

	t.Run("happy validation", func(t *testing.T) {
		path, record := createBillarBackupFixture(t, 2)
		record, err := restorer.Resolve(ctx, RestoreRequest{File: path}, t.TempDir())
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}
		got, err := restorer.Validate(ctx, record, filepath.Join(t.TempDir(), "live.db"), 4)
		if err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
		if !got.OK || !got.SidecarOK || !got.SizeOK || !got.HashOK || !got.IntegrityOK || !got.TablesOK || !got.SchemaOK || got.BackupSchema != 2 || got.BinarySchema != 4 {
			t.Fatalf("Validate() = %+v, want all checks ok with schema values", got)
		}
	})
}

func TestRestorerReplaceCopiesAtomicallyAndPreservesSource(t *testing.T) {
	ctx := context.Background()
	restorer := Restorer{}
	t.Run("target absent creates secure replacement", func(t *testing.T) {
		path, record := createBillarBackupFixture(t, 1)
		sourceHash := fileSHA256(t, path)
		target := filepath.Join(t.TempDir(), "data", "billar.db")

		if err := restorer.Replace(ctx, record, target); err != nil {
			t.Fatalf("Replace() error = %v", err)
		}
		if got := fileSHA256(t, target); got != sourceHash {
			t.Fatalf("target hash = %s, want source hash %s", got, sourceHash)
		}
		if got := fileSHA256(t, path); got != sourceHash {
			t.Fatalf("source hash after Replace = %s, want unchanged %s", got, sourceHash)
		}
		assertPerm(t, target, 0o600)
		assertSQLiteIntegrity(t, target)
	})

	t.Run("existing target remains untouched when temp rehash fails", func(t *testing.T) {
		path, record := createBillarBackupFixture(t, 1)
		target := createSQLiteWithTables(t, 1, infrasqlite.RequiredBillarTables)
		before := fileSHA256(t, target)
		restorer := Restorer{AfterCopy: func(tempPath string) error {
			return os.WriteFile(tempPath, []byte("tampered"), 0o600)
		}}

		err := restorer.Replace(ctx, record, target)
		if err == nil || !strings.Contains(err.Error(), "temp_hash_mismatch") {
			t.Fatalf("Replace(tampered temp) error = %v, want temp_hash_mismatch", err)
		}
		if got := fileSHA256(t, target); got != before {
			t.Fatalf("target hash after failed Replace = %s, want original %s", got, before)
		}
		if got := fileSHA256(t, path); got != record.SHA256 {
			t.Fatalf("source hash after failed Replace = %s, want original %s", got, record.SHA256)
		}
	})
}

func createBillarBackupFixture(t *testing.T, schemaVersion int) (string, Record) {
	t.Helper()
	path := createSQLiteWithTables(t, schemaVersion, infrasqlite.RequiredBillarTables)
	return path, writeFixtureSidecar(t, path, schemaVersion)
}

func createSQLiteWithTables(t *testing.T, schemaVersion int, tables []string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "billar-20260501T170000Z-schema.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE schema_migrations (version INTEGER NOT NULL); INSERT INTO schema_migrations(version) VALUES (?)`, schemaVersion); err != nil {
		t.Fatalf("create schema_migrations: %v", err)
	}
	for _, name := range tables {
		if _, err := db.Exec(`CREATE TABLE ` + name + ` (id TEXT PRIMARY KEY)`); err != nil {
			t.Fatalf("create table %s: %v", name, err)
		}
	}
	return path
}

func writeFixtureSidecar(t *testing.T, path string, schemaVersion int) Record {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	record := Record{
		ID:            strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
		File:          path,
		SidecarFile:   path + ".json",
		CreatedAt:     time.Date(2026, 5, 1, 17, 0, 0, 0, time.UTC),
		SchemaVersion: schemaVersion,
		SizeBytes:     info.Size(),
		SHA256:        fileSHA256(t, path),
		SourceDBPath:  filepath.Join(filepath.Dir(path), "live.db"),
		Metadata:      true,
	}
	writeJSON(t, path+".json", record)
	return record
}
