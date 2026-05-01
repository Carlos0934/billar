package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
)

type DoctorReadOnlyProbe struct{}

func (DoctorReadOnlyProbe) Probe(ctx context.Context, dbPath string) (int, error) {
	dbPath = strings.TrimSpace(dbPath)
	if dbPath == "" {
		return 0, fmt.Errorf("read-only sqlite probe: database path is required")
	}
	db, err := sql.Open("sqlite", "file:"+dbPath+"?mode=ro")
	if err != nil {
		return 0, fmt.Errorf("open sqlite read-only: %w", err)
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		return 0, fmt.Errorf("ping sqlite read-only: %w", err)
	}
	var version int
	if err := db.QueryRowContext(ctx, `SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&version); err != nil {
		return 0, fmt.Errorf("read sqlite schema version read-only: %w", err)
	}
	return version, nil
}
