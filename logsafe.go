package govalin

import (
	"strings"
	"unicode"
)

const (
	// maxLoggedValueLength bounds a request-derived value in the access log. Long
	// enough for a realistic user agent or path, short enough that a request
	// cannot decide how much a log line costs.
	maxLoggedValueLength = 256

	// truncationMarker is appended to a value that was cut short, so a truncated
	// value is never mistaken for the value the client actually sent.
	truncationMarker = "…"
)

// logSafeValue bounds a request-derived value and drops control characters,
// marking truncation so a cut value is not read as what the client sent. The
// bound is the point — nothing else caps a client-chosen path or user agent.
// Scrubbing is defence in depth: slog's own handlers escape control characters,
// but a percent-encoded path does decode into raw ones.
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
