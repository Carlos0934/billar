package mcphttp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
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
	// Verify message_classified event
	for _, want := range []string{"operation=mcp.request_auth", "connector=mcp-http", "outcome=message_classified", "message_type=invalid", "method=", "has_bearer_token=true", "body_parseable=false"} {
		if !strings.Contains(logged, want) {
			t.Fatalf("log output missing %q: %q", want, logged)
		}
	}
	// Verify denied event with enhanced fields
	for _, want := range []string{"outcome=denied", "reason=invalid_bearer_token"} {
		if !strings.Contains(logged, want) {
			t.Fatalf("log output missing %q: %q", want, logged)
		}
	}
	// Verify received event
	for _, want := range []string{"outcome=received", "has_authorization=true", "looks_bearer=true", "path=/v1/mcp"} {
		if !strings.Contains(logged, want) {
			t.Fatalf("log output missing %q: %q", want, logged)
		}
	}
	// Ensure no sensitive data is logged
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

func TestMCPHTTPAuthMiddlewareChallengesAllUnauthenticated(t *testing.T) {
	t.Parallel()

	// All methods — including formerly-allowlisted ones — must return 401 when unauthenticated.
	methods := []string{
		"initialize",
		"notifications/initialized",
		"tools/list",
		"tools/call",
		"resources/read",
	}

	for _, method := range methods {
		method := method
		t.Run(method, func(t *testing.T) {
			t.Parallel()
			authenticator := &requestAuthenticatorStub{err: app.ErrMissingBearerToken}
			handler := NewMCPHTTPAuthMiddleware(authenticator, app.OAuthChallengeDTO{ResourceURI: "https://resource.example"}, nil).Wrap(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				t.Fatalf("next should not be called for unauthenticated %q", method)
			}))
			body := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":%q,"params":{}}`, method)
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "http://example.test/v1/mcp", strings.NewReader(body))
			handler.ServeHTTP(rec, req)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("method %q: status = %d, want %d", method, rec.Code, http.StatusUnauthorized)
			}
			if rec.Header().Get("WWW-Authenticate") == "" {
				t.Fatalf("method %q: WWW-Authenticate header missing", method)
			}
		})
	}
}

func TestPeekJSONRPCMethod(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body []byte
		want string
	}{
		{name: "valid JSON with method", body: []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`), want: "initialize"},
		{name: "valid JSON with different method", body: []byte(`{"jsonrpc":"2.0","method":"tools/call"}`), want: "tools/call"},
		{name: "malformed JSON", body: []byte(`{invalid json}`), want: ""},
		{name: "JSON array (batch)", body: []byte(`[{"jsonrpc":"2.0","method":"initialize"}]`), want: ""},
		{name: "missing method field", body: []byte(`{"jsonrpc":"2.0","id":1}`), want: ""},
		{name: "empty body", body: []byte(``), want: ""},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := peekJSONRPCMethod(tc.body); got != tc.want {
				t.Fatalf("peekJSONRPCMethod(%q) = %q, want %q", string(tc.body), got, tc.want)
			}
		})
	}
}

func TestMCPHTTPAuthMiddlewareBodyBuffering(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{name: "buffered body readable downstream", body: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{}}`, wantStatus: http.StatusNoContent},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			authenticator := &requestAuthenticatorStub{err: nil}
			downstreamBody := ""
			handler := NewMCPHTTPAuthMiddleware(authenticator, app.OAuthChallengeDTO{}, nil).Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				buf, err := io.ReadAll(r.Body)
				if err != nil {
					t.Fatalf("downstream ReadAll error: %v", err)
				}
				downstreamBody = string(buf)
				w.WriteHeader(http.StatusNoContent)
			}))

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "http://example.test/v1/mcp", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")

			handler.ServeHTTP(rec, req)

			// Downstream should receive the full body even after middleware has peeked at it
			if downstreamBody != tc.body {
				t.Fatalf("downstream body = %q, want %q", downstreamBody, tc.body)
			}
		})
	}
}

