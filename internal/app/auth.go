package app

import (
	"context"
	"errors"
	"strings"
)

var (
	ErrAccessTokenRejected  = errors.New("access token rejected")
	ErrUnauthorizedIdentity = errors.New("unauthorized identity")
	ErrEmailNotVerified     = errors.New("email not verified")
)

type AuthenticatedIdentity struct {
	Email         string
	EmailVerified bool
	Subject       string
	Issuer        string
}

type OAuthChallengeDTO struct {
	ResourceURI          string
	AuthorizationServers []string
}

type AccessTokenAuthenticator interface {
	AuthenticateAccessToken(ctx context.Context, rawToken string) (AuthenticatedIdentity, error)
}

type TokenAccessTokenAuthenticator struct {
	ExpectedToken string
	Identity      AuthenticatedIdentity
}

func (v TokenAccessTokenAuthenticator) AuthenticateAccessToken(ctx context.Context, rawToken string) (AuthenticatedIdentity, error) {
	if err := ctx.Err(); err != nil {
		return AuthenticatedIdentity{}, err
	}

	if strings.TrimSpace(rawToken) != strings.TrimSpace(v.ExpectedToken) {
		return AuthenticatedIdentity{}, ErrAccessTokenRejected
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

func (p IdentityPolicy) HasRules() bool {
	return len(p.AllowedEmails) > 0 || len(p.AllowedDomains) > 0
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
