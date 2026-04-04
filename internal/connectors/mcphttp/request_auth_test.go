package mcphttp

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Carlos0934/billar/internal/app"
)

type requestAuthenticatorStub struct {
	token    string
	identity app.AuthenticatedIdentity
	err      error
}

func (s *requestAuthenticatorStub) Authenticate(_ context.Context, bearerToken string) (app.AuthenticatedIdentity, error) {
	s.token = bearerToken
	return s.identity, s.err
}

func TestBearerTokenFromHeader(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		header string
		want   string
	}{
		{name: "extracts bearer token", header: "Bearer token-123", want: "token-123"},
		{name: "accepts lowercase scheme", header: "bearer token-123", want: "token-123"},
		{name: "rejects non bearer scheme", header: "Basic token-123", want: ""},
		{name: "rejects missing token", header: "Bearer   ", want: ""},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := bearerTokenFromHeader(tc.header); got != tc.want {
				t.Fatalf("bearerTokenFromHeader() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestMCPHTTPAuthMiddlewareWrap(t *testing.T) {
	t.Parallel()

	challenge := app.OAuthChallengeDTO{ResourceURI: "https://resource.example", AuthorizationServers: []string{"https://issuer.example"}}

	tests := []struct {
		name                  string
		header                string
		authErr               error
		wantStatus            int
		wantToken             string
		wantCalled            bool
		wantWWWAuthenticate   bool
		wantIdentityInContext bool
	}{
		{name: "accepts authenticated request", header: "Bearer token-123", wantStatus: http.StatusNoContent, wantToken: "token-123", wantCalled: true, wantIdentityInContext: true},
		{name: "challenges missing bearer token", authErr: app.ErrMissingBearerToken, wantStatus: http.StatusUnauthorized, wantWWWAuthenticate: true},
		{name: "challenges invalid bearer token", header: "Bearer bad-token", authErr: app.ErrInvalidBearerToken, wantStatus: http.StatusUnauthorized, wantToken: "bad-token", wantWWWAuthenticate: true},
		{name: "forbids unauthorized identity", header: "Bearer token-123", authErr: app.ErrUnauthorizedIdentity, wantStatus: http.StatusForbidden, wantToken: "token-123", wantWWWAuthenticate: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			authenticator := &requestAuthenticatorStub{identity: app.AuthenticatedIdentity{Email: "user@example.com", EmailVerified: true}, err: tc.authErr}
			called := false
			handler := NewMCPHTTPAuthMiddleware(authenticator, challenge, nil).Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				_, ok, err := (app.ContextIdentitySource{}).CurrentIdentity(r.Context())
				if err != nil {
					t.Fatalf("CurrentIdentity() error = %v", err)
				}
				if ok != tc.wantIdentityInContext {
					t.Fatalf("CurrentIdentity() ok = %v, want %v", ok, tc.wantIdentityInContext)
				}
				w.WriteHeader(http.StatusNoContent)
			}))

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "http://example.test/v1/mcp", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}

			handler.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if authenticator.token != tc.wantToken {
				t.Fatalf("Authenticate() token = %q, want %q", authenticator.token, tc.wantToken)
			}
			if called != tc.wantCalled {
				t.Fatalf("next called = %v, want %v", called, tc.wantCalled)
			}
			if tc.wantWWWAuthenticate && rec.Header().Get("WWW-Authenticate") == "" {
				t.Fatal("WWW-Authenticate = empty, want challenge header")
			}
		})
	}
}

func TestMCPHTTPAuthMiddlewareLogsSafeFields(t *testing.T) {
	t.Parallel()

	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	handler := NewMCPHTTPAuthMiddleware(&requestAuthenticatorStub{err: app.ErrInvalidBearerToken}, app.OAuthChallengeDTO{ResourceURI: "https://resource.example"}, logger).Wrap(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next handler should not be called")
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "http://example.test/v1/mcp", nil)
	req.Header.Set("Authorization", "Bearer secret-token")
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	logged := logBuf.String()
	for _, want := range []string{"operation=mcp.request_auth", "connector=mcp-http", "outcome=denied", "reason=invalid_bearer_token"} {
		if !strings.Contains(logged, want) {
			t.Fatalf("log output = %q, want substring %q", logged, want)
		}
	}
	for _, want := range []string{"has_authorization=true", "looks_bearer=true", "path=/v1/mcp"} {
		if !strings.Contains(logged, want) {
			t.Fatalf("log output = %q, want substring %q", logged, want)
		}
	}
	for _, unwanted := range []string{"secret-token", "Authorization"} {
		if strings.Contains(logged, unwanted) {
			t.Fatalf("log output = %q, should not contain %q", logged, unwanted)
		}
	}
}

func TestClassifyRequestAuthReason(t *testing.T) {
	t.Parallel()

	tests := []struct {
		err  error
		want string
	}{
		{err: nil, want: ""},
		{err: app.ErrMissingBearerToken, want: "missing_bearer_token"},
		{err: app.ErrInvalidBearerToken, want: "invalid_bearer_token"},
		{err: errors.New("tokeninfo_audience_mismatch: access token rejected"), want: "tokeninfo_audience_mismatch"},
		{err: errors.New("userinfo_unauthorized: access token rejected"), want: "userinfo_unauthorized"},
		{err: app.ErrUnauthorizedIdentity, want: "unauthorized_identity"},
		{err: app.ErrEmailNotVerified, want: "email_not_verified"},
		{err: errors.New("boom"), want: "internal_error"},
	}

	for _, tc := range tests {
		if got := classifyRequestAuthReason(tc.err); got != tc.want {
			t.Fatalf("classifyRequestAuthReason(%v) = %q, want %q", tc.err, got, tc.want)
		}
	}
}
