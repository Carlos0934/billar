package getsessionstatus_test

import (
	"context"
	"testing"
	"time"

	"github.com/Carlos0934/billar/internal/application/access/get_session_status"
	"github.com/Carlos0934/billar/internal/application/access/ports"
	"github.com/Carlos0934/billar/internal/domain/access/session"
)

func TestGetSessionStatusReturnsLockedWhenSessionIsMissing(t *testing.T) {
	store := &fakeCurrentUnlockedSessionStore{}

	useCase := getsessionstatus.NewUseCase(store)
	result, err := useCase.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if got := result.Status; got != "locked" {
		t.Fatalf("status = %q, want %q", got, "locked")
	}
	if result.SessionID != nil {
		t.Fatal("expected locked result to omit session id")
	}
	if got := store.getCalls; got != 1 {
		t.Fatalf("GetCurrent() calls = %d, want %d", got, 1)
	}
}

func TestGetSessionStatusReturnsExistingUnlockedSession(t *testing.T) {
	unlockedAt := time.Date(2026, time.March, 30, 13, 0, 0, 0, time.UTC)
	store := &fakeCurrentUnlockedSessionStore{current: mustSession(t, "session-123", unlockedAt)}

	useCase := getsessionstatus.NewUseCase(store)
	result, err := useCase.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if got := result.Status; got != "unlocked" {
		t.Fatalf("status = %q, want %q", got, "unlocked")
	}
	if result.SessionID == nil || *result.SessionID != "session-123" {
		t.Fatalf("session id = %v, want %q", result.SessionID, "session-123")
	}
	if result.UnlockedAt == nil || !result.UnlockedAt.Equal(unlockedAt) {
		t.Fatalf("unlocked at = %v, want %v", result.UnlockedAt, unlockedAt)
	}
	if result.LastActivityAt == nil || !result.LastActivityAt.Equal(unlockedAt) {
		t.Fatalf("last activity at = %v, want %v", result.LastActivityAt, unlockedAt)
	}
}

type fakeCurrentUnlockedSessionStore struct {
	current  *session.Session
	getCalls int
}

func (store *fakeCurrentUnlockedSessionStore) GetCurrent(context.Context) (*session.Session, error) {
	store.getCalls++
	return store.current, nil
}

func (store *fakeCurrentUnlockedSessionStore) SaveCurrent(context.Context, *session.Session) error {
	return nil
}

func (store *fakeCurrentUnlockedSessionStore) DeleteCurrent(context.Context) error {
	return nil
}

var _ ports.CurrentUnlockedSessionStore = (*fakeCurrentUnlockedSessionStore)(nil)

func mustSession(t *testing.T, value string, unlockedAt time.Time) *session.Session {
	t.Helper()

	id, err := session.NewSessionID(value)
	if err != nil {
		t.Fatalf("NewSessionID() error = %v", err)
	}

	current, err := session.New(session.CreateParams{ID: id, UnlockedAt: unlockedAt})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	return current
}
