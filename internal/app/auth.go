package app

import (
	"context"
	"errors"
	"strings"
)

var (
	ErrIdentityTokenMismatch = errors.New("identity token mismatch")
	ErrUnauthorizedIdentity  = errors.New("unauthorized identity")
)

type SessionDTO struct {
	ID        string
	Email     string
	ExpiresAt string
}

type AuthenticatedIdentity struct {
	Email         string
	EmailVerified bool
	Subject       string
	Issuer        string
}

type HandleOAuthCallbackCommand struct {
	Code  string
	State string
}

type OAuthChallengeDTO struct {
	ResourceURI          string
	AuthorizationServers []string
}

type IdentityVerifier interface {
	VerifyIDToken(ctx context.Context, rawToken string) (AuthenticatedIdentity, error)
}

type TokenIdentityVerifier struct {
	ExpectedToken string
	Identity      AuthenticatedIdentity
}

func (v TokenIdentityVerifier) VerifyIDToken(ctx context.Context, rawToken string) (AuthenticatedIdentity, error) {
	if err := ctx.Err(); err != nil {
		return AuthenticatedIdentity{}, err
	}

	if strings.TrimSpace(rawToken) != strings.TrimSpace(v.ExpectedToken) {
		return AuthenticatedIdentity{}, ErrIdentityTokenMismatch
	}

	return v.Identity, nil
}

type StateStore interface {
	Generate(ctx context.Context) (string, error)
	Validate(ctx context.Context, state string) error
}

type AccessPolicy interface {
	IsAllowed(email string) bool
}
