package session

import "time"

type CreateParams struct {
	ID         SessionID
	UnlockedAt time.Time
}

type Session struct {
	id             SessionID
	status         SessionStatus
	unlockedAt     time.Time
	lastActivityAt time.Time
	lockedAt       *time.Time
}

func New(params CreateParams) (*Session, error) {
	if params.ID.IsZero() {
		return nil, ErrSessionIDRequired
	}
	if params.UnlockedAt.IsZero() {
		return nil, ErrUnlockedAtRequired
	}

	unlockedAt := params.UnlockedAt.UTC()

	return &Session{
		id:             params.ID,
		status:         StatusUnlocked,
		unlockedAt:     unlockedAt,
		lastActivityAt: unlockedAt,
	}, nil
}

func (current *Session) ID() SessionID {
	return current.id
}

func (current *Session) Status() SessionStatus {
	return current.status
}

func (current *Session) UnlockedAt() time.Time {
	return current.unlockedAt
}

func (current *Session) LastActivityAt() time.Time {
	return current.lastActivityAt
}

func (current *Session) Lock(at time.Time) error {
	if at.IsZero() {
		return ErrLockTimeRequired
	}
	if current.status == StatusLocked {
		return nil
	}

	lockedAt := at.UTC()
	if lockedAt.Before(current.unlockedAt) {
		return ErrLockBeforeUnlockTime
	}
	if lockedAt.Before(current.lastActivityAt) {
		return ErrLockBeforeActivityTime
	}

	current.status = StatusLocked
	current.lockedAt = &lockedAt
	return nil
}

func (current *Session) LockedAt() *time.Time {
	if current.lockedAt == nil {
		return nil
	}

	lockedAt := *current.lockedAt
	return &lockedAt
}
