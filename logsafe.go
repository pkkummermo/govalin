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
