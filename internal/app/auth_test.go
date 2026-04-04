package app

import (
	"context"
	"errors"
	"testing"
)

func TestTokenAccessTokenAuthenticatorAuthenticateAccessToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		token         string
		authenticator TokenAccessTokenAuthenticator
		want          AuthenticatedIdentity
		wantErr       error
	}{
		{
			name:  "returns the configured verified identity",
			token: "token-123",
			authenticator: TokenAccessTokenAuthenticator{
				ExpectedToken: "token-123",
				Identity: AuthenticatedIdentity{
					Email:         "user@example.com",
					EmailVerified: true,
					Subject:       "subject-123",
					Issuer:        "https://issuer.example",
				},
			},
			want: AuthenticatedIdentity{
				Email:         "user@example.com",
				EmailVerified: true,
				Subject:       "subject-123",
				Issuer:        "https://issuer.example",
			},
		},
		{
			name:  "rejects a mismatched raw token",
			token: "other-token",
			authenticator: TokenAccessTokenAuthenticator{
				ExpectedToken: "token-123",
				Identity:      AuthenticatedIdentity{Email: "user@example.com"},
			},
			wantErr: ErrAccessTokenRejected,
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := tc.authenticator.AuthenticateAccessToken(context.Background(), tc.token)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("AuthenticateAccessToken() error = %v, want %v", err, tc.wantErr)
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
