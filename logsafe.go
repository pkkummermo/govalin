package govalin

import (
	"strings"
	"unicode"
)

const (
	// maxLoggedValueLength keeps a request from deciding how much a log line costs.
	maxLoggedValueLength = 256

	// truncationMarker marks a cut value so it is not read as what the client sent.
	truncationMarker = "…"
)

// logSafeValue bounds a request-derived value and drops control characters. The bound is
// the point — nothing else caps a client-chosen path or user agent.
func logSafeValue(value string) string {
	var builder strings.Builder

	length := 0

	for _, char := range value {
		if unicode.IsControl(char) {
			continue
		}

		if length == maxLoggedValueLength {
			builder.WriteString(truncationMarker)
			break
		}

		builder.WriteRune(char)
		length++
	}

	return builder.String()
}
