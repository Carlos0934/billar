package auth

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Carlos0934/billar/internal/app"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newTestHTTPClient(fn roundTripFunc) *http.Client {
	return &http.Client{Transport: fn}
}

func TestGoogleAccessTokenAuthenticatorAuthenticateAccessToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		auth    GoogleAccessTokenAuthenticator
		token   string
		want    app.AuthenticatedIdentity
		wantErr error
		wantMsg string
	}{
		{
			name:  "accepts valid Google access token",
			token: "good-token",
			auth: GoogleAccessTokenAuthenticator{
				httpClient: newTestHTTPClient(func(req *http.Request) (*http.Response, error) {
					switch {
					case strings.Contains(req.URL.Path, "/tokeninfo"):
						if got := req.URL.Query().Get("access_token"); got != "good-token" {
							t.Fatalf("tokeninfo access_token = %q, want %q", got, "good-token")
						}
						return httpResponse(http.StatusOK, `{"aud":"client-123","iss":"accounts.google.com","expires_in":"3600"}`), nil
					case strings.Contains(req.URL.Path, "/userinfo"):
						if got := req.Header.Get("Authorization"); got != "Bearer good-token" {
							t.Fatalf("userinfo Authorization = %q, want %q", got, "Bearer good-token")
						}
						return httpResponse(http.StatusOK, `{"sub":"subject-123","email":"user@example.com","email_verified":true}`), nil
					default:
						t.Fatalf("unexpected request URL %q", req.URL.String())
						return nil, nil
					}
				}),
				tokenInfoURL:     googleTokenInfoURL,
				userInfoURL:      googleUserInfoURL,
				expectedAudience: "client-123",
				allowedIssuers:   allowedIssuerSet("https://accounts.google.com"),
			},
			want: app.AuthenticatedIdentity{Email: "user@example.com", EmailVerified: true, Subject: "subject-123", Issuer: "https://accounts.google.com"},
		},
		{
			name:  "accepts empty issuer when audience and userinfo are valid",
			token: "good-token",
			auth: GoogleAccessTokenAuthenticator{
				httpClient: newTestHTTPClient(func(req *http.Request) (*http.Response, error) {
					switch {
					case strings.Contains(req.URL.Path, "/tokeninfo"):
						return httpResponse(http.StatusOK, `{"aud":"client-123","expires_in":"3600"}`), nil
					case strings.Contains(req.URL.Path, "/userinfo"):
						return httpResponse(http.StatusOK, `{"sub":"subject-123","email":"user@example.com","email_verified":true}`), nil
					default:
						t.Fatalf("unexpected request URL %q", req.URL.String())
						return nil, nil
					}
				}),
				tokenInfoURL:     googleTokenInfoURL,
				userInfoURL:      googleUserInfoURL,
				expectedAudience: "client-123",
				allowedIssuers:   allowedIssuerSet("https://accounts.google.com"),
			},
			want: app.AuthenticatedIdentity{Email: "user@example.com", EmailVerified: true, Subject: "subject-123", Issuer: ""},
		},
		{
			name:  "rejects mismatched audience",
			token: "good-token",
			auth: GoogleAccessTokenAuthenticator{
				httpClient: newTestHTTPClient(func(req *http.Request) (*http.Response, error) {
					if strings.Contains(req.URL.Path, "/tokeninfo") {
						return httpResponse(http.StatusOK, `{"aud":"other-client","iss":"https://accounts.google.com","expires_in":"3600"}`), nil
					}
					t.Fatalf("userinfo should not be called")
					return nil, nil
				}),
				tokenInfoURL:     googleTokenInfoURL,
				userInfoURL:      googleUserInfoURL,
				expectedAudience: "client-123",
				allowedIssuers:   allowedIssuerSet("https://accounts.google.com"),
			},
			wantErr: app.ErrAccessTokenRejected,
		},
		{
			name:  "rejects expired token",
			token: "good-token",
			auth: GoogleAccessTokenAuthenticator{
				httpClient: newTestHTTPClient(func(req *http.Request) (*http.Response, error) {
					if strings.Contains(req.URL.Path, "/tokeninfo") {
						return httpResponse(http.StatusOK, `{"aud":"client-123","iss":"https://accounts.google.com","expires_in":"0"}`), nil
					}
					t.Fatalf("userinfo should not be called")
					return nil, nil
				}),
				tokenInfoURL:     googleTokenInfoURL,
				userInfoURL:      googleUserInfoURL,
				expectedAudience: "client-123",
				allowedIssuers:   allowedIssuerSet("https://accounts.google.com"),
			},
			wantErr: app.ErrAccessTokenRejected,
		},
		{
			name:  "rejects invalid token from tokeninfo",
			token: "bad-token",
			auth: GoogleAccessTokenAuthenticator{
				httpClient: newTestHTTPClient(func(req *http.Request) (*http.Response, error) {
					if strings.Contains(req.URL.Path, "/tokeninfo") {
						return httpResponse(http.StatusBadRequest, `{"error_description":"Invalid Value"}`), nil
					}
					t.Fatalf("userinfo should not be called")
					return nil, nil
				}),
				tokenInfoURL:     googleTokenInfoURL,
				userInfoURL:      googleUserInfoURL,
				expectedAudience: "client-123",
				allowedIssuers:   allowedIssuerSet("https://accounts.google.com"),
			},
			wantErr: app.ErrAccessTokenRejected,
		},
		{
			name:  "rejects missing subject from userinfo",
			token: "good-token",
			auth: GoogleAccessTokenAuthenticator{
				httpClient: newTestHTTPClient(func(req *http.Request) (*http.Response, error) {
					if strings.Contains(req.URL.Path, "/tokeninfo") {
						return httpResponse(http.StatusOK, `{"aud":"client-123","iss":"https://accounts.google.com","expires_in":"3600"}`), nil
					}
					return httpResponse(http.StatusOK, `{"email":"user@example.com","email_verified":true}`), nil
				}),
				tokenInfoURL:     googleTokenInfoURL,
				userInfoURL:      googleUserInfoURL,
				expectedAudience: "client-123",
				allowedIssuers:   allowedIssuerSet("https://accounts.google.com"),
			},
			wantErr: app.ErrAccessTokenRejected,
		},
		{
			name:  "returns network failures with context",
			token: "good-token",
			auth: GoogleAccessTokenAuthenticator{
				httpClient: newTestHTTPClient(func(*http.Request) (*http.Response, error) {
					return nil, errors.New("boom")
				}),
				tokenInfoURL:     googleTokenInfoURL,
				userInfoURL:      googleUserInfoURL,
				expectedAudience: "client-123",
				allowedIssuers:   allowedIssuerSet("https://accounts.google.com"),
			},
			wantMsg: "boom",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := tc.auth.AuthenticateAccessToken(context.Background(), tc.token)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("AuthenticateAccessToken() error = %v, want %v", err, tc.wantErr)
				}
				return
			}
			if tc.wantMsg != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantMsg) {
					t.Fatalf("AuthenticateAccessToken() error = %v, want substring %q", err, tc.wantMsg)
				}
				return
			}
			if err != nil {
				t.Fatalf("AuthenticateAccessToken() error = %v", err)
			}
			if got != tc.want {
				t.Fatalf("AuthenticateAccessToken() = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestNewGoogleAccessTokenAuthenticator(t *testing.T) {
	t.Parallel()

	if _, err := NewGoogleAccessTokenAuthenticator("", "client-123"); err == nil {
		t.Fatal("NewGoogleAccessTokenAuthenticator() error = nil, want issuer validation")
	}
	if _, err := NewGoogleAccessTokenAuthenticator("https://accounts.google.com", ""); err == nil {
		t.Fatal("NewGoogleAccessTokenAuthenticator() error = nil, want client id validation")
	}
}

func httpResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
