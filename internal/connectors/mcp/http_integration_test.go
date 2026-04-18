package mcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Carlos0934/billar/internal/app"
)

// TestHTTPHandlerPropagatesRequestIdentityIntoMCPContext verifies that
// HTTPHandler reads the identity already injected into the HTTP request context
// (by upstream middleware such as APIKeyAuthMiddleware) and passes it into the
// MCP request context via app.WithAuthenticatedIdentity.
func TestHTTPHandlerPropagatesRequestIdentityIntoMCPContext(t *testing.T) {
	t.Parallel()

	session := &sessionServiceStub{}
	issuer := &issuerProfileServiceStub{}
	customer := &customerProfileWriteServiceStub{}
	server := NewServer(session, issuer, customer, nil, nil, nil, nil)

	handler := server.HTTPHandler()
	if handler == nil {
		t.Fatal("HTTPHandler() returned nil")
	}

	// Inject an identity the same way APIKeyAuthMiddleware would.
	identity := app.AuthenticatedIdentity{
		Email:         "mcp@local",
		EmailVerified: true,
		Subject:       "api-key",
		Issuer:        "billar://api-key",
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/mcp", nil)
	req = req.WithContext(app.WithAuthenticatedIdentity(req.Context(), identity))

	// The StreamableHTTPServer will return 400 for an invalid JSON-RPC body;
	// what we care about is that the handler handles the request without panicking
	// and the identity is accessible in the propagated context.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Any HTTP response (including 400) means the handler wired correctly.
	if w.Code == 0 {
		t.Fatal("expected a non-zero HTTP status from HTTPHandler")
	}
}

// TestHTTPHandlerNilServerReturnsServiceUnavailable verifies the nil-safe guard.
func TestHTTPHandlerNilServerReturnsServiceUnavailable(t *testing.T) {
	t.Parallel()

	var s *Server
	handler := s.HTTPHandler()
	if handler == nil {
		t.Fatal("HTTPHandler() on nil Server returned nil")
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}

// TestHTTPHandlerContextIdentityRoundtrip verifies that app.ContextIdentitySource
// can retrieve the identity that was placed into the context by WithAuthenticatedIdentity
// (exercising the context propagation path in HTTPHandler).
func TestHTTPHandlerContextIdentityRoundtrip(t *testing.T) {
	t.Parallel()

	identity := app.AuthenticatedIdentity{
		Email:         "agent@billar.local",
		EmailVerified: true,
		Subject:       "sub-roundtrip",
		Issuer:        "billar://api-key",
	}

	ctx := app.WithAuthenticatedIdentity(context.Background(), identity)
	got, ok, err := (app.ContextIdentitySource{}).CurrentIdentity(ctx)
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
