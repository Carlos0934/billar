package mcp

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/Carlos0934/billar/internal/app"
	"github.com/mark3labs/mcp-go/mcp"
)

type sessionServiceStub struct {
	statusCalled bool
	statusDTO    app.SessionStatusDTO
	statusErr    error
}

func (s *sessionServiceStub) Status(context.Context) (app.SessionStatusDTO, error) {
	s.statusCalled = true
	return s.statusDTO, s.statusErr
}

func TestSessionToolHandlers(t *testing.T) {
	t.Parallel()

	service := &sessionServiceStub{
		statusDTO: app.SessionStatusDTO{
			Status:        "active",
			Email:         "user@example.com",
			EmailVerified: true,
			Subject:       "subject-123",
			Issuer:        "https://issuer.example",
		},
	}
	guard := NewIngressGuard([]string{"127.0.0.1"})

	tests := []struct {
		name    string
		handler func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)
		request mcp.CallToolRequest
		want    string
		check   func(*testing.T)
	}{
		{
			name:    "status",
			handler: statusTool(service, guard, nil),
			request: mcp.CallToolRequest{Header: headerWithValues(map[string]string{
				"X-Forwarded-For": "127.0.0.1",
			})},
			want: "Status: active\nEmail: user@example.com\nEmail verified: true\nSubject: subject-123\nIssuer: https://issuer.example\n",
			check: func(t *testing.T) {
				if !service.statusCalled {
					t.Fatal("Status() was not called")
				}
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			result, err := tc.handler(context.Background(), tc.request)
			if err != nil {
				t.Fatalf("handler error = %v", err)
			}
			if result == nil || len(result.Content) != 1 {
				t.Fatalf("handler result = %+v, want single text content", result)
			}
			if got := mcp.GetTextFromContent(result.Content[0]); got != tc.want {
				t.Fatalf("handler text = %q, want %q", got, tc.want)
			}
			tc.check(t)
		})
	}
}

func TestSessionToolHandlersRejectIngress(t *testing.T) {
	t.Parallel()

	service := &sessionServiceStub{}
	guard := NewIngressGuard([]string{"127.0.0.1"})

	tests := []struct {
		name    string
		handler func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)
		request mcp.CallToolRequest
		check   func(*testing.T)
	}{
		{
			name:    "status rejects disallowed ingress ip",
			handler: statusTool(service, guard, nil),
			request: mcp.CallToolRequest{Header: headerWithValues(map[string]string{"X-Forwarded-For": "192.0.2.10"})},
			check: func(t *testing.T) {
				if service.statusCalled {
					t.Fatal("Status() was called for a rejected request")
				}
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			result, err := tc.handler(context.Background(), tc.request)
			if err != nil {
				t.Fatalf("handler error = %v", err)
			}
			if result == nil || !result.IsError {
				t.Fatalf("handler result = %+v, want error result", result)
			}
			tc.check(t)
		})
	}
}

func TestSessionToolHandlersPermitNoAllowlist(t *testing.T) {
	t.Parallel()

	service := &sessionServiceStub{
		statusDTO: app.SessionStatusDTO{Status: "unauthenticated"},
	}

	result, err := statusTool(service, NewIngressGuard(nil), nil)(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("handler error = %v", err)
	}
	if result == nil || result.IsError {
		t.Fatalf("handler result = %+v, want success result", result)
	}
	if got := mcp.GetTextFromContent(result.Content[0]); got != "Status: unauthenticated\n" {
		t.Fatalf("handler text = %q, want %q", got, "Status: unauthenticated\n")
	}
	if !service.statusCalled {
		t.Fatal("Status() was not called")
	}
}

func TestSessionToolHandlersPermitAllowedIngressIP(t *testing.T) {
	t.Parallel()

	service := &sessionServiceStub{
		statusDTO: app.SessionStatusDTO{Status: "active", Email: "person@example.com", EmailVerified: true},
	}
	guard := NewIngressGuard([]string{"127.0.0.1"})

	result, err := statusTool(service, guard, nil)(context.Background(), mcp.CallToolRequest{Header: headerWithValues(map[string]string{
		"X-Forwarded-For": "127.0.0.1",
	})})
	if err != nil {
		t.Fatalf("handler error = %v", err)
	}
	if result == nil || result.IsError {
		t.Fatalf("handler result = %+v, want success result", result)
	}
	if got := mcp.GetTextFromContent(result.Content[0]); got != "Status: active\nEmail: person@example.com\nEmail verified: true\n" {
		t.Fatalf("handler text = %q, want %q", got, "Status: active\nEmail: person@example.com\nEmail verified: true\n")
	}
	if !service.statusCalled {
		t.Fatal("Status() was not called")
	}
}

func TestSessionToolHandlersReturnToolErrors(t *testing.T) {
	t.Parallel()

	service := &sessionServiceStub{statusErr: errors.New("boom")}
	result, err := statusTool(service, NewIngressGuard(nil), nil)(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("handler error = %v", err)
	}
	if result == nil || !result.IsError {
		t.Fatalf("handler result = %+v, want error tool result", result)
	}
}

func TestSessionToolHandlersLogSafeFieldsOnDeniedIngress(t *testing.T) {
	t.Parallel()

	service := &sessionServiceStub{}
	guard := NewIngressGuard([]string{"127.0.0.1"})
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, nil))

	result, err := statusTool(service, guard, logger)(context.Background(), mcp.CallToolRequest{Header: headerWithValues(map[string]string{
		"X-Forwarded-For": "192.0.2.10",
	})})
	if err != nil {
		t.Fatalf("handler error = %v", err)
	}
	if result == nil || !result.IsError {
		t.Fatalf("handler result = %+v, want error result", result)
	}

	logged := logBuf.String()
	for _, want := range []string{"operation=session.status", "connector=mcp", "outcome=denied", "reason=ip_not_allowed"} {
		if !strings.Contains(logged, want) {
			t.Fatalf("log output = %q, want substring %q", logged, want)
		}
	}
	for _, unwanted := range []string{"192.0.2.10"} {
		if strings.Contains(logged, unwanted) {
			t.Fatalf("log output = %q, should not contain %q", logged, unwanted)
		}
	}
}

func TestSessionToolUsesContextAuthenticatedIdentity(t *testing.T) {
	t.Parallel()

	service := app.NewRequestSessionService(app.ContextIdentitySource{})
	result, err := statusTool(service, NewIngressGuard(nil), nil)(app.WithAuthenticatedIdentity(context.Background(), app.AuthenticatedIdentity{
		Email:         "person@example.com",
		EmailVerified: true,
		Subject:       "subject-123",
		Issuer:        "https://issuer.example",
	}), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("handler error = %v", err)
	}
	if result == nil || result.IsError {
		t.Fatalf("handler result = %+v, want success result", result)
	}
	if got := mcp.GetTextFromContent(result.Content[0]); got != "Status: active\nEmail: person@example.com\nEmail verified: true\nSubject: subject-123\nIssuer: https://issuer.example\n" {
		t.Fatalf("handler text = %q", got)
	}
}

func headerWithValues(values map[string]string) http.Header {
	headers := make(http.Header)
	for key, value := range values {
		headers.Set(key, value)
	}
	return headers
}
