package accessdto

import (
	"time"

	"github.com/Carlos0934/billar/internal/domain/access/session"
)

type SessionStatus struct {
	Status         string
	SessionID      *string
	UnlockedAt     *time.Time
	LastActivityAt *time.Time
}

func LockedSessionStatus() SessionStatus {
	return SessionStatus{Status: string(session.StatusLocked)}
}

func UnlockedSessionStatus(current *session.Session) SessionStatus {
	sessionID := current.ID().String()
	unlockedAt := current.UnlockedAt()
	lastActivityAt := current.LastActivityAt()

	return SessionStatus{
		Status:         string(session.StatusUnlocked),
		SessionID:      &sessionID,
		UnlockedAt:     &unlockedAt,
		LastActivityAt: &lastActivityAt,
	}
}
