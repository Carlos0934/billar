package sqlite

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenCreatesDatabaseAndRunsSchema(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "billar.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	})

	if store.DB() == nil {
		t.Fatal("DB() = nil, want database handle")
	}

	if err := assertLegalEntitiesTableExists(store.DB()); err != nil {
		t.Fatal(err)
	}
}

func TestOpenUsesTempDatabaseWhenPathIsBlank(t *testing.T) {
	t.Parallel()

	store, err := Open("")
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	})

	if err := assertLegalEntitiesTableExists(store.DB()); err != nil {
		t.Fatal(err)
	}
}

func TestOpenCreatesParentDirectoryWithUserOnlyPermissions(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "data", "billar")
	store, err := Open(filepath.Join(dir, "billar.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	})

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("Stat(%q) error = %v", dir, err)
	}
	if got := info.Mode().Perm(); got != 0o700 {
		t.Fatalf("created directory mode = %o, want 700", got)
	}
}

func assertLegalEntitiesTableExists(db *sql.DB) error {
	var name string
	return db.QueryRowContext(context.Background(), `SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'legal_entities'`).Scan(&name)
}

func newTestStore(t *testing.T) *Store {
	t.Helper()

	store, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	})
	return store
}
