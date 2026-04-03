package mcp

import (
	"errors"
	"net/http"
	"testing"

	"github.com/Carlos0934/billar/internal/infra/config"
)

func TestIngressGuardCheckIP(t *testing.T) {
	t.Parallel()

	guard := NewIngressGuard([]string{"127.0.0.1", "10.0.0.1"})

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

		if err := NewIngressGuard(nil).CheckIP("203.0.113.9"); err != nil {
			t.Fatalf("CheckIP() error = %v, want nil", err)
		}
	})
}

func TestIngressGuardAuthorizeUsesOnlyIPAllowlist(t *testing.T) {
	t.Parallel()

	guard := NewIngressGuard([]string{"127.0.0.1"})

	if err := guard.authorize(http.Header{
		"X-Forwarded-For":       []string{"127.0.0.1"},
		"X-Authenticated-Email": []string{"blocked@example.com"},
	}); err != nil {
		t.Fatalf("authorize() error = %v", err)
	}

	if err := guard.authorize(http.Header{
		"X-Forwarded-For": []string{"192.0.2.10"},
	}); err == nil || !errors.Is(err, ErrIPNotAllowed) {
		t.Fatalf("authorize() error = %v, want ErrIPNotAllowed", err)
	}
}

func TestNewIngressGuardFromConfigBuildsIPOnlyPolicy(t *testing.T) {
	t.Parallel()

	guard := NewIngressGuardFromConfig(config.AccessPolicy{
		AllowedIPs:     []string{"127.0.0.1"},
		AllowedDomains: []string{"example.com"},
	})

	if err := guard.CheckIP("127.0.0.1"); err != nil {
		t.Fatalf("CheckIP() error = %v", err)
	}
	if err := guard.authorize(http.Header{"X-Forwarded-For": []string{"127.0.0.1"}, "X-Authenticated-Email": []string{"person@example.com"}}); err != nil {
		t.Fatalf("authorize() error = %v", err)
	}
}
