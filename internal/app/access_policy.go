package app

import (
	"strings"
)

type EnvAccessPolicy struct {
	AllowedEmails  []string
	AllowedDomains []string
}

func (p EnvAccessPolicy) IsAllowed(email string) bool {
	email = strings.TrimSpace(email)
	if email == "" {
		return false
	}

	for _, allowedEmail := range p.AllowedEmails {
		if strings.EqualFold(email, strings.TrimSpace(allowedEmail)) {
			return true
		}
	}

	_, domain, found := strings.Cut(email, "@")
	if !found {
		return false
	}

	domain = strings.TrimSpace(domain)
	if domain == "" {
		return false
	}

	for _, allowedDomain := range p.AllowedDomains {
		if strings.EqualFold(domain, strings.TrimSpace(allowedDomain)) {
			return true
		}
	}

	return false
}
