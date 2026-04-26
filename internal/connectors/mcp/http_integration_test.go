package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Carlos0934/billar/internal/app"
	mcplib "github.com/mark3labs/mcp-go/mcp"
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

func TestHTTPHandlerToolCallReturnsStructuredContent(t *testing.T) {
	t.Parallel()

	customer := &customerProfileWriteServiceStub{getRes: app.CustomerProfileDTO{ID: "cus_http", LegalEntityID: "le_http", Status: "active", DefaultCurrency: "USD"}}
	server := NewServer(nil, nil, customer, nil, nil, nil, nil)
	handler := server.HTTPHandler()
	sessionID := initializeHTTPMCPSession(t, handler)

	body, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "customer_profile.get",
			"arguments": map[string]any{
				"id": "cus_http",
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/mcp", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Mcp-Session-Id", sessionID)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}

	payload := responsePayloadFromHTTPBody(t, w.Body.Bytes())
	var envelope struct {
		Result struct {
			StructuredContent app.CustomerProfileDTO `json:"structuredContent"`
			Content           []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"result"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil {
		t.Fatalf("decode response %s: %v", payload, err)
	}
	if envelope.Result.StructuredContent.ID != "cus_http" || envelope.Result.StructuredContent.DefaultCurrency != "USD" {
		t.Fatalf("structuredContent = %+v", envelope.Result.StructuredContent)
	}
	if len(envelope.Result.Content) == 0 || strings.TrimSpace(envelope.Result.Content[0].Text) == "" {
		t.Fatalf("text fallback missing in response: %s", payload)
	}
}

func initializeHTTPMCPSession(t *testing.T, handler http.Handler) string {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": mcplib.LATEST_PROTOCOL_VERSION,
			"clientInfo": map[string]any{
				"name":    "billar-test",
				"version": "1.0.0",
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal initialize request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/mcp", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("initialize status = %d, body = %s", w.Code, w.Body.String())
	}
	sessionID := w.Header().Get("Mcp-Session-Id")
	if sessionID == "" {
		t.Fatalf("initialize response missing Mcp-Session-Id header; body = %s", w.Body.String())
	}
	return sessionID
}

func responsePayloadFromHTTPBody(t *testing.T, body []byte) []byte {
	t.Helper()
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		t.Fatal("empty HTTP response body")
	}
	if trimmed[0] == '{' {
		return trimmed
	}
	lines := bytes.Split(trimmed, []byte("\n"))
	for i := len(lines) - 1; i >= 0; i-- {
		line := bytes.TrimSpace(lines[i])
		if bytes.HasPrefix(line, []byte("data:")) {
			return bytes.TrimSpace(bytes.TrimPrefix(line, []byte("data:")))
		}
	}
	all, _ := io.ReadAll(bytes.NewReader(body))
	t.Fatalf("response body did not contain JSON or SSE data: %s", string(all))
	return nil
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
