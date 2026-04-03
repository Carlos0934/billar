package sqlite

import (
	"context"
	"database/sql"
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

	if err := assertCustomersTableExists(store.DB()); err != nil {
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

	if err := assertCustomersTableExists(store.DB()); err != nil {
		t.Fatal(err)
	}
}

func assertCustomersTableExists(db *sql.DB) error {
	var name string
	return db.QueryRowContext(context.Background(), `SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'customers'`).Scan(&name)
}
