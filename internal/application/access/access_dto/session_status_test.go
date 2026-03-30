package accessdto_test

import (
	"testing"
	"time"

	"github.com/Carlos0934/billar/internal/application/access/access_dto"
)

func TestLockedSessionStatus(t *testing.T) {
	status := accessdto.LockedSessionStatus()

	if got := status.Status; got != "locked" {
		t.Fatalf("status = %q, want %q", got, "locked")
	}
	if status.SessionID != nil {
		t.Fatal("expected locked status to omit session id")
	}
	if status.UnlockedAt != nil {
		t.Fatal("expected locked status to omit unlocked at")
	}
	if status.LastActivityAt != nil {
		t.Fatal("expected locked status to omit last activity at")
	}
}

func TestSessionStatusIsTransportNeutral(t *testing.T) {
	sessionID := "session-123"
	unlockedAt := time.Date(2026, time.March, 30, 11, 0, 0, 0, time.UTC)
	lastActivityAt := unlockedAt

	status := accessdto.SessionStatus{
		Status:         "unlocked",
		SessionID:      &sessionID,
		UnlockedAt:     &unlockedAt,
		LastActivityAt: &lastActivityAt,
	}

	if got := status.Status; got != "unlocked" {
		t.Fatalf("status = %q, want %q", got, "unlocked")
	}
	if got := *status.SessionID; got != sessionID {
		t.Fatalf("session id = %q, want %q", got, sessionID)
	}
	if got := *status.UnlockedAt; !got.Equal(unlockedAt) {
		t.Fatalf("unlocked at = %v, want %v", got, unlockedAt)
	}
	if got := *status.LastActivityAt; !got.Equal(lastActivityAt) {
		t.Fatalf("last activity at = %v, want %v", got, lastActivityAt)
	}
}
