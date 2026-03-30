package session

import "strings"

type SessionStatus string

const (
	StatusLocked   SessionStatus = "locked"
	StatusUnlocked SessionStatus = "unlocked"
)

type SessionID struct {
	value string
}

func NewSessionID(value string) (SessionID, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return SessionID{}, ErrSessionIDRequired
	}

	return SessionID{value: trimmed}, nil
}

func (id SessionID) String() string {
	return id.value
}

func (id SessionID) IsZero() bool {
	return id.value == ""
}
