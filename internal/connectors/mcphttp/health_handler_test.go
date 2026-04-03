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

type healthUseCaseStub struct {
	called bool
	result app.HealthDTO
	err    error
}

func (s *healthUseCaseStub) Status(ctx context.Context) (app.HealthDTO, error) {
	_ = ctx
	s.called = true
	return s.result, s.err
}

func TestHealthHandler(t *testing.T) {
	tests := []struct {
		method     string
		name       string
		result     app.HealthDTO
		err        error
		wantStatus int
		wantCalled bool
	}{
		{
			name:       "returns health json",
			method:     http.MethodGet,
			result:     app.HealthDTO{Name: "billar", Status: "ok"},
			wantStatus: http.StatusOK,
			wantCalled: true,
		},
		{
			name:       "returns internal server error when health fails",
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
			useCase := &healthUseCaseStub{result: tc.result, err: tc.err}
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, "http://example.test/healthz", nil)

			HealthHandler(useCase).ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if useCase.called != tc.wantCalled {
				t.Fatalf("Status() called = %v, want %v", useCase.called, tc.wantCalled)
			}
			if tc.wantStatus == http.StatusOK {
				var got app.HealthDTO
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
