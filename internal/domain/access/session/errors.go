package session

import "errors"

var (
	ErrSessionIDRequired      = errors.New("session: id is required")
	ErrUnlockedAtRequired     = errors.New("session: unlocked at is required")
	ErrLockTimeRequired       = errors.New("session: lock time is required")
	ErrLockBeforeUnlockTime   = errors.New("session: lock time cannot be before unlock time")
	ErrLockBeforeActivityTime = errors.New("session: lock time cannot be before last activity time")
)
