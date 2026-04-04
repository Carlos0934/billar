package mcp

import (
	"fmt"
	"strings"

	"github.com/Carlos0934/billar/internal/app"
)

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

func customerListText(result app.ListResult[app.CustomerDTO]) string {
	var builder strings.Builder
	builder.WriteString("Billar Customers\n")
	builder.WriteString("───────────────\n")
	builder.WriteString(fmt.Sprintf("Page: %d\n", result.Page))
	builder.WriteString(fmt.Sprintf("Page size: %d\n", result.PageSize))
	builder.WriteString(fmt.Sprintf("Total: %d\n", result.Total))

	if len(result.Items) == 0 {
		builder.WriteString("No customers found\n")
		return builder.String()
	}

	builder.WriteString("\n")
	for i, customer := range result.Items {
		if i > 0 {
			builder.WriteString("\n")
		}
		builder.WriteString(fmt.Sprintf("%d. %s\n", i+1, customer.LegalName))
		if customer.TradeName != "" && customer.TradeName != customer.LegalName {
			builder.WriteString(fmt.Sprintf("   Trade name: %s\n", customer.TradeName))
		}
		builder.WriteString(fmt.Sprintf("   Type: %s\n", customer.Type))
		builder.WriteString(fmt.Sprintf("   Status: %s\n", customer.Status))
		if customer.Email != "" {
			builder.WriteString(fmt.Sprintf("   Email: %s\n", customer.Email))
		}
		if customer.DefaultCurrency != "" {
			builder.WriteString(fmt.Sprintf("   Default currency: %s\n", customer.DefaultCurrency))
		}
		if customer.CreatedAt != "" {
			builder.WriteString(fmt.Sprintf("   Created at: %s\n", customer.CreatedAt))
		}
		if customer.UpdatedAt != "" {
			builder.WriteString(fmt.Sprintf("   Updated at: %s\n", customer.UpdatedAt))
		}
	}

	return builder.String()
}
