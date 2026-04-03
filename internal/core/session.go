package core

import "fmt"

type SessionStatus int

const (
	SessionStatusUnauthenticated SessionStatus = iota
	SessionStatusActive
)

func (s SessionStatus) String() string {
	switch s {
	case SessionStatusActive:
		return "active"
	case SessionStatusUnauthenticated:
		return "unauthenticated"
	default:
		return fmt.Sprintf("session_status(%d)", int(s))
	}
}

type Identity struct {
	Email         string
	EmailVerified bool
	Subject       string
	Issuer        string
}

type Session struct {
	Status   SessionStatus
	Identity Identity
	ID       string
}
