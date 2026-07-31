package govalin

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// logSafeValue guards values a client fully controls, so it is tested directly
// rather than only through the server: net/http rejects some hostile header
// values before they ever reach a handler, and the check must hold regardless
// of what that layer happens to filter today.

func TestLogSafeValue(t *testing.T) {
	assert.Equal(
		t,
		"govalin",
		logSafeValue("govalin"),
		"An ordinary value should pass through unchanged",
	)
	assert.Equal(
		t,
		"forgedmsg=evil",
		logSafeValue("forged\r\nmsg=evil"),
		"Line breaks used to forge a second record should be dropped",
	)
	assert.Equal(
		t,
		"[0m",
		logSafeValue("\x1b[0m"),
		"Terminal escape sequences should lose their control character",
	)
	assert.Equal(
		t,
		strings.Repeat("a", maxLoggedValueLength)+truncationMarker,
		logSafeValue(strings.Repeat("a", maxLoggedValueLength+1)),
		"A value past the limit should be truncated and marked",
	)
	assert.Equal(
		t,
		strings.Repeat("a", maxLoggedValueLength),
		logSafeValue(strings.Repeat("a", maxLoggedValueLength)),
		"A value exactly at the limit should not be marked as truncated",
	)
	assert.Equal(
		t,
		"æøå",
		logSafeValue("æøå"),
		"The limit counts characters, so multi-byte text is not cut mid-rune",
	)
}
