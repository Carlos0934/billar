package mcp

import (
	"errors"
	"testing"

	"github.com/Carlos0934/billar/internal/infra/config"
)

func TestIngressGuardCheckIP(t *testing.T) {
	t.Parallel()

	guard := NewIngressGuard(config.AccessPolicy{AllowedIPs: []string{"127.0.0.1", "10.0.0.1"}})

	tests := []struct {
		name    string
		ip      string
		wantErr bool
	}{
		{name: "allowed ip", ip: "127.0.0.1"},
		{name: "allowed ip trimmed", ip: " 10.0.0.1 "},
		{name: "disallowed ip", ip: "192.168.1.5", wantErr: true},
		{name: "invalid ip", ip: "not-an-ip", wantErr: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := guard.CheckIP(tc.ip)
			if tc.wantErr {
				if err == nil {
					t.Fatal("CheckIP() error = nil, want non-nil")
				}
				if !errors.Is(err, ErrIPNotAllowed) {
					t.Fatalf("CheckIP() error = %v, want ErrIPNotAllowed", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("CheckIP() error = %v", err)
			}
		})
	}

	t.Run("no allowlist permits all ips", func(t *testing.T) {
		t.Parallel()

		if err := NewIngressGuard(config.AccessPolicy{}).CheckIP("203.0.113.9"); err != nil {
			t.Fatalf("CheckIP() error = %v, want nil", err)
		}
	})
}

func TestIngressGuardCheckIdentity(t *testing.T) {
	t.Parallel()

	guard := NewIngressGuard(config.AccessPolicy{
		AllowedEmails:  []string{"admin@example.com"},
		AllowedDomains: []string{"company.com"},
	})

	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{name: "exact email match", email: "ADMIN@example.com"},
		{name: "domain match", email: "employee@Company.Com"},
		{name: "denied identity", email: "hacker@other.com", wantErr: true},
		{name: "invalid email", email: "not-an-email", wantErr: true},
		{name: "subdomain does not match", email: "employee@sub.company.com", wantErr: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := guard.CheckIdentity(tc.email)
			if tc.wantErr {
				if err == nil {
					t.Fatal("CheckIdentity() error = nil, want non-nil")
				}
				if !errors.Is(err, ErrIdentityNotAllowed) {
					t.Fatalf("CheckIdentity() error = %v, want ErrIdentityNotAllowed", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("CheckIdentity() error = %v", err)
			}
		})
	}
}
