package backup

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	infrasqlite "github.com/Carlos0934/billar/internal/infra/sqlite"

	_ "modernc.org/sqlite"
)

type RestoreRequest struct {
	BackupID string
	File     string
}

type Validation struct {
	OK           bool
	SidecarOK    bool
	HashOK       bool
	SizeOK       bool
	IntegrityOK  bool
	TablesOK     bool
	SchemaOK     bool
	BackupSchema int
	BinarySchema int
}

type Restorer struct {
	AfterCopy func(tempPath string) error
}

func (r Restorer) Resolve(ctx context.Context, req RestoreRequest, backupDir string) (Record, error) {
	select {
	case <-ctx.Done():
		return Record{}, ctx.Err()
	default:
	}
	if strings.TrimSpace(req.BackupID) != "" {
		return LookupByID(backupDir, strings.TrimSpace(req.BackupID))
	}
	path := strings.TrimSpace(req.File)
	if path == "" {
		return Record{}, fmt.Errorf("exactly_one_selector: provide --id or --file")
	}
	record, err := readRecord(path)
	if err != nil {
		if strings.Contains(err.Error(), "parse backup metadata") {
			return Record{}, fmt.Errorf("sidecar_parse_error: %w", err)
		}
		return Record{}, err
	}
	if !record.Metadata {
		return Record{}, fmt.Errorf("sidecar_missing: backup %q is missing metadata sidecar %q", path, path+".json")
	}
	if wantID := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)); record.ID != wantID {
		return Record{}, fmt.Errorf("id_basename_mismatch: sidecar id %q does not match file basename %q", record.ID, wantID)
	}
	return record, nil
}

func (r Restorer) Validate(ctx context.Context, record Record, targetDBPath string, binarySchema int) (Validation, error) {
	validation := Validation{SidecarOK: record.Metadata, BinarySchema: binarySchema}
	if !record.Metadata {
		return validation, fmt.Errorf("sidecar_missing: backup %q is missing metadata sidecar", record.File)
	}
	if filepath.Clean(record.File) == filepath.Clean(targetDBPath) {
		return validation, fmt.Errorf("source_is_target: backup source and target database are the same path")
	}

	size, sha, err := fileStatsAndHash(record.File)
	if err != nil {
		return validation, err
	}
	validation.SizeOK = size == record.SizeBytes
	if !validation.SizeOK {
		return validation, fmt.Errorf("size_mismatch: sidecar size %d does not match computed size %d", record.SizeBytes, size)
	}
	validation.HashOK = strings.EqualFold(sha, record.SHA256)
	if !validation.HashOK {
		return validation, fmt.Errorf("hash_mismatch: sidecar sha256 %q does not match computed %q", record.SHA256, sha)
	}

	if err := sqliteIntegrity(ctx, record.File); err != nil {
		return validation, err
	}
	validation.IntegrityOK = true

	backupSchema, err := sqliteSchemaVersion(ctx, record.File)
	if err != nil {
		return validation, err
	}
	validation.BackupSchema = backupSchema
	if backupSchema != record.SchemaVersion {
		return validation, fmt.Errorf("schema_sidecar_mismatch: sidecar schema %d does not match database schema %d", record.SchemaVersion, backupSchema)
	}
	if backupSchema > binarySchema {
		return validation, fmt.Errorf("schema_newer_than_binary: backup schema %d exceeds binary schema %d", backupSchema, binarySchema)
	}
	validation.SchemaOK = true

	missing, err := missingRequiredTables(ctx, record.File)
	if err != nil {
		return validation, err
	}
	if len(missing) > 0 {
		return validation, fmt.Errorf("missing_required_tables: %s", strings.Join(missing, ", "))
	}
	validation.TablesOK = true
	validation.OK = validation.SidecarOK && validation.SizeOK && validation.HashOK && validation.IntegrityOK && validation.SchemaOK && validation.TablesOK
	return validation, nil
}

func (r Restorer) Replace(ctx context.Context, record Record, targetDBPath string) error {
	parent := filepath.Dir(targetDBPath)
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return fmt.Errorf("create target database directory: %w", err)
	}
	tmp, err := os.CreateTemp(parent, "."+filepath.Base(targetDBPath)+".restore-*")
	if err != nil {
		return fmt.Errorf("create restore temp file: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()

	src, err := os.Open(record.File)
	if err != nil {
		_ = tmp.Close()
		return fmt.Errorf("open backup source: %w", err)
	}
	if _, err := io.Copy(tmp, src); err != nil {
		_ = src.Close()
		_ = tmp.Close()
		return fmt.Errorf("copy backup to restore temp file: %w", err)
	}
	if err := src.Close(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("close backup source: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync restore temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close restore temp file: %w", err)
	}
	if err := os.Chmod(tmpPath, 0o600); err != nil {
		return fmt.Errorf("secure restore temp file: %w", err)
	}
	if r.AfterCopy != nil {
		if err := r.AfterCopy(tmpPath); err != nil {
			return fmt.Errorf("after copy restore hook: %w", err)
		}
	}
	size, sha, err := fileStatsAndHash(tmpPath)
	if err != nil {
		return fmt.Errorf("rehash restore temp file: %w", err)
	}
	if size != record.SizeBytes || !strings.EqualFold(sha, record.SHA256) {
		return fmt.Errorf("temp_hash_mismatch: restore temp file does not match sidecar metadata")
	}
	if err := sqliteIntegrity(ctx, tmpPath); err != nil {
		return fmt.Errorf("temp_integrity_failed: %w", err)
	}
	if err := os.Rename(tmpPath, targetDBPath); err != nil {
		return fmt.Errorf("publish restored database atomically: %w", err)
	}
	cleanup = false
	fSyncDir(parent)
	return nil
}

func sqliteIntegrity(ctx context.Context, path string) error {
	db, err := sql.Open("sqlite", "file:"+path+"?mode=ro")
	if err != nil {
		return fmt.Errorf("open backup read-only: %w", err)
	}
	defer db.Close()
	var got string
	if err := db.QueryRowContext(ctx, "PRAGMA integrity_check").Scan(&got); err != nil {
		return fmt.Errorf("integrity_check_failed: %w", err)
	}
	if got != "ok" {
		return fmt.Errorf("integrity_check_failed: %s", got)
	}
	return nil
}

func sqliteSchemaVersion(ctx context.Context, path string) (int, error) {
	db, err := sql.Open("sqlite", "file:"+path+"?mode=ro")
	if err != nil {
		return 0, fmt.Errorf("open backup read-only: %w", err)
	}
	defer db.Close()
	var version int
	if err := db.QueryRowContext(ctx, "SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&version); err != nil {
		return 0, fmt.Errorf("schema_migrations_missing: %w", err)
	}
	return version, nil
}

func missingRequiredTables(ctx context.Context, path string) ([]string, error) {
	db, err := sql.Open("sqlite", "file:"+path+"?mode=ro")
	if err != nil {
		return nil, fmt.Errorf("open backup read-only: %w", err)
	}
	defer db.Close()
	missing := make([]string, 0)
	for _, name := range infrasqlite.RequiredBillarTables {
		var count int
		if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", name).Scan(&count); err != nil {
			return nil, fmt.Errorf("check required table %q: %w", name, err)
		}
		if count == 0 {
			missing = append(missing, name)
		}
	}
	return missing, nil
}

func fSyncDir(path string) {
	dir, err := os.Open(path)
	if err != nil {
		return
	}
	defer dir.Close()
	_ = dir.Sync()
}
