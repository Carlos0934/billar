package mcp

import (
	"context"
	"errors"
	"net/http"
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

	tests := []struct {
		name    string
		handler func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)
		request mcp.CallToolRequest
		want    string
		check   func(*testing.T)
	}{
		{
			name:    "status",
			handler: statusTool(service, nil),
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

func TestSessionToolHandlersPermitNoAllowlist(t *testing.T) {
	t.Parallel()

	service := &sessionServiceStub{
		statusDTO: app.SessionStatusDTO{Status: "unauthenticated"},
	}

	result, err := statusTool(service, nil)(context.Background(), mcp.CallToolRequest{})
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

	result, err := statusTool(service, nil)(context.Background(), mcp.CallToolRequest{Header: headerWithValues(map[string]string{
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
	result, err := statusTool(service, nil)(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("handler error = %v", err)
	}
	if result == nil || !result.IsError {
		t.Fatalf("handler result = %+v, want error tool result", result)
	}
}

func TestSessionToolUsesContextAuthenticatedIdentity(t *testing.T) {
	t.Parallel()

	service := app.NewRequestSessionService(app.ContextIdentitySource{})
	result, err := statusTool(service, nil)(app.WithAuthenticatedIdentity(context.Background(), app.AuthenticatedIdentity{
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