func TestMCPHTTPAuthMiddlewareAllowlistAuthLogic(t *testing.T) {
	t.Parallel()

	challenge := app.OAuthChallengeDTO{ResourceURI: "https://resource.example"}

	tests := []struct {
		name                  string
		body                  string
		authHeader            string
		authErr               error
		wantStatus            int
		wantIdentityInContext bool
		wantWWWAuthenticate   bool
	}{
		{
			name:                  "initialize with no token rejects with 401",
			body:                  `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`,
			authHeader:            "",
			authErr:               app.ErrMissingBearerToken,
			wantStatus:            http.StatusUnauthorized,
			wantIdentityInContext: false,
			wantWWWAuthenticate:   true,
		},
		{
			name:                  "tools/list with no token rejects with 401",
			body:                  `{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`,
			authHeader:            "",
			authErr:               app.ErrMissingBearerToken,
			wantStatus:            http.StatusUnauthorized,
			wantIdentityInContext: false,
			wantWWWAuthenticate:   true,
		},
		{
			name:                  "notifications/initialized with no token rejects with 401",
			body:                  `{"jsonrpc":"2.0","id":3,"method":"notifications/initialized","params":{}}`,
			authHeader:            "",
			authErr:               app.ErrMissingBearerToken,
			wantStatus:            http.StatusUnauthorized,
			wantIdentityInContext: false,
			wantWWWAuthenticate:   true,
		},
		{
			name:                  "initialize with valid token adds identity to context",
			body:                  `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`,
			authHeader:            "Bearer token-123",
			authErr:               nil,
			wantStatus:            http.StatusNoContent,
			wantIdentityInContext: true,
			wantWWWAuthenticate:   false,
		},
		{
			name:                  "tools/call with no token rejects with 401",
			body:                  `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{}}`,
			authHeader:            "",
			authErr:               app.ErrMissingBearerToken,
			wantStatus:            http.StatusUnauthorized,
			wantIdentityInContext: false,
			wantWWWAuthenticate:   true,
		},
		{
			name:                  "tools/call with valid token allows request",
			body:                  `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{}}`,
			authHeader:            "Bearer token-123",
			authErr:               nil,
			wantStatus:            http.StatusNoContent,
			wantIdentityInContext: true,
			wantWWWAuthenticate:   false,
		},
		{
			name:                  "malformed body with no token rejects with 401 fail-closed",
			body:                  `{invalid json}`,
			authHeader:            "",
			authErr:               app.ErrMissingBearerToken,
			wantStatus:            http.StatusUnauthorized,
			wantIdentityInContext: false,
			wantWWWAuthenticate:   true,
		},
		{
			name:                  "missing method field with no token rejects with 401 fail-closed",
			body:                  `{"jsonrpc":"2.0","id":1}`,
			authHeader:            "",
			authErr:               app.ErrMissingBearerToken,
			wantStatus:            http.StatusUnauthorized,
			wantIdentityInContext: false,
			wantWWWAuthenticate:   true,
		},
		{
			name:                  "JSON array batch request with no token rejects with 401 fail-closed",
			body:                  `[{"jsonrpc":"2.0","method":"initialize"}]`,
			authHeader:            "",
			authErr:               app.ErrMissingBearerToken,
			wantStatus:            http.StatusUnauthorized,
			wantIdentityInContext: false,
			wantWWWAuthenticate:   true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			authenticator := &requestAuthenticatorStub{
				identity: app.AuthenticatedIdentity{Email: "user@example.com", EmailVerified: true},
				err:      tc.authErr,
			}
			called := false
			downstreamBody := ""
			handler := NewMCPHTTPAuthMiddleware(authenticator, challenge, nil).Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				buf, err := io.ReadAll(r.Body)
				if err != nil {
					t.Fatalf("downstream ReadAll error: %v", err)
				}
				downstreamBody = string(buf)
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
			req := httptest.NewRequest(http.MethodPost, "http://example.test/v1/mcp", strings.NewReader(tc.body))
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}

			handler.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if called != (tc.wantStatus == http.StatusNoContent) {
				t.Fatalf("next called = %v, want %v", called, tc.wantStatus == http.StatusNoContent)
			}
			if tc.wantStatus == http.StatusNoContent && downstreamBody != tc.body {
				t.Fatalf("downstream body = %q, want %q", downstreamBody, tc.body)
			}
			if tc.wantWWWAuthenticate && rec.Header().Get("WWW-Authenticate") == "" {
				t.Fatal("WWW-Authenticate = empty, want challenge header")
			}
			if !tc.wantWWWAuthenticate && rec.Header().Get("WWW-Authenticate") != "" {
				t.Fatalf("WWW-Authenticate = %q, want empty", rec.Header().Get("WWW-Authenticate"))
			}
		})
	}
}

