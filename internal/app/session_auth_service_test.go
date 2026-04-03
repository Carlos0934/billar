package app

import (
	"context"
	"errors"
	"testing"

	"github.com/Carlos0934/billar/internal/core"
)

type authorizationURLBuilderStub struct {
	loginURL string
	err      error
	state    string
}

func (s *authorizationURLBuilderStub) AuthorizationURL(ctx context.Context, state string) (string, error) {
	_ = ctx
	s.state = state
	return s.loginURL, s.err
}

type oauthCodeExchangerStub struct {
	code     string
	rawToken string
	err      error
}

func (s *oauthCodeExchangerStub) ExchangeCodeForIDToken(ctx context.Context, code string) (string, error) {
	_ = ctx
	s.code = code
	return s.rawToken, s.err
}

type identityVerifierStub struct {
	rawToken string
	identity AuthenticatedIdentity
	err      error
}

func (s *identityVerifierStub) VerifyIDToken(ctx context.Context, rawToken string) (AuthenticatedIdentity, error) {
	_ = ctx
	s.rawToken = rawToken
	return s.identity, s.err
}

type accessPolicyStub struct {
	allowed bool
	email   string
}

func (s *accessPolicyStub) IsAllowed(email string) bool {
	s.email = email
	return s.allowed
}

type stateStoreServiceStub struct {
	state       string
	generateErr error
	validated   string
	validateErr error
}

func (s *stateStoreServiceStub) Generate(ctx context.Context) (string, error) {
	_ = ctx
	return s.state, s.generateErr
}

func (s *stateStoreServiceStub) Validate(ctx context.Context, state string) error {
	_ = ctx
	s.validated = state
	return s.validateErr
}

type sessionStoreStub struct {
	session *core.Session
	saved   *core.Session
	err     error
}

func (s *sessionStoreStub) Save(ctx context.Context, session *core.Session) error {
	_ = ctx
	if s.err != nil {
		return s.err
	}
	if session == nil {
		s.saved = nil
		s.session = nil
		return nil
	}
	copy := *session
	s.saved = &copy
	s.session = &copy
	return nil
}

func (s *sessionStoreStub) GetCurrent(ctx context.Context) (*core.Session, error) {
	_ = ctx
	if s.err != nil {
		return nil, s.err
	}
	if s.session == nil {
		return nil, nil
	}
	copy := *s.session
	return &copy, nil
}

func TestAuthSessionServiceStartLogin(t *testing.T) {
	t.Parallel()

	states := &stateStoreServiceStub{state: "state-123"}
	builder := &authorizationURLBuilderStub{loginURL: "https://accounts.example/auth?state=state-123"}
	svc := NewAuthSessionService(builder, nil, nil, nil, states, nil)

	got, err := svc.StartLogin(context.Background())
	if err != nil {
		t.Fatalf("StartLogin() error = %v", err)
	}
	if got.LoginURL != builder.loginURL {
		t.Fatalf("StartLogin() login URL = %q, want %q", got.LoginURL, builder.loginURL)
	}
	if builder.state != states.state {
		t.Fatalf("AuthorizationURL() state = %q, want %q", builder.state, states.state)
	}
}

