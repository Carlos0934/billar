package mcp

import (
	"strings"

	"github.com/Carlos0934/billar/internal/app"
)

// parseSortValue extracts field and direction from a sort string
func parseSortValue(value string) (string, string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", ""
	}

	if strings.HasPrefix(value, "-") {
		return strings.TrimSpace(strings.TrimPrefix(value, "-")), "desc"
	}

	field, dir, found := strings.Cut(value, ":")
	if !found {
		return strings.TrimSpace(value), ""
	}

	return strings.TrimSpace(field), strings.TrimSpace(dir)
}

// ptrTo returns a pointer to the given string
func ptrTo(s string) *string {
	return &s
}

// extractAddressDTO builds an app.AddressDTO from a map[string]any argument
func extractAddressDTO(m map[string]any) app.AddressDTO {
	return app.AddressDTO{
		Street:     extractString(m, "street"),
		City:       extractString(m, "city"),
		State:      extractString(m, "state"),
		PostalCode: extractString(m, "postal_code"),
		Country:    extractString(m, "country"),
	}
}

// extractString safely extracts a string value from a map
func extractString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
