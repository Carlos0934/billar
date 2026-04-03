package mcp

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/Carlos0934/billar/internal/infra/config"
)

var (
	ErrIPNotAllowed       = errors.New("ip not allowed")
	ErrIdentityNotAllowed = errors.New("identity not allowed")
)

type IngressGuard struct {
	allowedIPs     map[string]struct{}
	allowedEmails  map[string]struct{}
	allowedDomains map[string]struct{}
}

func DefaultAccessPolicy() config.AccessPolicy {
	return config.AccessPolicy{}
}

func NewIngressGuard(policy config.AccessPolicy) IngressGuard {
	guard := IngressGuard{
		allowedIPs:     make(map[string]struct{}, len(policy.AllowedIPs)),
		allowedEmails:  make(map[string]struct{}, len(policy.AllowedEmails)),
		allowedDomains: make(map[string]struct{}, len(policy.AllowedDomains)),
	}

	for _, value := range policy.AllowedIPs {
		if normalized := normalizeIP(value); normalized != "" {
			guard.allowedIPs[normalized] = struct{}{}
		}
	}
	for _, value := range policy.AllowedEmails {
		if normalized := normalizeEmail(value); normalized != "" {
			guard.allowedEmails[normalized] = struct{}{}
		}
	}
	for _, value := range policy.AllowedDomains {
		if normalized := normalizeDomain(value); normalized != "" {
			guard.allowedDomains[normalized] = struct{}{}
		}
	}

	return guard
}

func (g IngressGuard) CheckIP(ip string) error {
	ip = strings.TrimSpace(ip)
	if len(g.allowedIPs) == 0 {
		return nil
	}

	normalized := normalizeIP(ip)
	if normalized == "" {
		return fmt.Errorf("%w: %q", ErrIPNotAllowed, ip)
	}
	if _, ok := g.allowedIPs[normalized]; ok {
		return nil
	}

	return fmt.Errorf("%w: %s", ErrIPNotAllowed, ip)
}

func (g IngressGuard) CheckIdentity(email string) error {
	email = strings.TrimSpace(email)
	if email == "" {
		return fmt.Errorf("%w: empty email", ErrIdentityNotAllowed)
	}
	if len(g.allowedEmails) == 0 && len(g.allowedDomains) == 0 {
		return fmt.Errorf("%w: no identity allowlist configured", ErrIdentityNotAllowed)
	}

	if normalized := normalizeEmail(email); normalized != "" {
		if _, ok := g.allowedEmails[normalized]; ok {
			return nil
		}
	}

	_, domain, found := strings.Cut(strings.ToLower(email), "@")
	if !found || domain == "" {
		return fmt.Errorf("%w: invalid email %q", ErrIdentityNotAllowed, email)
	}
	if _, ok := g.allowedDomains[domain]; ok {
		return nil
	}

	return fmt.Errorf("%w: %s", ErrIdentityNotAllowed, email)
}

func normalizeEmail(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return strings.ToLower(value)
}

func normalizeDomain(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return strings.ToLower(value)
}

func normalizeIP(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	parsed := net.ParseIP(value)
	if parsed == nil {
		return ""
	}
	return parsed.String()
}

func (g IngressGuard) hasPolicy() bool {
	return len(g.allowedIPs) > 0 || len(g.allowedEmails) > 0 || len(g.allowedDomains) > 0
}

func (g IngressGuard) authorize(headers http.Header) error {
	if !g.hasPolicy() {
		return nil
	}

	if len(g.allowedIPs) > 0 {
		if err := g.CheckIP(requestIP(headers)); err != nil {
			return err
		}
	}

	if len(g.allowedEmails) > 0 || len(g.allowedDomains) > 0 {
		if err := g.CheckIdentity(requestEmail(headers)); err != nil {
			return err
		}
	}

	return nil
}

func requestIP(headers http.Header) string {
	if headers == nil {
		return ""
	}

	if forwardedFor := headers.Get("X-Forwarded-For"); forwardedFor != "" {
		if ip, _, found := strings.Cut(forwardedFor, ","); found {
			return strings.TrimSpace(ip)
		}
		return strings.TrimSpace(forwardedFor)
	}

	if realIP := headers.Get("X-Real-IP"); realIP != "" {
		return strings.TrimSpace(realIP)
	}

	return ""
}

func requestEmail(headers http.Header) string {
	if headers == nil {
		return ""
	}

	for _, key := range []string{"X-Authenticated-Email", "X-User-Email", "X-Forwarded-Email"} {
		if value := strings.TrimSpace(headers.Get(key)); value != "" {
			return value
		}
	}

	return ""
}
