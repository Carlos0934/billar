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

type IdentityPolicy struct {
	AllowedEmails  []string
	AllowedDomains []string
}

func (p IdentityPolicy) IsAllowed(email string) bool {
	email = strings.TrimSpace(email)
	if email == "" {
		return false
	}

	for _, allowedEmail := range p.AllowedEmails {
		if strings.EqualFold(email, strings.TrimSpace(allowedEmail)) {
			return true
		}
	}

	_, domain, found := strings.Cut(email, "@")
	if !found {
		return false
	}

	domain = strings.TrimSpace(domain)
	if domain == "" {
		return false
	}

	for _, allowedDomain := range p.AllowedDomains {
		if strings.EqualFold(domain, strings.TrimSpace(allowedDomain)) {
			return true
		}
	}

	return false
}
