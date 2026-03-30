package unlocksession_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Carlos0934/billar/internal/application/access/ports"
	"github.com/Carlos0934/billar/internal/application/access/unlock_session"
	"github.com/Carlos0934/billar/internal/domain/access/session"
)

func TestUnlockSessionUnlocksLockedAccessWithValidSecret(t *testing.T) {
	sessionID := mustSessionID(t, "session-123")
	now := time.Date(2026, time.March, 30, 13, 0, 0, 0, time.UTC)
	store := &fakeCurrentUnlockedSessionStore{}
	verifier := &fakeUnlockSecretVerifier{}
	clock := &fakeClock{now: now}
	idGenerator := &fakeSessionIDGenerator{id: sessionID}

	useCase := unlocksession.NewUseCase(store, verifier, clock, idGenerator)
	result, err := useCase.Execute(context.Background(), unlocksession.Command{Secret: "top-secret"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if got := verifier.secrets; len(got) != 1 || got[0] != "top-secret" {
		t.Fatalf("verified secrets = %v, want [top-secret]", got)
	}
	if got := store.getCalls; got != 1 {
		t.Fatalf("GetCurrent() calls = %d, want %d", got, 1)
	}
	if got := store.saveCalls; got != 1 {
		t.Fatalf("SaveCurrent() calls = %d, want %d", got, 1)
	}
	if got := idGenerator.calls; got != 1 {
		t.Fatalf("SessionIDGenerator calls = %d, want %d", got, 1)
	}
	if got := clock.calls; got != 1 {
		t.Fatalf("Clock calls = %d, want %d", got, 1)
	}

	assertUnlockedStatus(t, result, sessionID.String(), now)

	if store.current == nil {
		t.Fatal("expected store to hold current session after unlock")
	}
	if got := store.current.Status(); got != session.StatusUnlocked {
		t.Fatalf("stored status = %q, want %q", got, session.StatusUnlocked)
	}
}

func TestUnlockSessionFailsOnInvalidSecretAndKeepsLocked(t *testing.T) {
	verificationErr := errors.New("verification failed")
	store := &fakeCurrentUnlockedSessionStore{}
	verifier := &fakeUnlockSecretVerifier{err: verificationErr}
	clock := &fakeClock{now: time.Date(2026, time.March, 30, 13, 0, 0, 0, time.UTC)}
	idGenerator := &fakeSessionIDGenerator{id: mustSessionID(t, "session-123")}

	useCase := unlocksession.NewUseCase(store, verifier, clock, idGenerator)
	result, err := useCase.Execute(context.Background(), unlocksession.Command{Secret: "bad-secret"})
	if !errors.Is(err, verificationErr) {
		t.Fatalf("Execute() error = %v, want %v", err, verificationErr)
	}

	if got := result.Status; got != "locked" {
		t.Fatalf("status = %q, want %q", got, "locked")
	}
	if result.SessionID != nil {
		t.Fatal("expected locked result to omit session id")
	}
	if got := store.saveCalls; got != 0 {
		t.Fatalf("SaveCurrent() calls = %d, want %d", got, 0)
	}
	if got := idGenerator.calls; got != 0 {
		t.Fatalf("SessionIDGenerator calls = %d, want %d", got, 0)
	}
	if got := clock.calls; got != 0 {
		t.Fatalf("Clock calls = %d, want %d", got, 0)
	}
}

func TestUnlockSessionReturnsExistingUnlockedSessionIdempotently(t *testing.T) {
	now := time.Date(2026, time.March, 30, 13, 0, 0, 0, time.UTC)
	existing := mustSession(t, "session-123", now)
	store := &fakeCurrentUnlockedSessionStore{current: existing}
	verifier := &fakeUnlockSecretVerifier{}
	clock := &fakeClock{now: now.Add(time.Hour)}
	idGenerator := &fakeSessionIDGenerator{id: mustSessionID(t, "session-456")}

	useCase := unlocksession.NewUseCase(store, verifier, clock, idGenerator)
	result, err := useCase.Execute(context.Background(), unlocksession.Command{Secret: "top-secret"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	assertUnlockedStatus(t, result, "session-123", now)

	if got := verifier.secrets; len(got) != 0 {
		t.Fatalf("verified secrets = %v, want none", got)
	}
	if got := store.saveCalls; got != 0 {
		t.Fatalf("SaveCurrent() calls = %d, want %d", got, 0)
	}
	if got := idGenerator.calls; got != 0 {
		t.Fatalf("SessionIDGenerator calls = %d, want %d", got, 0)
	}
	if got := clock.calls; got != 0 {
		t.Fatalf("Clock calls = %d, want %d", got, 0)
	}
}

func assertUnlockedStatus(t *testing.T, result unlocksession.Result, wantSessionID string, wantTime time.Time) {
	t.Helper()

	if got := result.Status; got != "unlocked" {
		t.Fatalf("status = %q, want %q", got, "unlocked")
	}
	if result.SessionID == nil || *result.SessionID != wantSessionID {
		t.Fatalf("session id = %v, want %q", result.SessionID, wantSessionID)
	}
	if result.UnlockedAt == nil || !result.UnlockedAt.Equal(wantTime) {
		t.Fatalf("unlocked at = %v, want %v", result.UnlockedAt, wantTime)
	}
	if result.LastActivityAt == nil || !result.LastActivityAt.Equal(wantTime) {
		t.Fatalf("last activity at = %v, want %v", result.LastActivityAt, wantTime)
	}
}

type fakeCurrentUnlockedSessionStore struct {
	current     *session.Session
	err         error
	getCalls    int
	saveCalls   int
	deleteCalls int
}

func (store *fakeCurrentUnlockedSessionStore) GetCurrent(context.Context) (*session.Session, error) {
	store.getCalls++
	if store.err != nil {
		return nil, store.err
	}

	return store.current, nil
}

func (store *fakeCurrentUnlockedSessionStore) SaveCurrent(_ context.Context, current *session.Session) error {
	store.saveCalls++
	store.current = current
	return nil
}

func (store *fakeCurrentUnlockedSessionStore) DeleteCurrent(context.Context) error {
	store.deleteCalls++
	store.current = nil
	return nil
}

type fakeUnlockSecretVerifier struct {
	secrets []string
	err     error
}

func (verifier *fakeUnlockSecretVerifier) Verify(_ context.Context, secret string) error {
	verifier.secrets = append(verifier.secrets, secret)
	return verifier.err
}

type fakeClock struct {
	now   time.Time
	calls int
}

func (clock *fakeClock) Now() time.Time {
	clock.calls++
	return clock.now
}

type fakeSessionIDGenerator struct {
	id    session.SessionID
	calls int
}

func (generator *fakeSessionIDGenerator) New() session.SessionID {
	generator.calls++
	return generator.id
}

var _ ports.CurrentUnlockedSessionStore = (*fakeCurrentUnlockedSessionStore)(nil)
var _ ports.UnlockSecretVerifier = (*fakeUnlockSecretVerifier)(nil)
var _ ports.Clock = (*fakeClock)(nil)
var _ ports.SessionIDGenerator = (*fakeSessionIDGenerator)(nil)

func mustSessionID(t *testing.T, value string) session.SessionID {
	t.Helper()

	id, err := session.NewSessionID(value)
	if err != nil {
		t.Fatalf("NewSessionID() error = %v", err)
	}

	return id
}

func mustSession(t *testing.T, value string, unlockedAt time.Time) *session.Session {
	t.Helper()

	current, err := session.New(session.CreateParams{
		ID:         mustSessionID(t, value),
		UnlockedAt: unlockedAt,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	return current
}
