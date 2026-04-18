package mcp

import (
	"testing"

	"github.com/Carlos0934/billar/internal/app"
)

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
		{
			name: "synthetic MCP API-key identity hides internal fields",
			in: app.SessionStatusDTO{
				Status:        "active",
				Email:         syntheticMCPEmail,
				EmailVerified: true,
				Subject:       syntheticMCPSubject,
				Issuer:        syntheticMCPIssuer,
			},
			want: "Status: active\n",
		},
		{
			name: "real identity that coincidentally matches email only is not suppressed",
			in: app.SessionStatusDTO{
				Status:        "active",
				Email:         syntheticMCPEmail,
				EmailVerified: true,
				Subject:       "real-subject",
				Issuer:        "https://real.issuer",
			},
			want: "Status: active\nEmail: mcp@local\nEmail verified: true\nSubject: real-subject\nIssuer: https://real.issuer\n",
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
