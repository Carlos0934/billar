package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Carlos0934/billar/internal/app"
	"github.com/Carlos0934/billar/internal/core"
)

func TestMemoryStateStoreGenerateAndValidate(t *testing.T) {
	t.Parallel()

	store := NewMemoryStateStore(time.Minute)
	state, err := store.Generate(context.Background())
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if state == "" {
		t.Fatal("Generate() state = empty, want generated state")
	}
	if err := store.Validate(context.Background(), state); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if err := store.Validate(context.Background(), state); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("Validate() second call error = %v, want %v", err, ErrInvalidState)
	}
}

func TestMemoryStateStoreRejectsExpiredState(t *testing.T) {
	t.Parallel()

	store := NewMemoryStateStore(time.Minute)
	now := time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }
	state, err := store.Generate(context.Background())
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	store.now = func() time.Time { return now.Add(2 * time.Minute) }
	if err := store.Validate(context.Background(), state); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("Validate() error = %v, want %v", err, ErrInvalidState)
	}
}

func TestMemorySessionStoreSaveAndGetCurrent(t *testing.T) {
	t.Parallel()

	store := NewMemorySessionStore()
	if err := store.Save(context.Background(), &core.Session{Status: core.SessionStatusActive, ID: "session-123"}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	got, err := store.GetCurrent(context.Background())
	if err != nil {
		t.Fatalf("GetCurrent() error = %v", err)
	}
	if got == nil || got.ID != "session-123" {
		t.Fatalf("GetCurrent() = %+v, want session-123", got)
	}
}

func TestEmailAccessPolicy(t *testing.T) {
	t.Parallel()

	policy := app.IdentityPolicy{
		AllowedEmails:  []string{"admin@example.com"},
		AllowedDomains: []string{"company.com"},
	}
	if !policy.IsAllowed("admin@example.com") {
		t.Fatal("IsAllowed(admin@example.com) = false, want true")
	}
	if !policy.IsAllowed("user@company.com") {
		t.Fatal("IsAllowed(user@company.com) = false, want true")
	}
	if policy.IsAllowed("user@elsewhere.com") {
		t.Fatal("IsAllowed(user@elsewhere.com) = true, want false")
	}
}
