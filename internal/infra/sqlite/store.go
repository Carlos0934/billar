package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "embed"
	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

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
		if err := os.MkdirAll(dir, 0o755); err != nil {
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

	store := &Store{db: db, cleanup: cleanup}
	if err := store.bootstrap(); err != nil {
		_ = store.Close()
		return nil, err
	}

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

func (s *Store) bootstrap() error {
	if s == nil || s.db == nil {
		return errors.New("sqlite store is not open")
	}

	if strings.TrimSpace(schemaSQL) == "" {
		return errors.New("sqlite schema is empty")
	}

	if _, err := s.db.Exec(schemaSQL); err != nil {
		return fmt.Errorf("bootstrap sqlite schema: %w", err)
	}

	return nil
}