func TestMCPHTTPAuthMiddlewareChallengesInitializeWithoutToken(t *testing.T) {
	t.Parallel()

	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	authenticator := &requestAuthenticatorStub{err: app.ErrMissingBearerToken}
	handler := NewMCPHTTPAuthMiddleware(authenticator, app.OAuthChallengeDTO{ResourceURI: "https://resource.example"}, logger).Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called for unauthenticated initialize")
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "http://example.test/v1/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`))

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if rec.Header().Get("WWW-Authenticate") == "" {
		t.Fatal("WWW-Authenticate header missing for unauthenticated initialize request")
	}

	logged := logBuf.String()
	if !strings.Contains(logged, "outcome=message_classified") {
		t.Fatalf("log output missing message_classified event: %q", logged)
	}
	if !strings.Contains(logged, "method=initialize") {
		t.Fatalf("log output missing method=initialize: %q", logged)
	}
	if !strings.Contains(logged, "outcome=denied") {
		t.Fatalf("log output missing denied event: %q", logged)
	}
}

func TestMCPHTTPAuthMiddlewareLogsAuthenticatedInitialize(t *testing.T) {
	t.Parallel()

	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	authenticator := &requestAuthenticatorStub{identity: app.AuthenticatedIdentity{Email: "user@example.com", EmailVerified: true}, err: nil}
	handler := NewMCPHTTPAuthMiddleware(authenticator, app.OAuthChallengeDTO{ResourceURI: "https://resource.example"}, logger).Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "http://example.test/v1/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`))
	req.Header.Set("Authorization", "Bearer token-123")

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}

	logged := logBuf.String()
	if !strings.Contains(logged, "outcome=message_classified") {
		t.Fatalf("log output missing message_classified event: %q", logged)
	}
	if !strings.Contains(logged, "method=initialize") {
		t.Fatalf("log output missing method=initialize: %q", logged)
	}
	if !strings.Contains(logged, "has_bearer_token=true") {
		t.Fatalf("log output missing has_bearer_token=true: %q", logged)
	}
	if !strings.Contains(logged, "message_type=request") {
		t.Fatalf("log output missing message_type=request: %q", logged)
	}
}

