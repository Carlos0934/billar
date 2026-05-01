package backup

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestListerList(t *testing.T) {
	t.Run("empty dir returns empty records", func(t *testing.T) {
		got, err := Lister{}.List(context.Background(), t.TempDir())
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if got == nil {
			t.Fatal("List() = nil, want empty non-nil slice")
		}
		if len(got) != 0 {
			t.Fatalf("List() = %+v, want empty", got)
		}
	})

	t.Run("mixed metadata includes sidecar and orphan db sorted newest first", func(t *testing.T) {
		dir := t.TempDir()
		oldTime := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
		newTime := time.Date(2026, 5, 1, 11, 0, 0, 0, time.UTC)

		withMetaDB := filepath.Join(dir, "billar-20260501T100000Z-schema7.db")
		writeFile(t, withMetaDB, "sqlite")
		writeJSON(t, withMetaDB+".json", Record{
			ID:            "billar-20260501T100000Z-schema7",
			File:          withMetaDB,
			SidecarFile:   withMetaDB + ".json",
			CreatedAt:     oldTime,
			SchemaVersion: 7,
			SizeBytes:     6,
			SHA256:        "abc",
			SourceDBPath:  "/source.db",
			Metadata:      true,
		})

		orphanDB := filepath.Join(dir, "billar-20260501T110000Z-schema0.db")
		writeFile(t, orphanDB, "orphan")

		got, err := Lister{}.List(context.Background(), dir)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("List() length = %d, want 2: %+v", len(got), got)
		}
		if got[0].File != orphanDB || got[0].Metadata {
			t.Fatalf("first record = %+v, want newest orphan with metadata false", got[0])
		}
		if got[0].SchemaVersion != 0 || got[0].SHA256 != "" || !got[0].CreatedAt.Equal(newTime) {
			t.Fatalf("orphan metadata fields = %+v, want zero metadata with time from filename", got[0])
		}
		if got[1].File != withMetaDB || !got[1].Metadata || got[1].SchemaVersion != 7 || got[1].SHA256 != "abc" {
			t.Fatalf("second record = %+v, want parsed sidecar", got[1])
		}
	})
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func writeJSON(t *testing.T, path string, record Record) {
	t.Helper()
	data, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}
