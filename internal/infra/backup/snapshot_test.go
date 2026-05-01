package backup

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestSnapshotterCreateWritesConsistentSQLiteBackupAndSidecar(t *testing.T) {
	dbPath := createSQLiteSource(t)
	destDir := filepath.Join(t.TempDir(), "backups")
	now := time.Date(2026, 5, 1, 16, 30, 0, 0, time.UTC)

	record, err := Snapshotter{Now: func() time.Time { return now }}.Create(context.Background(), dbPath, destDir)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	wantFile := filepath.Join(destDir, "billar-20260501T163000Z-schema1.db")
	if record.ID != "billar-20260501T163000Z-schema1" {
		t.Fatalf("Record.ID = %q, want timestamp/schema id", record.ID)
	}
	if record.File != wantFile {
		t.Fatalf("Record.File = %q, want %q", record.File, wantFile)
	}
	if record.SidecarFile != wantFile+".json" {
		t.Fatalf("Record.SidecarFile = %q, want sidecar next to db", record.SidecarFile)
	}
	if record.CreatedAt != now {
		t.Fatalf("Record.CreatedAt = %s, want %s", record.CreatedAt, now)
	}
	if record.SourceDBPath != dbPath {
		t.Fatalf("Record.SourceDBPath = %q, want %q", record.SourceDBPath, dbPath)
	}
	if record.SchemaVersion != 1 {
		t.Fatalf("Record.SchemaVersion = %d, want 1", record.SchemaVersion)
	}
	if record.SizeBytes <= 0 {
		t.Fatalf("Record.SizeBytes = %d, want non-zero", record.SizeBytes)
	}
	if record.SHA256 != fileSHA256(t, wantFile) {
		t.Fatalf("Record.SHA256 = %q, want hash of backup file", record.SHA256)
	}
	if !record.Metadata {
		t.Fatal("Record.Metadata = false, want true for sidecar-backed snapshot")
	}

	assertPerm(t, destDir, 0o700)
	assertPerm(t, wantFile, 0o600)
	assertPerm(t, wantFile+".json", 0o600)
	assertSQLiteIntegrity(t, wantFile)

	listed, err := Lister{}.List(context.Background(), destDir)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(listed) != 1 || listed[0].SHA256 != record.SHA256 {
		t.Fatalf("List() = %+v, want saved record", listed)
	}
}

func TestSnapshotterCreateAvoidsCollisionsAndCleansUpOnFailure(t *testing.T) {
	dbPath := createSQLiteSource(t)
	now := time.Date(2026, 5, 1, 16, 30, 0, 0, time.UTC)

	t.Run("existing snapshot id gets collision-resistant suffix", func(t *testing.T) {
		destDir := t.TempDir()

		first, err := Snapshotter{Now: func() time.Time { return now }}.Create(context.Background(), dbPath, destDir)
		if err != nil {
			t.Fatalf("first Create() error = %v", err)
		}
		second, err := Snapshotter{Now: func() time.Time { return now }}.Create(context.Background(), dbPath, destDir)
		if err != nil {
			t.Fatalf("second Create() error = %v", err)
		}
		if first.ID == second.ID || first.File == second.File {
			t.Fatalf("Create() IDs/files collided: first=%+v second=%+v", first, second)
		}
		if !strings.HasPrefix(second.ID, "billar-20260501T163000Z-schema1-") {
			t.Fatalf("second ID = %q, want timestamp/schema id with collision suffix", second.ID)
		}
		assertSQLiteIntegrity(t, first.File)
		assertSQLiteIntegrity(t, second.File)
	})

	t.Run("vacuum failure removes temp and final files", func(t *testing.T) {
		destDir := t.TempDir()
		boom := errors.New("vacuum failed")
		snap := Snapshotter{
			Now:  func() time.Time { return now },
			Open: func(string) (vacuumDB, error) { return failingVacuumDB{err: boom}, nil },
		}

		_, err := snap.Create(context.Background(), dbPath, destDir)
		if err == nil || !strings.Contains(err.Error(), boom.Error()) {
			t.Fatalf("Create() error = %v, want wrapped vacuum error", err)
		}
		entries, err := os.ReadDir(destDir)
		if err != nil {
			t.Fatalf("ReadDir() error = %v", err)
		}
		if len(entries) != 0 {
			t.Fatalf("dest dir entries = %v, want cleanup of db/tmp/sidecar", entries)
		}
	})
}

type failingVacuumDB struct{ err error }

func (f failingVacuumDB) ExecContext(context.Context, string, ...any) (sql.Result, error) {
	return nil, f.err
}
func (f failingVacuumDB) Close() error { return nil }

func createSQLiteSource(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "source.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`CREATE TABLE schema_migrations (version INTEGER NOT NULL); INSERT INTO schema_migrations(version) VALUES (1); CREATE TABLE invoices (id TEXT PRIMARY KEY); INSERT INTO invoices(id) VALUES ('inv_1')`); err != nil {
		t.Fatalf("seed sqlite source: %v", err)
	}
	return path
}

func assertSQLiteIntegrity(t *testing.T, path string) {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+path+"?mode=ro")
	if err != nil {
		t.Fatalf("open snapshot: %v", err)
	}
	defer db.Close()
	var got string
	if err := db.QueryRow("PRAGMA integrity_check").Scan(&got); err != nil {
		t.Fatalf("integrity_check query: %v", err)
	}
	if got != "ok" {
		t.Fatalf("integrity_check = %q, want ok", got)
	}
}

func assertPerm(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%q) error = %v", path, err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("%q mode = %o, want %o", path, got, want)
	}
}

func fileSHA256(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
