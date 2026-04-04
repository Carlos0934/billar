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

func TestCustomerListText(t *testing.T) {
	t.Parallel()

	got := customerListText(app.ListResult[app.CustomerDTO]{
		Items: []app.CustomerDTO{{
			ID:              "cus_123",
			Type:            "company",
			LegalName:       "Acme SRL",
			TradeName:       "Acme",
			Email:           "billing@acme.example",
			Status:          "active",
			DefaultCurrency: "USD",
			CreatedAt:       "2026-04-03T10:00:00Z",
			UpdatedAt:       "2026-04-03T10:05:00Z",
		}},
		Total:    1,
		Page:     2,
		PageSize: 1,
	})
	want := "Billar Customers\n───────────────\nPage: 2\nPage size: 1\nTotal: 1\n\n1. Acme SRL\n   Trade name: Acme\n   Type: company\n   Status: active\n   Email: billing@acme.example\n   Default currency: USD\n   Created at: 2026-04-03T10:00:00Z\n   Updated at: 2026-04-03T10:05:00Z\n"
	if got != want {
		t.Fatalf("customerListText() = %q, want %q", got, want)
	}
}
