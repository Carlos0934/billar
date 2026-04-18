package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrMissingBearerToken = errors.New("missing bearer token")
	ErrInvalidBearerToken = errors.New("invalid bearer token")
)

type AuthenticatedIdentitySource interface {
	CurrentIdentity(ctx context.Context) (AuthenticatedIdentity, bool, error)
}

type requestIdentityKey struct{}

func WithAuthenticatedIdentity(ctx context.Context, identity AuthenticatedIdentity) context.Context {
	return context.WithValue(ctx, requestIdentityKey{}, identity)
}

type ContextIdentitySource struct{}

func (ContextIdentitySource) CurrentIdentity(ctx context.Context) (AuthenticatedIdentity, bool, error) {
	if err := ctx.Err(); err != nil {
		return AuthenticatedIdentity{}, false, err
	}

	identity, ok := ctx.Value(requestIdentityKey{}).(AuthenticatedIdentity)
	if !ok || strings.TrimSpace(identity.Email) == "" {
		return AuthenticatedIdentity{}, false, nil
	}

	return identity, true, nil
}

type StaticIdentitySource struct {
	identity AuthenticatedIdentity
	ok       bool
}

func NewStaticIdentitySource(identity AuthenticatedIdentity) StaticIdentitySource {
	identity.Email = strings.TrimSpace(identity.Email)
	return StaticIdentitySource{identity: identity, ok: identity.Email != ""}
}

func (s StaticIdentitySource) CurrentIdentity(ctx context.Context) (AuthenticatedIdentity, bool, error) {
	if err := ctx.Err(); err != nil {
		return AuthenticatedIdentity{}, false, err
	}

	if !s.ok || strings.TrimSpace(s.identity.Email) == "" {
		return AuthenticatedIdentity{}, false, nil
	}

	return s.identity, true, nil
}

type RequestSessionService struct {
	identities AuthenticatedIdentitySource
}

func NewRequestSessionService(identities AuthenticatedIdentitySource) RequestSessionService {
	return RequestSessionService{identities: identities}
}

func (s RequestSessionService) Status(ctx context.Context) (SessionStatusDTO, error) {
	if s.identities == nil {
		return SessionStatusDTO{}, errors.New("authenticated identity source is required")
	}

	identity, ok, err := s.identities.CurrentIdentity(ctx)
	if err != nil {
		return SessionStatusDTO{}, fmt.Errorf("load authenticated identity: %w", err)
	}
	if !ok {
		return SessionStatusDTO{Status: "unauthenticated"}, nil
	}

	return SessionStatusDTO{
		Status:        "active",
		Email:         identity.Email,
		EmailVerified: identity.EmailVerified,
		Subject:       identity.Subject,
		Issuer:        identity.Issuer,
	}, nil
}