func TestMCPHTTPAuthMiddlewareLogsDeniedWithMethodContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		body            string
		wantParseable   bool
		wantMethod      string
		wantMessageType string
	}{
		{name: "initialize denied without token", body: `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`, wantParseable: true, wantMethod: "initialize", wantMessageType: "request"},
		{name: "non-discovery method denied", body: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{}}`, wantParseable: true, wantMethod: "tools/call", wantMessageType: "request"},
		{name: "malformed body denied", body: `{invalid json}`, wantParseable: false, wantMethod: "", wantMessageType: "invalid"},
		{name: "missing method field denied", body: `{"jsonrpc":"2.0","id":1}`, wantParseable: true, wantMethod: "", wantMessageType: "unknown"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var logBuf bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelDebug}))

			authenticator := &requestAuthenticatorStub{err: app.ErrMissingBearerToken}
			handler := NewMCPHTTPAuthMiddleware(authenticator, app.OAuthChallengeDTO{ResourceURI: "https://resource.example"}, logger).Wrap(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				t.Fatal("next handler should not be called")
			}))

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "http://example.test/v1/mcp", strings.NewReader(tc.body))

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
			}

			logged := logBuf.String()
			// Verify denied event includes method context
			if !strings.Contains(logged, "outcome=denied") {
				t.Fatalf("log output missing denied event: %q", logged)
			}
			if !strings.Contains(logged, "reason=missing_bearer_token") {
				t.Fatalf("log output missing reason=missing_bearer_token: %q", logged)
			}
			if !strings.Contains(logged, fmt.Sprintf("method=%s", tc.wantMethod)) {
				t.Fatalf("log output missing method=%s: %q", tc.wantMethod, logged)
			}
			if !strings.Contains(logged, fmt.Sprintf("message_type=%s", tc.wantMessageType)) {
				t.Fatalf("log output missing message_type=%s: %q", tc.wantMessageType, logged)
			}
			if !strings.Contains(logged, fmt.Sprintf("body_parseable=%v", tc.wantParseable)) {
				t.Fatalf("log output missing body_parseable=%v: %q", tc.wantParseable, logged)
			}
		})
	}
}

func TestClassifyJSONRPCMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		body              string
		wantType          JSONRPCMessageType
		wantMethod        string
		wantHasID         bool
		wantBodyParseable bool
	}{
		// Request cases (has method and id)
		{name: "request with method and id", body: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{}}`, wantType: JSONRPCMessageRequest, wantMethod: "tools/call", wantHasID: true, wantBodyParseable: true},
		{name: "request with string id", body: `{"jsonrpc":"2.0","id":"abc","method":"initialize","params":{}}`, wantType: JSONRPCMessageRequest, wantMethod: "initialize", wantHasID: true, wantBodyParseable: true},

		// Notification cases (has method, no id)
		{name: "notification with method no id", body: `{"jsonrpc":"2.0","method":"notifications/initialized","params":{}}`, wantType: JSONRPCMessageNotification, wantMethod: "notifications/initialized", wantHasID: false, wantBodyParseable: true},

		// Response cases (has result or error)
		{name: "response with result", body: `{"jsonrpc":"2.0","id":1,"result":{"capabilities":{}}}`, wantType: JSONRPCMessageResponse, wantMethod: "", wantHasID: true, wantBodyParseable: true},
		{name: "response with error", body: `{"jsonrpc":"2.0","id":1,"error":{"code":-32600,"message":"Invalid Request"}}`, wantType: JSONRPCMessageResponse, wantMethod: "", wantHasID: true, wantBodyParseable: true},
		{name: "response with result and no id", body: `{"jsonrpc":"2.0","result":{}}`, wantType: JSONRPCMessageResponse, wantMethod: "", wantHasID: false, wantBodyParseable: true},

		// Batch cases (array)
		{name: "batch request array", body: `[{"jsonrpc":"2.0","id":1,"method":"tools/call"},{"jsonrpc":"2.0","id":2,"method":"resources/list"}]`, wantType: JSONRPCMessageBatch, wantMethod: "", wantHasID: false, wantBodyParseable: true},
		{name: "empty batch array", body: `[]`, wantType: JSONRPCMessageBatch, wantMethod: "", wantHasID: false, wantBodyParseable: true},
		{name: "batch with whitespace", body: `  [ {"jsonrpc":"2.0"} ]  `, wantType: JSONRPCMessageBatch, wantMethod: "", wantHasID: false, wantBodyParseable: true},

		// Invalid cases
		{name: "malformed JSON", body: `{invalid json}`, wantType: JSONRPCMessageInvalid, wantMethod: "", wantHasID: false, wantBodyParseable: false},
		{name: "empty body", body: ``, wantType: JSONRPCMessageInvalid, wantMethod: "", wantHasID: false, wantBodyParseable: false},

		// Unknown cases (valid JSON but no recognizable fields)
		{name: "unknown with only jsonrpc field", body: `{"jsonrpc":"2.0"}`, wantType: JSONRPCMessageUnknown, wantMethod: "", wantHasID: false, wantBodyParseable: true},
		{name: "unknown with only id field", body: `{"jsonrpc":"2.0","id":1}`, wantType: JSONRPCMessageUnknown, wantMethod: "", wantHasID: true, wantBodyParseable: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			info := classifyJSONRPCMessage([]byte(tc.body))
			if info.Type != tc.wantType {
				t.Fatalf("classifyJSONRPCMessage(%q).Type = %q, want %q", tc.body, info.Type, tc.wantType)
			}
			if info.Method != tc.wantMethod {
				t.Fatalf("classifyJSONRPCMessage(%q).Method = %q, want %q", tc.body, info.Method, tc.wantMethod)
			}
			if info.HasID != tc.wantHasID {
				t.Fatalf("classifyJSONRPCMessage(%q).HasID = %v, want %v", tc.body, info.HasID, tc.wantHasID)
			}
			if info.BodyParseable != tc.wantBodyParseable {
				t.Fatalf("classifyJSONRPCMessage(%q).BodyParseable = %v, want %v", tc.body, info.BodyParseable, tc.wantBodyParseable)
			}
		})
	}
}

