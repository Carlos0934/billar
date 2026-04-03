package httpauth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Carlos0934/billar/internal/app"
)

type callbackUseCaseStub struct {
	called bool
	cmd    app.HandleOAuthCallbackCommand
	result app.SessionDTO
	err    error
}

func (s *callbackUseCaseStub) HandleOAuthCallback(ctx context.Context, cmd app.HandleOAuthCallbackCommand) (app.SessionDTO, error) {
	_ = ctx
	s.called = true
	s.cmd = cmd
	return s.result, s.err
}

type stateStoreStub struct {
	validated   []string
	validateErr error
}

func (s *stateStoreStub) Generate(ctx context.Context) (string, error) {
	_ = ctx
	return "state-123", nil
}

func (s *stateStoreStub) Validate(ctx context.Context, state string) error {
	_ = ctx
	s.validated = append(s.validated, state)
	return s.validateErr
}

func TestCallbackHandler(t *testing.T) {
	tests := []struct {
		method     string
		name       string
		query      string
		stateErr   error
		useCaseErr error
		wantStatus int
		wantCalled bool
		wantState  string
		wantCode   string
	}{
		{
			method:     http.MethodGet,
			name:       "missing params returns bad request",
			query:      "",
			wantStatus: http.StatusBadRequest,
		},
		{
			method:     http.MethodGet,
			name:       "missing code only returns bad request",
			query:      "state=state-123",
			wantStatus: http.StatusBadRequest,
		},
		{
			method:     http.MethodGet,
			name:       "missing state only returns bad request",
			query:      "code=code-123",
			wantStatus: http.StatusBadRequest,
		},
		{
			method:     http.MethodGet,
			name:       "invalid state returns unauthorized",
			query:      "code=code-123&state=state-123",
			stateErr:   errors.New("state invalid"),
			wantStatus: http.StatusUnauthorized,
			wantCalled: false,
			wantState:  "state-123",
		},
		{
			method:     http.MethodGet,
			name:       "success redirects",
			query:      "code=code-123&state=state-123",
			wantStatus: http.StatusFound,
			wantCalled: true,
			wantState:  "state-123",
			wantCode:   "code-123",
		},
		{
			method:     http.MethodGet,
			name:       "policy rejection returns forbidden",
			query:      "code=code-123&state=state-123",
			useCaseErr: app.ErrUnauthorizedIdentity,
			wantStatus: http.StatusForbidden,
			wantCalled: true,
			wantState:  "state-123",
			wantCode:   "code-123",
		},
		{
			method:     http.MethodPost,
			name:       "non-get method returns method not allowed",
			query:      "code=code-123&state=state-123",
			wantStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			useCase := &callbackUseCaseStub{err: tc.useCaseErr}
			stateStore := &stateStoreStub{validateErr: tc.stateErr}
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, "http://example.test/auth/callback?"+tc.query, nil)

			CallbackHandler(useCase, stateStore).ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if tc.wantStatus == http.StatusMethodNotAllowed {
				if got := rec.Header().Get("Allow"); got != http.MethodGet {
					t.Fatalf("Allow = %q, want %q", got, http.MethodGet)
				}
				return
			}
			if tc.wantStatus == http.StatusFound {
				if got := rec.Header().Get("Location"); got != "/" {
					t.Fatalf("Location = %q, want %q", got, "/")
				}
			}
			if tc.wantState != "" {
				if len(stateStore.validated) != 1 {
					t.Fatalf("Validate() calls = %d, want 1", len(stateStore.validated))
				}
				if stateStore.validated[0] != tc.wantState {
					t.Fatalf("validated state = %q, want %q", stateStore.validated[0], tc.wantState)
				}
			}
			if useCase.called != tc.wantCalled {
				t.Fatalf("use case called = %v, want %v", useCase.called, tc.wantCalled)
			}
			if !useCase.called {
				return
			}
			if tc.wantCode != "" && useCase.cmd.Code != tc.wantCode {
				t.Fatalf("command code = %q, want %q", useCase.cmd.Code, tc.wantCode)
			}
			if tc.wantState != "" && useCase.cmd.State != tc.wantState {
				t.Fatalf("command state = %q, want %q", useCase.cmd.State, tc.wantState)
			}
		})
	}
}
