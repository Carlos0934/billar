package app

import "testing"

func TestEnvAccessPolicyIsAllowed(t *testing.T) {
	tests := []struct {
		name   string
		policy IdentityPolicy
		email  string
		want   bool
	}{
		{
			name:   "exact email match is case-insensitive",
			policy: IdentityPolicy{AllowedEmails: []string{"admin@example.com"}},
			email:  "ADMIN@example.com",
			want:   true,
		},
		{
			name:   "domain match is case-insensitive",
			policy: IdentityPolicy{AllowedDomains: []string{"company.com"}},
			email:  "employee@Company.Com",
			want:   true,
		},
		{
			name:   "subdomain rejection requires exact domain match",
			policy: IdentityPolicy{AllowedDomains: []string{"company.com"}},
			email:  "employee@sub.company.com",
			want:   false,
		},
		{
			name:   "empty policy denies all identities",
			policy: IdentityPolicy{},
			email:  "user@example.com",
			want:   false,
		},
		{
			name:   "configured allowlists deny identities that do not match",
			policy: IdentityPolicy{AllowedEmails: []string{"admin@example.com"}, AllowedDomains: []string{"company.com"}},
			email:  "unknown@other.com",
			want:   false,
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			if got := tc.policy.IsAllowed(tc.email); got != tc.want {
				t.Fatalf("IsAllowed(%q) = %v, want %v", tc.email, got, tc.want)
			}
		})
	}
}
