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
	startLoginCalled bool
	statusCalled     bool
	logoutCalled     bool
	startLoginDTO    app.LoginIntentDTO
	statusDTO        app.SessionStatusDTO
	logoutDTO        app.LogoutDTO
	startLoginErr    error
	statusErr        error
	logoutErr        error
}

func (s *sessionServiceStub) StartLogin(context.Context) (app.LoginIntentDTO, error) {
	s.startLoginCalled = true
	return s.startLoginDTO, s.startLoginErr
}

func (s *sessionServiceStub) Status(context.Context) (app.SessionStatusDTO, error) {
	s.statusCalled = true
	return s.statusDTO, s.statusErr
}

func (s *sessionServiceStub) Logout(context.Context) (app.LogoutDTO, error) {
	s.logoutCalled = true
	return s.logoutDTO, s.logoutErr
}

func TestSessionToolHandlers(t *testing.T) {
	t.Parallel()

	service := &sessionServiceStub{
		startLoginDTO: app.LoginIntentDTO{LoginURL: "https://login.example"},
		statusDTO: app.SessionStatusDTO{
			Status:        "active",
			Email:         "user@example.com",
			EmailVerified: true,
			Subject:       "subject-123",
			Issuer:        "https://issuer.example",
		},
		logoutDTO: app.LogoutDTO{Message: "Logged out"},
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
			name:    "start login",
			handler: startLoginTool(service),
			request: mcp.CallToolRequest{Header: headerWithValues(map[string]string{
				"X-Forwarded-For":       "192.0.2.10",
				"X-Authenticated-Email": "blocked@other.com",
			})},
			want: "Login URL: https://login.example\n",
			check: func(t *testing.T) {
				if !service.startLoginCalled {
					t.Fatal("StartLogin() was not called")
				}
			},
		},
		{
			name:    "status",
			handler: statusTool(service, guard),
			request: mcp.CallToolRequest{Header: headerWithValues(map[string]string{
				"X-Forwarded-For":       "127.0.0.1",
				"X-Authenticated-Email": "user@example.com",
			})},
			want: "Status: active\nEmail: user@example.com\nEmail verified: true\nSubject: subject-123\nIssuer: https://issuer.example\n",
			check: func(t *testing.T) {
				if !service.statusCalled {
					t.Fatal("Status() was not called")
				}
			},
		},
		{
			name:    "logout",
			handler: logoutTool(service, guard),
			request: mcp.CallToolRequest{Header: headerWithValues(map[string]string{
				"X-Real-IP":             "127.0.0.1",
				"X-Authenticated-Email": "user@example.com",
			})},
			want: "Logged out\n",
			check: func(t *testing.T) {
				if !service.logoutCalled {
					t.Fatal("Logout() was not called")
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
			handler: statusTool(service, guard),
			request: mcp.CallToolRequest{Header: headerWithValues(map[string]string{"X-Forwarded-For": "192.0.2.10", "X-Authenticated-Email": "blocked@other.com"})},
			check: func(t *testing.T) {
				if service.statusCalled {
					t.Fatal("Status() was called for a rejected request")
				}
			},
		},
		{
			name:    "logout rejects disallowed ingress ip",
			handler: logoutTool(service, guard),
			request: mcp.CallToolRequest{Header: headerWithValues(map[string]string{"X-Forwarded-For": "192.0.2.10"})},
			check: func(t *testing.T) {
				if service.logoutCalled {
					t.Fatal("Logout() was called for a rejected request")
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

	result, err := statusTool(service, NewIngressGuard(nil))(context.Background(), mcp.CallToolRequest{})
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
		logoutDTO: app.LogoutDTO{Message: "Logged out"},
	}
	guard := NewIngressGuard([]string{"127.0.0.1"})

	result, err := logoutTool(service, guard)(context.Background(), mcp.CallToolRequest{Header: headerWithValues(map[string]string{
		"X-Forwarded-For":       "127.0.0.1",
		"X-Authenticated-Email": "person@example.com",
	})})
	if err != nil {
		t.Fatalf("handler error = %v", err)
	}
	if result == nil || result.IsError {
		t.Fatalf("handler result = %+v, want success result", result)
	}
	if got := mcp.GetTextFromContent(result.Content[0]); got != "Logged out\n" {
		t.Fatalf("handler text = %q, want %q", got, "Logged out\n")
	}
	if !service.logoutCalled {
		t.Fatal("Logout() was not called")
	}
}

func TestSessionToolHandlersReturnToolErrors(t *testing.T) {
	t.Parallel()

	service := &sessionServiceStub{startLoginErr: errors.New("boom")}
	result, err := startLoginTool(service)(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("handler error = %v", err)
	}
	if result == nil || !result.IsError {
		t.Fatalf("handler result = %+v, want error tool result", result)
	}
}

func headerWithValues(values map[string]string) http.Header {
	headers := make(http.Header)
	for key, value := range values {
		headers.Set(key, value)
	}
	return headers
}
