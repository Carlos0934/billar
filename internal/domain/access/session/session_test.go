package session_test

import (
	"testing"
	"time"

	"github.com/Carlos0934/billar/internal/domain/access/session"
)

func TestNewSessionCreatesUnlockedSessionWithMatchingActivityTimestamp(t *testing.T) {
	sessionID := mustSessionID(t, "session-123")
	unlockedAt := time.Date(2026, time.March, 30, 12, 0, 0, 0, time.UTC)

	current, err := session.New(session.CreateParams{
		ID:         sessionID,
		UnlockedAt: unlockedAt,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if got := current.ID().String(); got != sessionID.String() {
		t.Fatalf("id = %q, want %q", got, sessionID.String())
	}
	if got := current.Status(); got != session.StatusUnlocked {
		t.Fatalf("status = %q, want %q", got, session.StatusUnlocked)
	}
	if got := current.UnlockedAt(); !got.Equal(unlockedAt) {
		t.Fatalf("unlocked at = %v, want %v", got, unlockedAt)
	}
	if got := current.LastActivityAt(); !got.Equal(unlockedAt) {
		t.Fatalf("last activity at = %v, want %v", got, unlockedAt)
	}
}

func TestNewSessionRejectsMissingIDOrUnlockedTime(t *testing.T) {
	tests := []struct {
		name   string
		params session.CreateParams
	}{
		{
			name: "missing id",
			params: session.CreateParams{
				UnlockedAt: time.Date(2026, time.March, 30, 12, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "missing unlocked time",
			params: session.CreateParams{
				ID: mustSessionID(t, "session-123"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := session.New(tt.params)
			if err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestSessionLockMarksSessionLockedAndIsIdempotent(t *testing.T) {
	unlockedAt := time.Date(2026, time.March, 30, 12, 0, 0, 0, time.UTC)
	current, err := session.New(session.CreateParams{
		ID:         mustSessionID(t, "session-123"),
		UnlockedAt: unlockedAt,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	lockedAt := unlockedAt.Add(2 * time.Hour)
	if err := current.Lock(lockedAt); err != nil {
		t.Fatalf("Lock() error = %v", err)
	}

	if got := current.Status(); got != session.StatusLocked {
		t.Fatalf("status = %q, want %q", got, session.StatusLocked)
	}
	if got := current.LockedAt(); got == nil || !got.Equal(lockedAt) {
		t.Fatalf("locked at = %v, want %v", got, lockedAt)
	}
	if got := current.LastActivityAt(); !got.Equal(unlockedAt) {
		t.Fatalf("last activity at = %v, want %v", got, unlockedAt)
	}

	secondLockedAt := lockedAt.Add(1 * time.Hour)
	if err := current.Lock(secondLockedAt); err != nil {
		t.Fatalf("Lock() repeated error = %v", err)
	}
	if got := current.LockedAt(); got == nil || !got.Equal(lockedAt) {
		t.Fatalf("locked at after repeated lock = %v, want %v", got, lockedAt)
	}
}

func TestSessionLockRejectsInvalidLockTime(t *testing.T) {
	unlockedAt := time.Date(2026, time.March, 30, 12, 0, 0, 0, time.UTC)
	current, err := session.New(session.CreateParams{
		ID:         mustSessionID(t, "session-123"),
		UnlockedAt: unlockedAt,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	tests := []struct {
		name string
		at   time.Time
	}{
		{name: "zero time"},
		{name: "before unlock", at: unlockedAt.Add(-time.Second)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := current.Lock(tt.at); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func mustSessionID(t *testing.T, value string) session.SessionID {
	t.Helper()

	id, err := session.NewSessionID(value)
	if err != nil {
		t.Fatalf("NewSessionID() error = %v", err)
	}

	return id
}
