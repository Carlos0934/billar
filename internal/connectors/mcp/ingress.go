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
	ErrIPNotAllowed = errors.New("ip not allowed")
)

type IngressGuard struct {
	allowedIPs map[string]struct{}
}

func NewIngressGuard(allowedIPs []string) IngressGuard {
	guard := IngressGuard{
		allowedIPs: make(map[string]struct{}, len(allowedIPs)),
	}

	for _, value := range allowedIPs {
		if normalized := normalizeIP(value); normalized != "" {
			guard.allowedIPs[normalized] = struct{}{}
		}
	}

	return guard
}

func NewIngressGuardFromConfig(policy config.AccessPolicy) IngressGuard {
	return NewIngressGuard(policy.AllowedIPs)
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
	return len(g.allowedIPs) > 0
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
