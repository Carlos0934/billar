package mcp

import (
	"fmt"
	"strings"

	"github.com/Carlos0934/billar/internal/app"
)

// syntheticMCPEmail, syntheticMCPSubject, syntheticMCPIssuer are the fixed
// values injected by the API-key auth middleware. They carry no real user
// information and should be hidden from human-readable output.
const (
	syntheticMCPEmail   = "mcp@local"
	syntheticMCPSubject = "mcp-api-key"
	syntheticMCPIssuer  = "billar://local"
)

// isSyntheticAPIKeyIdentity returns true when the DTO was produced by the
// fixed API-key identity injected by APIKeyAuthMiddleware.
func isSyntheticAPIKeyIdentity(dto app.SessionStatusDTO) bool {
	return dto.Email == syntheticMCPEmail &&
		dto.Subject == syntheticMCPSubject &&
		dto.Issuer == syntheticMCPIssuer
}

func sessionStatusText(dto app.SessionStatusDTO) string {
	if strings.EqualFold(strings.TrimSpace(dto.Status), "unauthenticated") && dto.Email == "" && !dto.EmailVerified && dto.Subject == "" && dto.Issuer == "" {
		return "Status: unauthenticated\n"
	}

	// Hide synthetic API-key identity fields — they are internal implementation
	// details, not real user information.
	if isSyntheticAPIKeyIdentity(dto) {
		return fmt.Sprintf("Status: %s\n", strings.TrimSpace(dto.Status))
	}

	var builder strings.Builder
	fmt.Fprintf(&builder, "Status: %s\n", strings.TrimSpace(dto.Status))
	if dto.Email != "" {
		fmt.Fprintf(&builder, "Email: %s\n", dto.Email)
	}
	if dto.EmailVerified {
		fmt.Fprintf(&builder, "Email verified: %t\n", dto.EmailVerified)
	}
	if dto.Subject != "" {
		fmt.Fprintf(&builder, "Subject: %s\n", dto.Subject)
	}
	if dto.Issuer != "" {
		fmt.Fprintf(&builder, "Issuer: %s\n", dto.Issuer)
	}

	return builder.String()
}
