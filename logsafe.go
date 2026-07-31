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

	// maxCorrelationIDLength bounds an inbound X-Govalin-Id.
	maxCorrelationIDLength = 128

	// truncationMarker is appended to a value that was cut short, so a truncated
	// value is never mistaken for the value the client actually sent.
	truncationMarker = "…"
)

// logSafeValue makes a request-derived value safe to write to a log sink:
// control characters (which forge line breaks and drive terminal escape
// sequences) are dropped and the result is bounded by maxLoggedValueLength.
// Ranging over the string also folds invalid UTF-8 into U+FFFD.
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

// isValidCorrelationID reports whether a client-supplied X-Govalin-Id may be
// adopted as the call ID. The ID is echoed into every log record for the
// request, so only a bounded, printable, whitespace-free token is trusted;
// anything else is discarded in favour of a generated one.
func isValidCorrelationID(id string) bool {
	if id == "" || len(id) > maxCorrelationIDLength {
		return false
	}

	for _, char := range []byte(id) {
		if char <= ' ' || char > '~' {
			return false
		}
	}

	return true
}
