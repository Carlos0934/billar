package locksession_test

import (
	"context"
	"testing"
	"time"

	"github.com/Carlos0934/billar/internal/application/access/lock_session"
	"github.com/Carlos0934/billar/internal/application/access/ports"
	"github.com/Carlos0934/billar/internal/domain/access/session"
)

func TestLockSessionLocksActiveSession(t *testing.T) {
	unlockedAt := time.Date(2026, time.March, 30, 13, 0, 0, 0, time.UTC)
	lockedAt := unlockedAt.Add(90 * time.Minute)
	store := &fakeCurrentUnlockedSessionStore{current: mustSession(t, "session-123", unlockedAt)}
	clock := &fakeClock{now: lockedAt}

	useCase := locksession.NewUseCase(store, clock)
	result, err := useCase.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if got := result.Status; got != "locked" {
		t.Fatalf("status = %q, want %q", got, "locked")
	}
	if got := store.getCalls; got != 1 {
		t.Fatalf("GetCurrent() calls = %d, want %d", got, 1)
	}
	if got := store.deleteCalls; got != 1 {
		t.Fatalf("DeleteCurrent() calls = %d, want %d", got, 1)
	}
	if got := clock.calls; got != 1 {
		t.Fatalf("Clock calls = %d, want %d", got, 1)
	}
	if got := store.deleted.Status(); got != session.StatusLocked {
		t.Fatalf("deleted session status = %q, want %q", got, session.StatusLocked)
	}
	if got := store.deleted.LockedAt(); got == nil || !got.Equal(lockedAt) {
		t.Fatalf("deleted session locked at = %v, want %v", got, lockedAt)
	}
}

func TestLockSessionIsIdempotentWhenAlreadyLocked(t *testing.T) {
	store := &fakeCurrentUnlockedSessionStore{}
	clock := &fakeClock{now: time.Date(2026, time.March, 30, 14, 0, 0, 0, time.UTC)}

	useCase := locksession.NewUseCase(store, clock)
	result, err := useCase.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if got := result.Status; got != "locked" {
		t.Fatalf("status = %q, want %q", got, "locked")
	}
	if got := store.deleteCalls; got != 0 {
		t.Fatalf("DeleteCurrent() calls = %d, want %d", got, 0)
	}
	if got := clock.calls; got != 0 {
		t.Fatalf("Clock calls = %d, want %d", got, 0)
	}
}

type fakeCurrentUnlockedSessionStore struct {
	current     *session.Session
	deleted     *session.Session
	getCalls    int
	deleteCalls int
}

func (store *fakeCurrentUnlockedSessionStore) GetCurrent(context.Context) (*session.Session, error) {
	store.getCalls++
	return store.current, nil
}

func (store *fakeCurrentUnlockedSessionStore) SaveCurrent(context.Context, *session.Session) error {
	return nil
}

func (store *fakeCurrentUnlockedSessionStore) DeleteCurrent(context.Context) error {
	store.deleteCalls++
	store.deleted = store.current
	store.current = nil
	return nil
}

type fakeClock struct {
	now   time.Time
	calls int
}

func (clock *fakeClock) Now() time.Time {
	clock.calls++
	return clock.now
}

var _ ports.CurrentUnlockedSessionStore = (*fakeCurrentUnlockedSessionStore)(nil)
var _ ports.Clock = (*fakeClock)(nil)

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
