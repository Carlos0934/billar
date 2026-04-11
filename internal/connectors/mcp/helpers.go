package mcp

import (
	"strings"
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
