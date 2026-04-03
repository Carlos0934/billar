package mcphttp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Carlos0934/billar/internal/app"
)

type statusUseCaseStub struct {
	called bool
	result app.SessionStatusDTO
	err    error
}

func (s *statusUseCaseStub) Status(ctx context.Context) (app.SessionStatusDTO, error) {
	_ = ctx
	s.called = true
	return s.result, s.err
}

type logoutUseCaseStub struct {
	called bool
	result app.LogoutDTO
	err    error
}

func (s *logoutUseCaseStub) Logout(ctx context.Context) (app.LogoutDTO, error) {
	_ = ctx
	s.called = true
	return s.result, s.err
}

func TestSessionStatusHandler(t *testing.T) {
	tests := []struct {
		method     string
		name       string
		result     app.SessionStatusDTO
		err        error
		wantStatus int
		wantCalled bool
	}{
		{
			name:       "returns session status json",
			method:     http.MethodGet,
			result:     app.SessionStatusDTO{Status: "active", Email: "user@example.com", EmailVerified: true},
			wantStatus: http.StatusOK,
			wantCalled: true,
		},
		{
			name:       "returns internal server error when status fails",
			method:     http.MethodGet,
			err:        errors.New("boom"),
			wantStatus: http.StatusInternalServerError,
			wantCalled: true,
		},
		{
			name:       "rejects non-get requests",
			method:     http.MethodPost,
			wantStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			useCase := &statusUseCaseStub{result: tc.result, err: tc.err}
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, "http://example.test/auth/session", nil)

			SessionStatusHandler(useCase).ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if useCase.called != tc.wantCalled {
				t.Fatalf("Status() called = %v, want %v", useCase.called, tc.wantCalled)
			}
			if tc.wantStatus == http.StatusOK {
				var got app.SessionStatusDTO
				if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
					t.Fatalf("json.Unmarshal() error = %v", err)
				}
				if got != tc.result {
					t.Fatalf("body = %+v, want %+v", got, tc.result)
				}
			}
		})
	}
}

func TestLogoutHandler(t *testing.T) {
	tests := []struct {
		method     string
		name       string
		result     app.LogoutDTO
		err        error
		wantStatus int
		wantCalled bool
	}{
		{
			name:       "returns logout json",
			method:     http.MethodPost,
			result:     app.LogoutDTO{Message: "Logged out"},
			wantStatus: http.StatusOK,
			wantCalled: true,
		},
		{
			name:       "returns internal server error when logout fails",
			method:     http.MethodPost,
			err:        errors.New("boom"),
			wantStatus: http.StatusInternalServerError,
			wantCalled: true,
		},
		{
			name:       "rejects non-post requests",
			method:     http.MethodGet,
			wantStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			useCase := &logoutUseCaseStub{result: tc.result, err: tc.err}
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, "http://example.test/auth/logout", nil)

			LogoutHandler(useCase).ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if useCase.called != tc.wantCalled {
				t.Fatalf("Logout() called = %v, want %v", useCase.called, tc.wantCalled)
			}
		})
	}
}
