package mcphttp

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Carlos0934/billar/internal/app"
)

type loginUseCaseStub struct {
	called bool
	result app.LoginIntentDTO
	err    error
}

func (s *loginUseCaseStub) StartLogin(ctx context.Context) (app.LoginIntentDTO, error) {
	_ = ctx
	s.called = true
	return s.result, s.err
}

func TestLoginHandler(t *testing.T) {
	tests := []struct {
		method       string
		name         string
		result       app.LoginIntentDTO
		err          error
		wantLocation string
		wantStatus   int
		wantCalled   bool
	}{
		{
			name:         "redirects to generated login URL",
			method:       http.MethodGet,
			result:       app.LoginIntentDTO{LoginURL: "https://accounts.google.com/o/oauth2/v2/auth?state=abc"},
			wantLocation: "https://accounts.google.com/o/oauth2/v2/auth?state=abc",
			wantStatus:   http.StatusFound,
			wantCalled:   true,
		},
		{
			name:       "returns internal server error when start login fails",
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
			useCase := &loginUseCaseStub{result: tc.result, err: tc.err}
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, "http://example.test/auth/login/start", nil)

			LoginHandler(useCase).ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if useCase.called != tc.wantCalled {
				t.Fatalf("StartLogin() called = %v, want %v", useCase.called, tc.wantCalled)
			}
			if tc.wantLocation != "" && rec.Header().Get("Location") != tc.wantLocation {
				t.Fatalf("Location = %q, want %q", rec.Header().Get("Location"), tc.wantLocation)
			}
		})
	}
}
