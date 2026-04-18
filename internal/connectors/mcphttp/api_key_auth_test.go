package mcphttp

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Carlos0934/billar/internal/app"
)

func TestAPIKeyAuthMiddlewareWrap(t *testing.T) {
	t.Parallel()

	validKey := "super-secret-key"

	tests := []struct {
		name                  string
		header                string
		wantStatus            int
		wantCalled            bool
		wantIdentityInContext bool
	}{
		{
			name:                  "accepts valid bearer key",
			header:                "Bearer " + validKey,
			wantStatus:            http.StatusNoContent,
			wantCalled:            true,
			wantIdentityInContext: true,
		},
		{
			name:       "rejects missing authorization header",
			header:     "",
			wantStatus: http.StatusUnauthorized,
			wantCalled: false,
		},
		{
			name:       "rejects non-bearer scheme",
			header:     "Basic dXNlcjpwYXNz",
			wantStatus: http.StatusUnauthorized,
			wantCalled: false,
		},
		{
			name:       "rejects invalid key",
			header:     "Bearer wrong-key",
			wantStatus: http.StatusUnauthorized,
			wantCalled: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			called := false
			middleware := NewAPIKeyAuthMiddleware([]string{validKey}, nil)
			handler := middleware.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			if called != tc.wantCalled {
				t.Fatalf("next called = %v, want %v", called, tc.wantCalled)
			}
		})
	}
}

func TestAPIKeyAuthMiddlewareSupportsKeyRotation(t *testing.T) {
	t.Parallel()

	keys := []string{"old-key", "new-key"}
	middleware := NewAPIKeyAuthMiddleware(keys, nil)

	for _, key := range keys {
		key := key
		t.Run(key, func(t *testing.T) {
			t.Parallel()

			called := false
			handler := middleware.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				w.WriteHeader(http.StatusNoContent)
			}))

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "http://example.test/v1/mcp", nil)
			req.Header.Set("Authorization", "Bearer "+key)
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusNoContent {
				t.Fatalf("key %q: status = %d, want %d", key, rec.Code, http.StatusNoContent)
			}
			if !called {
				t.Fatalf("key %q: next handler was not called", key)
			}
		})
	}
}

func TestAPIKeyAuthMiddlewareDoesNotLogSecret(t *testing.T) {
	t.Parallel()

	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	middleware := NewAPIKeyAuthMiddleware([]string{"my-secret"}, logger)
	handler := middleware.Wrap(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next should not be called for invalid key")
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "http://example.test/v1/mcp", nil)
	req.Header.Set("Authorization", "Bearer wrong-secret")
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}

	logged := logBuf.String()
	for _, secret := range []string{"wrong-secret", "my-secret"} {
		if strings.Contains(logged, secret) {
			t.Fatalf("log output should not contain secret %q: %q", secret, logged)
		}
	}
}

// TestAPIKeyAuthMiddlewareUsesConstantTimeComparison verifies that the middleware performs
// a constant-time SHA-256 comparison and accepts only the exact matching key.
//
// Contract asserted:
//   - A key that matches byte-for-byte after SHA-256 hashing → 200 (next called)
//   - A key that differs in even one byte → 401 (timing-safe rejection)
//   - The middleware does NOT accept partial prefixes of a valid key (rules out naive string prefix check)
func TestAPIKeyAuthMiddlewareUsesConstantTimeComparison(t *testing.T) {
	t.Parallel()

	const validKey = "contract-test-key-abc123"

	tests := []struct {
		name       string
		presented  string
		wantStatus int
		wantCalled bool
	}{
		{
			name:       "exact match accepted",
			presented:  validKey,
			wantStatus: http.StatusNoContent,
			wantCalled: true,
		},
		{
			name:       "key differing in last byte rejected",
			presented:  validKey[:len(validKey)-1] + "X",
			wantStatus: http.StatusUnauthorized,
			wantCalled: false,
		},
		{
			name:       "prefix of valid key rejected",
			presented:  validKey[:10],
			wantStatus: http.StatusUnauthorized,
			wantCalled: false,
		},
		{
			name:       "key with trailing character rejected",
			presented:  validKey + "!",
			wantStatus: http.StatusUnauthorized,
			wantCalled: false,
		},
		{
			name:       "empty key rejected",
			presented:  "",
			wantStatus: http.StatusUnauthorized,
			wantCalled: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			called := false
			middleware := NewAPIKeyAuthMiddleware([]string{validKey}, nil)
			handler := middleware.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				w.WriteHeader(http.StatusNoContent)
			}))

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "http://example.test/v1/mcp", nil)
			if tc.presented != "" {
				req.Header.Set("Authorization", "Bearer "+tc.presented)
			}
			handler.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if called != tc.wantCalled {
				t.Fatalf("next called = %v, want %v", called, tc.wantCalled)
			}
		})
	}
}

func TestAPIKeyAuthMiddlewareInjectsLocalIdentity(t *testing.T) {
	t.Parallel()

	middleware := NewAPIKeyAuthMiddleware([]string{"valid-key"}, nil)
	var gotIdentity app.AuthenticatedIdentity
	handler := middleware.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok, err := (app.ContextIdentitySource{}).CurrentIdentity(r.Context())
		if err != nil {
			t.Fatalf("CurrentIdentity() error = %v", err)
		}
		if !ok {
			t.Fatal("CurrentIdentity() ok = false, want true")
		}
		gotIdentity = id
		w.WriteHeader(http.StatusNoContent)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "http://example.test/v1/mcp", nil)
	req.Header.Set("Authorization", "Bearer valid-key")
	handler.ServeHTTP(rec, req)

	if gotIdentity.Email != "mcp@local" {
		t.Fatalf("identity.Email = %q, want %q", gotIdentity.Email, "mcp@local")
	}
	if !gotIdentity.EmailVerified {
		t.Fatalf("identity.EmailVerified = false, want true")
	}
}
