package mcp

import (
	"testing"

	"github.com/Carlos0934/billar/internal/app"
)

func TestLoginIntentText(t *testing.T) {
	t.Parallel()

	got := loginIntentText(app.LoginIntentDTO{LoginURL: "https://login.example"})
	want := "Login URL: https://login.example\n"
	if got != want {
		t.Fatalf("loginIntentText() = %q, want %q", got, want)
	}
}

func TestSessionStatusText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   app.SessionStatusDTO
		want string
	}{
		{
			name: "unauthenticated",
			in:   app.SessionStatusDTO{Status: "unauthenticated"},
			want: "Status: unauthenticated\n",
		},
		{
			name: "active identity",
			in: app.SessionStatusDTO{
				Status:        "active",
				Email:         "user@example.com",
				EmailVerified: true,
				Subject:       "subject-123",
				Issuer:        "https://issuer.example",
			},
			want: "Status: active\nEmail: user@example.com\nEmail verified: true\nSubject: subject-123\nIssuer: https://issuer.example\n",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := sessionStatusText(tc.in); got != tc.want {
				t.Fatalf("sessionStatusText() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestLogoutText(t *testing.T) {
	t.Parallel()

	got := logoutText(app.LogoutDTO{Message: "Logged out"})
	want := "Logged out\n"
	if got != want {
		t.Fatalf("logoutText() = %q, want %q", got, want)
	}
}
