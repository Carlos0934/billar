package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

type Store struct {
	db      *sql.DB
	cleanup func()
}

func Open(path string) (*Store, error) {
	cleanup := func() {}
	path = strings.TrimSpace(path)
	if path == "" {
		tempFile, err := os.CreateTemp("", "billar-*.db")
		if err != nil {
			return nil, fmt.Errorf("create sqlite temp file: %w", err)
		}
		if err := tempFile.Close(); err != nil {
			_ = os.Remove(tempFile.Name())
			return nil, fmt.Errorf("close sqlite temp file: %w", err)
		}
		path = tempFile.Name()
		cleanup = func() { _ = os.Remove(path) }
	}

	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return nil, fmt.Errorf("create sqlite directory: %w", err)
		}
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	// Enable foreign key constraints (SQLite disabled by default)
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		cleanup()
		_ = db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	if err := applyMigrations(db, migrationsFS); err != nil {
		_ = db.Close()
		cleanup()
		return nil, fmt.Errorf("apply sqlite migrations: %w", err)
	}

	store := &Store{db: db, cleanup: cleanup}
	return store, nil
}

func (s *Store) DB() *sql.DB {
	if s == nil {
		return nil
	}
	return s.db
}

func (s *Store) Close() error {
	if s == nil {
		return nil
	}

	var errs []error
	if s.db != nil {
		if err := s.db.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if s.cleanup != nil {
		s.cleanup()
		s.cleanup = nil
	}

	return errors.Join(errs...)
}
