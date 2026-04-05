package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Carlos0934/billar/internal/app"
	"github.com/Carlos0934/billar/internal/core"
)

type IssuerProfileStore struct {
	db *sql.DB
}

func NewIssuerProfileStore(store *Store) *IssuerProfileStore {
	if store == nil {
		return nil
	}
	return &IssuerProfileStore{db: store.DB()}
}

func (s *IssuerProfileStore) Save(ctx context.Context, profile *core.IssuerProfile) error {
	if s == nil || s.db == nil {
		return errors.New("issuer profile sqlite store is required")
	}
	if profile == nil {
		return errors.New("issuer profile is required")
	}

	// Check if profile already exists to avoid INSERT OR REPLACE which bypasses UNIQUE constraint
	var exists bool
	err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM issuer_profiles WHERE id = ?)", profile.ID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check existence: %w", err)
	}

	if exists {
		// Use UPDATE for existing profiles - preserves legal_entity_id and created_at
		_, err = s.db.ExecContext(ctx, `
UPDATE issuer_profiles SET
	default_currency = ?, default_notes = ?, updated_at = ?
WHERE id = ?`,
			profile.DefaultCurrency,
			profile.DefaultNotes,
			profile.UpdatedAt.UTC().UnixNano(),
			profile.ID,
		)
		if err != nil {
			return fmt.Errorf("update issuer profile: %w", err)
		}
	} else {
		// Use INSERT for new profiles - will fail if legal_entity_id UNIQUE constraint is violated
		_, err = s.db.ExecContext(ctx, `
INSERT INTO issuer_profiles (
	id, legal_entity_id, default_currency, default_notes, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?)`,
			profile.ID,
			profile.LegalEntityID,
			profile.DefaultCurrency,
			profile.DefaultNotes,
			profile.CreatedAt.UTC().UnixNano(),
			profile.UpdatedAt.UTC().UnixNano(),
		)
		if err != nil {
			return fmt.Errorf("insert issuer profile: %w", err)
		}
	}

	return nil
}

func (s *IssuerProfileStore) GetByID(ctx context.Context, id string) (*core.IssuerProfile, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("issuer profile sqlite store is required")
	}

	query := `SELECT id, legal_entity_id, default_currency, default_notes, created_at, updated_at FROM issuer_profiles WHERE id = ?`
	row := s.db.QueryRowContext(ctx, query, id)

	var profile core.IssuerProfile
	var createdAt int64
	var updatedAt int64

	if err := row.Scan(
		&profile.ID,
		&profile.LegalEntityID,
		&profile.DefaultCurrency,
		&profile.DefaultNotes,
		&createdAt,
		&updatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, app.ErrIssuerProfileNotFound
		}
		return nil, fmt.Errorf("get issuer profile by id: %w", err)
	}

	profile.CreatedAt = time.Unix(0, createdAt).UTC()
	profile.UpdatedAt = time.Unix(0, updatedAt).UTC()

	return &profile, nil
}
