package input

import (
	"strings"
)

// NormalizeStringInput takes a string and trims it for whitespaces and other riff-raff.
func NormalizeStringInput(s string) string {
	trimmedText := trimStringInput(s)
	return sanitizeStringInput(trimmedText)
}

// trimStringInput trimps the input string for space.
func trimStringInput(s string) string {
	return strings.Trim(s, " ")
}
