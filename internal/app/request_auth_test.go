package app

import (
	"context"
	"errors"
	"testing"
)

type identitySourceStub struct {
	identity AuthenticatedIdentity
	ok       bool
	err      error
}

func (s identitySourceStub) CurrentIdentity(context.Context) (AuthenticatedIdentity, bool, error) {
	return s.identity, s.ok, s.err
}

func TestRequestAuthServiceAuthenticate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		service RequestAuthService
		token   string
		want    AuthenticatedIdentity
		wantErr error
	}{
		{
			name: "accepts verified allowed identity",
			service: NewRequestAuthService(
				TokenAccessTokenAuthenticator{ExpectedToken: "token-123", Identity: AuthenticatedIdentity{Email: "user@example.com", EmailVerified: true, Subject: "sub-123", Issuer: "https://issuer.example"}},
				IdentityPolicy{AllowedDomains: []string{"example.com"}},
			),
			token: "token-123",
			want:  AuthenticatedIdentity{Email: "user@example.com", EmailVerified: true, Subject: "sub-123", Issuer: "https://issuer.example"},
		},
		{
			name: "rejects missing bearer token",
			service: NewRequestAuthService(
				TokenAccessTokenAuthenticator{ExpectedToken: "token-123", Identity: AuthenticatedIdentity{Email: "user@example.com", EmailVerified: true}},
				IdentityPolicy{AllowedDomains: []string{"example.com"}},
			),
			wantErr: ErrMissingBearerToken,
		},
		{
			name: "rejects invalid bearer token",
			service: NewRequestAuthService(
				TokenAccessTokenAuthenticator{ExpectedToken: "token-123", Identity: AuthenticatedIdentity{Email: "user@example.com", EmailVerified: true}},
				IdentityPolicy{AllowedDomains: []string{"example.com"}},
			),
			token:   "wrong-token",
			wantErr: ErrInvalidBearerToken,
		},
		{
			name: "rejects unverified email",
			service: NewRequestAuthService(
				TokenAccessTokenAuthenticator{ExpectedToken: "token-123", Identity: AuthenticatedIdentity{Email: "user@example.com"}},
				IdentityPolicy{AllowedDomains: []string{"example.com"}},
			),
			token:   "token-123",
			wantErr: ErrEmailNotVerified,
		},
		{
			name: "rejects unauthorized identity",
			service: NewRequestAuthService(
				TokenAccessTokenAuthenticator{ExpectedToken: "token-123", Identity: AuthenticatedIdentity{Email: "user@other.com", EmailVerified: true}},
				IdentityPolicy{AllowedDomains: []string{"example.com"}},
			),
			token:   "token-123",
			wantErr: ErrUnauthorizedIdentity,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := tc.service.Authenticate(context.Background(), tc.token)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("Authenticate() error = %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Authenticate() error = %v", err)
			}
			if got != tc.want {
				t.Fatalf("Authenticate() = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestContextIdentitySourceCurrentIdentity(t *testing.T) {
	t.Parallel()

	identity := AuthenticatedIdentity{Email: "user@example.com", EmailVerified: true}
	got, ok, err := (ContextIdentitySource{}).CurrentIdentity(WithAuthenticatedIdentity(context.Background(), identity))
	if err != nil {
		t.Fatalf("CurrentIdentity() error = %v", err)
	}
	if !ok {
		t.Fatal("CurrentIdentity() ok = false, want true")
	}
	if got != identity {
		t.Fatalf("CurrentIdentity() = %+v, want %+v", got, identity)
	}
}

func TestStaticIdentitySourceCurrentIdentity(t *testing.T) {
	t.Parallel()

	source := NewStaticIdentitySource(AuthenticatedIdentity{Email: "user@example.com", EmailVerified: true, Subject: "sub-123", Issuer: "https://issuer.example"})

	got, ok, err := source.CurrentIdentity(context.Background())
	if err != nil {
		t.Fatalf("CurrentIdentity() error = %v", err)
	}
	if !ok {
		t.Fatal("CurrentIdentity() ok = false, want true")
	}
	if got != (AuthenticatedIdentity{Email: "user@example.com", EmailVerified: true, Subject: "sub-123", Issuer: "https://issuer.example"}) {
		t.Fatalf("CurrentIdentity() = %+v", got)
	}
}

func TestNewLocalBypassIdentitySource(t *testing.T) {
	t.Parallel()

	source, err := NewLocalBypassIdentitySource("user@example.com", IdentityPolicy{AllowedDomains: []string{"example.com"}})
	if err != nil {
		t.Fatalf("NewLocalBypassIdentitySource() error = %v", err)
	}

	got, ok, err := source.CurrentIdentity(context.Background())
	if err != nil {
		t.Fatalf("CurrentIdentity() error = %v", err)
	}
	if !ok {
		t.Fatal("CurrentIdentity() ok = false, want true")
	}
	if got.Email != "user@example.com" || got.Issuer != "billar://local" || got.Subject != "local-bypass" || !got.EmailVerified {
		t.Fatalf("CurrentIdentity() = %+v", got)
	}

	if _, err := NewLocalBypassIdentitySource("user@other.com", IdentityPolicy{AllowedDomains: []string{"example.com"}}); err == nil {
		t.Fatal("NewLocalBypassIdentitySource() error = nil, want policy rejection")
	}
}

func TestRequestSessionServiceStatus(t *testing.T) {
	t.Parallel()

	svc := NewRequestSessionService(identitySourceStub{identity: AuthenticatedIdentity{Email: "user@example.com", EmailVerified: true, Subject: "sub-123", Issuer: "https://issuer.example"}, ok: true})
	status, err := svc.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if status != (SessionStatusDTO{Status: "active", Email: "user@example.com", EmailVerified: true, Subject: "sub-123", Issuer: "https://issuer.example"}) {
		t.Fatalf("Status() = %+v", status)
	}
}
