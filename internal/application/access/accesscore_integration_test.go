package access_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Carlos0934/billar/internal/application/access/get_session_status"
	"github.com/Carlos0934/billar/internal/application/access/lock_session"
	"github.com/Carlos0934/billar/internal/application/access/ports"
	"github.com/Carlos0934/billar/internal/application/access/unlock_session"
	"github.com/Carlos0934/billar/internal/domain/access/session"
)

func TestAccessCoreLifecycleUnlockStatusLockStatus(t *testing.T) {
	store := &inMemoryCurrentUnlockedSessionStore{}
	verifier := &allowSecretVerifier{validSecret: "top-secret"}
	clock := &sequenceClock{times: []time.Time{
		time.Date(2026, time.March, 30, 15, 0, 0, 0, time.UTC),
		time.Date(2026, time.March, 30, 16, 0, 0, 0, time.UTC),
	}}
	idGenerator := &fixedSessionIDGenerator{id: mustSessionID(t, "session-123")}

	unlock := unlocksession.NewUseCase(store, verifier, clock, idGenerator)
	status := getsessionstatus.NewUseCase(store)
	lock := locksession.NewUseCase(store, clock)

	unlocked, err := unlock.Execute(context.Background(), unlocksession.Command{Secret: "top-secret"})
	if err != nil {
		t.Fatalf("unlock Execute() error = %v", err)
	}
	if got := unlocked.Status; got != "unlocked" {
		t.Fatalf("unlock status = %q, want %q", got, "unlocked")
	}

	currentStatus, err := status.Execute(context.Background())
	if err != nil {
		t.Fatalf("status Execute() error = %v", err)
	}
	if got := currentStatus.Status; got != "unlocked" {
		t.Fatalf("current status = %q, want %q", got, "unlocked")
	}

	locked, err := lock.Execute(context.Background())
	if err != nil {
		t.Fatalf("lock Execute() error = %v", err)
	}
	if got := locked.Status; got != "locked" {
		t.Fatalf("lock status = %q, want %q", got, "locked")
	}

	finalStatus, err := status.Execute(context.Background())
	if err != nil {
		t.Fatalf("final status Execute() error = %v", err)
	}
	if got := finalStatus.Status; got != "locked" {
		t.Fatalf("final status = %q, want %q", got, "locked")
	}

	if got := verifier.calls; got != 1 {
		t.Fatalf("Verify() calls = %d, want %d", got, 1)
	}
	if got := idGenerator.calls; got != 1 {
		t.Fatalf("SessionIDGenerator calls = %d, want %d", got, 1)
	}
	if got := clock.calls; got != 2 {
		t.Fatalf("Clock calls = %d, want %d", got, 2)
	}
}

type inMemoryCurrentUnlockedSessionStore struct {
	current *session.Session
}

func (store *inMemoryCurrentUnlockedSessionStore) GetCurrent(context.Context) (*session.Session, error) {
	return store.current, nil
}

func (store *inMemoryCurrentUnlockedSessionStore) SaveCurrent(_ context.Context, current *session.Session) error {
	store.current = current
	return nil
}

func (store *inMemoryCurrentUnlockedSessionStore) DeleteCurrent(context.Context) error {
	store.current = nil
	return nil
}

type allowSecretVerifier struct {
	validSecret string
	calls       int
}

func (verifier *allowSecretVerifier) Verify(_ context.Context, secret string) error {
	verifier.calls++
	if secret != verifier.validSecret {
		return errors.New("invalid secret")
	}
	return nil
}

type sequenceClock struct {
	times []time.Time
	calls int
}

func (clock *sequenceClock) Now() time.Time {
	timeValue := clock.times[clock.calls]
	clock.calls++
	return timeValue
}

type fixedSessionIDGenerator struct {
	id    session.SessionID
	calls int
}

func (generator *fixedSessionIDGenerator) New() session.SessionID {
	generator.calls++
	return generator.id
}

var _ ports.CurrentUnlockedSessionStore = (*inMemoryCurrentUnlockedSessionStore)(nil)
var _ ports.UnlockSecretVerifier = (*allowSecretVerifier)(nil)
var _ ports.Clock = (*sequenceClock)(nil)
var _ ports.SessionIDGenerator = (*fixedSessionIDGenerator)(nil)

func mustSessionID(t *testing.T, value string) session.SessionID {
	t.Helper()

	id, err := session.NewSessionID(value)
	if err != nil {
		t.Fatalf("NewSessionID() error = %v", err)
	}

	return id
}
