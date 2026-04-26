package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

const baselineMigrationName = "0001_baseline"

var baselineTableNames = []string{
	"legal_entities",
	"issuer_profiles",
	"customer_profiles",
	"service_agreements",
	"time_entries",
	"invoice_sequences",
	"invoices",
	"invoice_lines",
}

type migration struct {
	version int
	name    string
	sql     string
}

func applyMigrations(db *sql.DB, source fs.FS) error {
	migrations, err := loadMigrations(source)
	if err != nil {
		return fmt.Errorf("load migrations: %w", err)
	}
	if err := ensureMigrationsTable(db); err != nil {
		return fmt.Errorf("metadata bootstrap: %w", err)
	}

	version, err := currentVersion(db)
	if err != nil {
		return fmt.Errorf("metadata current version: %w", err)
	}
	if version == 0 {
		baselined, err := baselineExisting(db)
		if err != nil {
			return fmt.Errorf("baseline detection: %w", err)
		}
		if baselined {
			if err := recordBaseline(db); err != nil {
				return fmt.Errorf("baseline detection record version 1: %w", err)
			}
			version = 1
		}
	}

	for _, migration := range migrations {
		if migration.version <= version {
			continue
		}
		if err := applyMigration(db, migration); err != nil {
			return err
		}
	}

	return nil
}

func loadMigrations(source fs.FS) ([]migration, error) {
	var migrations []migration
	seen := make(map[int]string)
	err := fs.WalkDir(source, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			return nil
		}

		migration, err := parseMigration(source, path)
		if err != nil {
			return err
		}
		if previous, ok := seen[migration.version]; ok {
			return fmt.Errorf("duplicate migration version %d: %s and %s", migration.version, previous, path)
		}
		seen[migration.version] = path
		migrations = append(migrations, migration)
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].version < migrations[j].version
	})
	for i, migration := range migrations {
		expected := i + 1
		if migration.version != expected {
			if i == 0 {
				return nil, fmt.Errorf("migration version gap: got %d before 1", migration.version)
			}
			return nil, fmt.Errorf("migration version gap: got %d after %d", migration.version, migrations[i-1].version)
		}
	}

	return migrations, nil
}

func parseMigration(source fs.FS, path string) (migration, error) {
	base := filepath.Base(path)
	name := strings.TrimSuffix(base, ".sql")
	parts := strings.SplitN(name, "_", 2)
	if len(parts) != 2 || len(parts[0]) != 4 {
		return migration{}, fmt.Errorf("invalid migration filename %q", path)
	}
	version, err := strconv.Atoi(parts[0])
	if err != nil {
		return migration{}, fmt.Errorf("parse migration filename %q: %w", path, err)
	}
	if version <= 0 {
		return migration{}, fmt.Errorf("invalid migration version %d in %q", version, path)
	}
	contents, err := fs.ReadFile(source, path)
	if err != nil {
		return migration{}, fmt.Errorf("read migration %q: %w", path, err)
	}
	return migration{version: version, name: name, sql: string(contents)}, nil
}

func ensureMigrationsTable(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
    version    INTEGER PRIMARY KEY,
    name       TEXT NOT NULL,
    applied_at INTEGER NOT NULL
)`)
	if err != nil {
		return fmt.Errorf("create schema_migrations table: %w", err)
	}
	return nil
}

func currentVersion(db *sql.DB) (int, error) {
	var version sql.NullInt64
	err := db.QueryRow(`SELECT MAX(version) FROM schema_migrations`).Scan(&version)
	if err != nil {
		return 0, fmt.Errorf("query schema_migrations max version: %w", err)
	}
	if !version.Valid {
		return 0, nil
	}
	return int(version.Int64), nil
}

func baselineExisting(db *sql.DB) (bool, error) {
	tables, err := userTables(db)
	if err != nil {
		return false, err
	}

	delete(tables, "schema_migrations")
	if len(tables) == 0 {
		return false, nil
	}
	missing, extra := baselineTableDiff(tables)
	if len(missing) == 0 && len(extra) == 0 {
		return true, nil
	}

	parts := make([]string, 0, 2)
	if len(missing) > 0 {
		parts = append(parts, "missing tables "+strings.Join(missing, ", "))
	}
	if len(extra) > 0 {
		parts = append(parts, "unexpected tables "+strings.Join(extra, ", "))
	}
	return false, fmt.Errorf("baseline mismatch: %s", strings.Join(parts, "; "))
}

func baselineTableDiff(tables map[string]bool) ([]string, []string) {
	expected := make(map[string]bool, len(baselineTableNames))
	for _, name := range baselineTableNames {
		expected[name] = true
	}

	var missing []string
	for _, name := range baselineTableNames {
		if !tables[name] {
			missing = append(missing, name)
		}
	}

	var extra []string
	for name := range tables {
		if !expected[name] {
			extra = append(extra, name)
		}
	}
	sort.Strings(extra)
	return missing, extra
}

func userTables(db *sql.DB) (map[string]bool, error) {
	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'`)
	if err != nil {
		return nil, fmt.Errorf("query sqlite_master tables: %w", err)
	}
	defer rows.Close()

	tables := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan sqlite_master table: %w", err)
		}
		tables[name] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sqlite_master tables: %w", err)
	}
	return tables, nil
}

func recordBaseline(db *sql.DB) error {
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("begin baseline transaction: %w", err)
	}
	if err := insertMigrationVersion(tx, 1, baselineMigrationName); err != nil {
		return rollbackWithCause(tx, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit baseline transaction: %w", err)
	}
	return nil
}

func applyMigration(db *sql.DB, migration migration) error {
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("begin migration %d: %w", migration.version, err)
	}
	if _, err := tx.Exec(migration.sql); err != nil {
		return fmt.Errorf("apply migration %d: %w", migration.version, rollbackWithCause(tx, err))
	}
	if err := insertMigrationVersion(tx, migration.version, migration.name); err != nil {
		return fmt.Errorf("record migration %d: %w", migration.version, rollbackWithCause(tx, err))
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration %d: %w", migration.version, err)
	}
	return nil
}

func insertMigrationVersion(tx *sql.Tx, version int, name string) error {
	_, err := tx.Exec(`INSERT INTO schema_migrations (version, name, applied_at) VALUES (?, ?, ?)`, version, name, time.Now().UTC().UnixNano())
	if err != nil {
		return fmt.Errorf("insert schema_migrations version %d: %w", version, err)
	}
	return nil
}

func rollbackWithCause(tx *sql.Tx, cause error) error {
	if err := tx.Rollback(); err != nil {
		return errors.Join(cause, fmt.Errorf("rollback migration transaction: %w", err))
	}
	return cause
}