func TestAuthSessionServiceHandleOAuthCallback(t *testing.T) {
	t.Parallel()

	exchanger := &oauthCodeExchangerStub{rawToken: "raw-id-token"}
	verifier := &identityVerifierStub{identity: AuthenticatedIdentity{
		Email:         "user@example.com",
		EmailVerified: true,
		Subject:       "subject-123",
		Issuer:        "https://accounts.google.com",
	}}
	policy := &accessPolicyStub{allowed: true}
	sessions := &sessionStoreStub{}
	svc := NewAuthSessionService(nil, exchanger, verifier, policy, nil, sessions)

	got, err := svc.HandleOAuthCallback(context.Background(), HandleOAuthCallbackCommand{Code: "code-123", State: "state-123"})
	if err != nil {
		t.Fatalf("HandleOAuthCallback() error = %v", err)
	}
	if got.ID == "" {
		t.Fatal("HandleOAuthCallback() session ID = empty, want generated ID")
	}
	if got.Email != "user@example.com" {
		t.Fatalf("HandleOAuthCallback() email = %q, want %q", got.Email, "user@example.com")
	}
	if exchanger.code != "code-123" {
		t.Fatalf("ExchangeCodeForIDToken() code = %q, want %q", exchanger.code, "code-123")
	}
	if verifier.rawToken != exchanger.rawToken {
		t.Fatalf("VerifyIDToken() token = %q, want %q", verifier.rawToken, exchanger.rawToken)
	}
	if policy.email != "user@example.com" {
		t.Fatalf("IsAllowed() email = %q, want %q", policy.email, "user@example.com")
	}
	if sessions.saved == nil || sessions.saved.Status != core.SessionStatusActive {
		t.Fatalf("saved session = %+v, want active session", sessions.saved)
	}
	if sessions.saved.Identity.Subject != "subject-123" {
		t.Fatalf("saved subject = %q, want %q", sessions.saved.Identity.Subject, "subject-123")
	}
}

func TestAuthSessionServiceHandleOAuthCallbackRejectsUnauthorizedIdentity(t *testing.T) {
	t.Parallel()

	svc := NewAuthSessionService(
		nil,
		&oauthCodeExchangerStub{rawToken: "raw-id-token"},
		&identityVerifierStub{identity: AuthenticatedIdentity{Email: "user@example.com", EmailVerified: true}},
		&accessPolicyStub{allowed: false},
		nil,
		&sessionStoreStub{},
	)

	_, err := svc.HandleOAuthCallback(context.Background(), HandleOAuthCallbackCommand{Code: "code-123"})
	if !errors.Is(err, ErrUnauthorizedIdentity) {
		t.Fatalf("HandleOAuthCallback() error = %v, want %v", err, ErrUnauthorizedIdentity)
	}
}

func TestAuthSessionServiceHandleOAuthCallbackRejectsUnverifiedEmail(t *testing.T) {
	t.Parallel()

	svc := NewAuthSessionService(
		nil,
		&oauthCodeExchangerStub{rawToken: "raw-id-token"},
		&identityVerifierStub{identity: AuthenticatedIdentity{Email: "user@example.com"}},
		&accessPolicyStub{allowed: true},
		nil,
		&sessionStoreStub{},
	)

	_, err := svc.HandleOAuthCallback(context.Background(), HandleOAuthCallbackCommand{Code: "code-123"})
	if !errors.Is(err, ErrEmailNotVerified) {
		t.Fatalf("HandleOAuthCallback() error = %v, want %v", err, ErrEmailNotVerified)
	}
}

func TestAuthSessionServiceStatusAndLogout(t *testing.T) {
	t.Parallel()

	sessions := &sessionStoreStub{session: &core.Session{
		Status: core.SessionStatusActive,
		ID:     "session-123",
		Identity: core.Identity{
			Email:         "user@example.com",
			EmailVerified: true,
			Subject:       "subject-123",
			Issuer:        "https://accounts.google.com",
		},
	}}
	svc := NewAuthSessionService(nil, nil, nil, nil, nil, sessions)

	status, err := svc.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if status.Status != "active" || status.Email != "user@example.com" || !status.EmailVerified {
		t.Fatalf("Status() = %+v, want active identity", status)
	}

	logout, err := svc.Logout(context.Background())
	if err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	if logout.Message != "Logged out" {
		t.Fatalf("Logout() message = %q, want %q", logout.Message, "Logged out")
	}

	status, err = svc.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() after logout error = %v", err)
	}
	if status.Status != "unauthenticated" {
		t.Fatalf("Status() after logout = %+v, want unauthenticated", status)
	}
}
