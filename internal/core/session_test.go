package core

import "testing"

func TestSessionZeroValue(t *testing.T) {
	t.Parallel()

	var got Session

	if got.ID != "" {
		t.Fatalf("Session{}.ID = %q, want empty", got.ID)
	}
	if got.Status != SessionStatusUnauthenticated {
		t.Fatalf("Session{}.Status = %v, want unauthenticated", got.Status)
	}
	if got.Identity != (Identity{}) {
		t.Fatalf("Session{}.Identity = %+v, want zero value", got.Identity)
	}
}

func TestSessionStatusString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		got  SessionStatus
		want string
	}{
		{name: "unauthenticated", got: SessionStatusUnauthenticated, want: "unauthenticated"},
		{name: "active", got: SessionStatusActive, want: "active"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := tc.got.String(); got != tc.want {
				t.Fatalf("SessionStatus.String() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestIdentityFields(t *testing.T) {
	t.Parallel()

	got := Identity{
		Email:         "user@example.com",
		EmailVerified: true,
		Subject:       "subject-123",
		Issuer:        "https://issuer.example",
	}

	if got.Email != "user@example.com" {
		t.Fatalf("Identity.Email = %q, want %q", got.Email, "user@example.com")
	}
	if !got.EmailVerified {
		t.Fatal("Identity.EmailVerified = false, want true")
	}
	if got.Subject != "subject-123" {
		t.Fatalf("Identity.Subject = %q, want %q", got.Subject, "subject-123")
	}
	if got.Issuer != "https://issuer.example" {
		t.Fatalf("Identity.Issuer = %q, want %q", got.Issuer, "https://issuer.example")
	}
}
