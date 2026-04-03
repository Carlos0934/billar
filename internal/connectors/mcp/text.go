package mcp

import (
	"fmt"
	"strings"

	"github.com/Carlos0934/billar/internal/app"
)

func loginIntentText(dto app.LoginIntentDTO) string {
	return fmt.Sprintf("Login URL: %s\n", dto.LoginURL)
}

func sessionStatusText(dto app.SessionStatusDTO) string {
	if strings.EqualFold(strings.TrimSpace(dto.Status), "unauthenticated") && dto.Email == "" && !dto.EmailVerified && dto.Subject == "" && dto.Issuer == "" {
		return "Status: unauthenticated\n"
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

func logoutText(dto app.LogoutDTO) string {
	message := strings.TrimSpace(dto.Message)
	if message == "" {
		message = "Logged out"
	}
	return message + "\n"
}
