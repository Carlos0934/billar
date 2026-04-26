package app

import (
	"errors"
)

var (
	ErrUnauthorizedIdentity = errors.New("unauthorized identity")
	ErrEmailNotVerified     = errors.New("email not verified")
)

type AuthenticatedIdentity struct {
	Email         string
	EmailVerified bool
	Subject       string
	Issuer        string
}
