package backup

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type Record struct {
	ID            string    `json:"id" toon:"id"`
	File          string    `json:"file" toon:"file"`
	SidecarFile   string    `json:"sidecar_file" toon:"sidecar_file"`
	CreatedAt     time.Time `json:"created_at" toon:"created_at"`
	SchemaVersion int       `json:"schema_version" toon:"schema_version"`
	SizeBytes     int64     `json:"size_bytes" toon:"size_bytes"`
	SHA256        string    `json:"sha256" toon:"sha256"`
	SourceDBPath  string    `json:"source_db_path" toon:"source_db_path"`
	Metadata      bool      `json:"metadata" toon:"metadata"`
}

type vacuumDB interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	Close() error
}

type Snapshotter struct {
	Now  func() time.Time
	Open func(dbPath string) (vacuumDB, error)
}

func (s Snapshotter) Create(ctx context.Context, dbPath, destDir string) (Record, error) {
	if err := os.MkdirAll(destDir, 0o700); err != nil {
		return Record{}, fmt.Errorf("create backup directory %q: %w", destDir, err)
	}
	if err := os.Chmod(destDir, 0o700); err != nil {
		return Record{}, fmt.Errorf("secure backup directory %q: %w", destDir, err)
	}

	createdAt := time.Now().UTC()
	if s.Now != nil {
		createdAt = s.Now().UTC()
	}
	schemaVersion, err := readSchemaVersion(dbPath)
	if err != nil {
		return Record{}, fmt.Errorf("read schema version: %w", err)
	}

	id := fmt.Sprintf("billar-%s-schema%d", createdAt.Format("20060102T150405Z"), schemaVersion)
	finalPath := filepath.Join(destDir, id+".db")
	sidecarPath := finalPath + ".json"
	tmpPath := finalPath + ".tmp"
	if _, err := os.Stat(finalPath); err == nil {
		return Record{}, fmt.Errorf("create backup: %s already exists", finalPath)
	} else if !os.IsNotExist(err) {
		return Record{}, fmt.Errorf("check backup target %q: %w", finalPath, err)
	}

	db, err := s.open(dbPath)
	if err != nil {
		return Record{}, fmt.Errorf("open source database read-only: %w", err)
	}
	defer db.Close()

	cleanup := func() {
		_ = os.Remove(tmpPath)
		_ = os.Remove(finalPath)
		_ = os.Remove(sidecarPath)
	}
	if _, err := db.ExecContext(ctx, "VACUUM INTO ?", tmpPath); err != nil {
		cleanup()
		return Record{}, fmt.Errorf("vacuum source database into backup: %w", err)
	}

	if err := os.Chmod(tmpPath, 0o600); err != nil {
		cleanup()
		return Record{}, fmt.Errorf("secure backup temp file: %w", err)
	}
	if err := os.Rename(tmpPath, finalPath); err != nil {
		cleanup()
		return Record{}, fmt.Errorf("publish backup atomically: %w", err)
	}

	size, sha, err := fileStatsAndHash(finalPath)
	if err != nil {
		cleanup()
		return Record{}, err
	}

	record := Record{
		ID:            id,
		File:          finalPath,
		SidecarFile:   sidecarPath,
		CreatedAt:     createdAt,
		SchemaVersion: schemaVersion,
		SizeBytes:     size,
		SHA256:        sha,
		SourceDBPath:  dbPath,
		Metadata:      true,
	}
	if err := writeSidecar(sidecarPath, record); err != nil {
		cleanup()
		return Record{}, err
	}

	return record, nil
}

func (s Snapshotter) open(dbPath string) (vacuumDB, error) {
	if s.Open != nil {
		return s.Open(dbPath)
	}
	return sql.Open("sqlite", "file:"+dbPath+"?mode=ro")
}

func readSchemaVersion(dbPath string) (int, error) {
	db, err := sql.Open("sqlite", "file:"+dbPath+"?mode=ro")
	if err != nil {
		return 0, err
	}
	defer db.Close()

	var version int
	if err := db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&version); err != nil {
		return 0, err
	}
	return version, nil
}

func fileStatsAndHash(path string) (int64, string, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, "", fmt.Errorf("open backup for hashing: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	size, err := io.Copy(h, f)
	if err != nil {
		return 0, "", fmt.Errorf("hash backup: %w", err)
	}
	return size, hex.EncodeToString(h.Sum(nil)), nil
}

func writeSidecar(path string, record Record) error {
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal backup metadata: %w", err)
	}

	tmp, err := os.OpenFile(path+".tmp", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("create backup metadata temp file: %w", err)
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("write backup metadata: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("close backup metadata: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("publish backup metadata: %w", err)
	}
	return nil
}
