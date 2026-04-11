package core

import (
	"crypto/rand"
	"encoding/hex"
)

func generatePrefixedID(prefix string, size int) string {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return ""
	}

	encoded := hex.EncodeToString(buf)
	if len(encoded) != size*2 {
		return ""
	}

	return prefix + encoded
}