func TestIsArray(t *testing.T) {
	t.Parallel()

	tests := []struct {
		body string
		want bool
	}{
		{body: `[]`, want: true},
		{body: `  []`, want: true},
		{body: "\n\t[]", want: true},
		{body: `[{"jsonrpc":"2.0"}]`, want: true},
		{body: `{}`, want: false},
		{body: `{"jsonrpc":"2.0"}`, want: false},
		{body: ``, want: false},
		{body: `"test"`, want: false},
		{body: `  {}`, want: false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.body, func(t *testing.T) {
			t.Parallel()
			if got := isArray([]byte(tc.body)); got != tc.want {
				t.Fatalf("isArray(%q) = %v, want %v", tc.body, got, tc.want)
			}
		})
	}
}

func TestMCPHTTPAuthMiddlewareLogsMessageType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		body            string
		wantMessageType string
		wantMethod      string
	}{
		// Response message (client sent response to server)
		{name: "response message logged correctly", body: `{"jsonrpc":"2.0","id":1,"result":{"capabilities":{}}}`, wantMessageType: "response", wantMethod: ""},

		// Notification message
		{name: "notification message logged correctly", body: `{"jsonrpc":"2.0","method":"notifications/progress"}`, wantMessageType: "notification", wantMethod: "notifications/progress"},

		// Batch message
		{name: "batch message logged correctly", body: `[{"jsonrpc":"2.0","method":"initialize"}]`, wantMessageType: "batch", wantMethod: ""},

		// Invalid message
		{name: "invalid message logged correctly", body: `{malformed json}`, wantMessageType: "invalid", wantMethod: ""},

		// Unknown message (valid JSON but no method/result/error)
		{name: "unknown message logged correctly", body: `{"jsonrpc":"2.0","id":"test"}`, wantMessageType: "unknown", wantMethod: ""},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var logBuf bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelDebug}))

			authenticator := &requestAuthenticatorStub{err: app.ErrMissingBearerToken}
			handler := NewMCPHTTPAuthMiddleware(authenticator, app.OAuthChallengeDTO{ResourceURI: "https://resource.example"}, logger).Wrap(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				t.Fatal("next handler should not be called")
			}))

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "http://example.test/v1/mcp", strings.NewReader(tc.body))

			handler.ServeHTTP(rec, req)

			logged := logBuf.String()
			if !strings.Contains(logged, fmt.Sprintf("message_type=%s", tc.wantMessageType)) {
				t.Fatalf("log output missing message_type=%s: %q", tc.wantMessageType, logged)
			}
			if tc.wantMethod != "" && !strings.Contains(logged, fmt.Sprintf("method=%s", tc.wantMethod)) {
				t.Fatalf("log output missing method=%s: %q", tc.wantMethod, logged)
			}
		})
	}
}
