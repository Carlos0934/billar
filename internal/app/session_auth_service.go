package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/Carlos0934/billar/internal/core"
)

var ErrEmailNotVerified = errors.New("email not verified")

type AuthorizationURLBuilder interface {
	AuthorizationURL(ctx context.Context, state string) (string, error)
}

type OAuthCodeExchanger interface {
	ExchangeCodeForIDToken(ctx context.Context, code string) (string, error)
}

type SessionStore interface {
	Save(ctx context.Context, session *core.Session) error
	GetCurrent(ctx context.Context) (*core.Session, error)
}

type AuthSessionService struct {
	authURLBuilder AuthorizationURLBuilder
	codeExchanger  OAuthCodeExchanger
	verifier       IdentityVerifier
	accessPolicy   AccessPolicy
	stateStore     StateStore
	sessions       SessionStore
}

func NewAuthSessionService(
	authURLBuilder AuthorizationURLBuilder,
	codeExchanger OAuthCodeExchanger,
	verifier IdentityVerifier,
	accessPolicy AccessPolicy,
	stateStore StateStore,
	sessions SessionStore,
) AuthSessionService {
	return AuthSessionService{
		authURLBuilder: authURLBuilder,
		codeExchanger:  codeExchanger,
		verifier:       verifier,
		accessPolicy:   accessPolicy,
		stateStore:     stateStore,
		sessions:       sessions,
	}

}

func (s AuthSessionService) StartLogin(ctx context.Context) (LoginIntentDTO, error) {
	if s.stateStore == nil {
		return LoginIntentDTO{}, errors.New("state store is required")
	}
	if s.authURLBuilder == nil {
		return LoginIntentDTO{}, errors.New("authorization url builder is required")
	}

	state, err := s.stateStore.Generate(ctx)
	if err != nil {
		return LoginIntentDTO{}, fmt.Errorf("generate login state: %w", err)
	}

	loginURL, err := s.authURLBuilder.AuthorizationURL(ctx, state)
	if err != nil {
		return LoginIntentDTO{}, fmt.Errorf("build login url: %w", err)
	}

	return LoginIntentDTO{LoginURL: loginURL}, nil
}

func (s AuthSessionService) HandleOAuthCallback(ctx context.Context, cmd HandleOAuthCallbackCommand) (SessionDTO, error) {
	if s.codeExchanger == nil {
		return SessionDTO{}, errors.New("oauth code exchanger is required")
	}
	if s.verifier == nil {
		return SessionDTO{}, errors.New("identity verifier is required")
	}
	if s.accessPolicy == nil {
		return SessionDTO{}, errors.New("access policy is required")
	}
	if s.sessions == nil {
		return SessionDTO{}, errors.New("session store is required")
	}

	rawToken, err := s.codeExchanger.ExchangeCodeForIDToken(ctx, strings.TrimSpace(cmd.Code))
	if err != nil {
		return SessionDTO{}, fmt.Errorf("exchange oauth code: %w", err)
	}

	identity, err := s.verifier.VerifyIDToken(ctx, rawToken)
	if err != nil {
		return SessionDTO{}, fmt.Errorf("verify id token: %w", err)
	}
	if !identity.EmailVerified {
		return SessionDTO{}, ErrEmailNotVerified
	}
	if !s.accessPolicy.IsAllowed(identity.Email) {
		return SessionDTO{}, ErrUnauthorizedIdentity
	}

	sessionID, err := newSessionID()
	if err != nil {
		return SessionDTO{}, fmt.Errorf("generate session id: %w", err)
	}

	session := &core.Session{
		Status: core.SessionStatusActive,
		ID:     sessionID,
		Identity: core.Identity{
			Email:         identity.Email,
			EmailVerified: identity.EmailVerified,
			Subject:       identity.Subject,
			Issuer:        identity.Issuer,
		},
	}
	if err := s.sessions.Save(ctx, session); err != nil {
		return SessionDTO{}, fmt.Errorf("save session: %w", err)
	}

	return SessionDTO{
		ID:    session.ID,
		Email: session.Identity.Email,
	}, nil
}

func (s AuthSessionService) Status(ctx context.Context) (SessionStatusDTO, error) {
	if s.sessions == nil {
		return SessionStatusDTO{}, errors.New("session store is required")
	}

	session, err := s.sessions.GetCurrent(ctx)
	if err != nil {
		return SessionStatusDTO{}, fmt.Errorf("load session: %w", err)
	}
	if session == nil || session.Status != core.SessionStatusActive {
		return SessionStatusDTO{Status: core.SessionStatusUnauthenticated.String()}, nil
	}

	return SessionStatusDTO{
		Status:        session.Status.String(),
		Email:         session.Identity.Email,
		EmailVerified: session.Identity.EmailVerified,
		Subject:       session.Identity.Subject,
		Issuer:        session.Identity.Issuer,
	}, nil
}

func (s AuthSessionService) Logout(ctx context.Context) (LogoutDTO, error) {
	if s.sessions == nil {
		return LogoutDTO{}, errors.New("session store is required")
	}

	if err := s.sessions.Save(ctx, &core.Session{Status: core.SessionStatusUnauthenticated}); err != nil {
		return LogoutDTO{}, fmt.Errorf("clear session: %w", err)
	}

	return LogoutDTO{Message: "Logged out"}, nil
}

func newSessionID() (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}
