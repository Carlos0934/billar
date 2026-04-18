package mcphttp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Carlos0934/billar/internal/app"
)

func TestMetadataHandler(t *testing.T) {
	tests := []struct {
		method       string
		name         string
		challenge    app.OAuthChallengeDTO
		wantResource string
		wantServers  []string
		wantStatus   int
	}{
		{
			method: http.MethodGet,
			name:   "writes challenge metadata with one authorization server",
			challenge: app.OAuthChallengeDTO{
				ResourceURI:          "https://resource.example",
				AuthorizationServers: []string{"https://issuer.example"},
			},
			wantResource: "https://resource.example",
			wantServers:  []string{"https://issuer.example"},
			wantStatus:   http.StatusOK,
		},
		{
			method: http.MethodGet,
			name:   "preserves multiple authorization servers",
			challenge: app.OAuthChallengeDTO{
				ResourceURI:          "https://resource.example",
				AuthorizationServers: []string{"https://issuer-a.example", "https://issuer-b.example"},
			},
			wantResource: "https://resource.example",
			wantServers:  []string{"https://issuer-a.example", "https://issuer-b.example"},
			wantStatus:   http.StatusOK,
		},
		{
			method:     http.MethodPost,
			name:       "non-get method returns method not allowed",
			wantStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, "http://example.test/.well-known/oauth-protected-resource", nil)

			MetadataHandler(tc.challenge).ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if tc.wantStatus == http.StatusMethodNotAllowed {
				if got := rec.Header().Get("Allow"); got != http.MethodGet {
					t.Fatalf("Allow = %q, want %q", got, http.MethodGet)
				}
				return
			}

			if got := rec.Header().Get("Content-Type"); got != "application/json" {
				t.Fatalf("Content-Type = %q, want application/json", got)
			}

			var got struct {
				ResourceURI            string   `json:"resource"`
				AuthorizationServers   []string `json:"authorization_servers"`
				BearerMethodsSupported []string `json:"bearer_methods_supported"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("json.Unmarshal() error = %v", err)
			}

			if got.ResourceURI != tc.wantResource {
				t.Fatalf("resource = %q, want %q", got.ResourceURI, tc.wantResource)
			}
			if len(got.AuthorizationServers) != len(tc.wantServers) {
				t.Fatalf("authorization_servers length = %d, want %d", len(got.AuthorizationServers), len(tc.wantServers))
			}
			for i := range tc.wantServers {
				if got.AuthorizationServers[i] != tc.wantServers[i] {
					t.Fatalf("authorization_servers[%d] = %q, want %q", i, got.AuthorizationServers[i], tc.wantServers[i])
				}
			}
			if len(got.BearerMethodsSupported) == 0 {
				t.Fatal("bearer_methods_supported = empty, want at least one method")
			}
			foundBearer := false
			for _, m := range got.BearerMethodsSupported {
				if m == "bearer" {
					foundBearer = true
				}
			}
			if !foundBearer {
				t.Fatalf("bearer_methods_supported = %v, want to contain \"bearer\"", got.BearerMethodsSupported)
			}
		})
	}
}
