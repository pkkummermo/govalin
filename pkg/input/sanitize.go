package input

import (
	"html"

	"github.com/microcosm-cc/bluemonday"
)

var bm = bluemonday.UGCPolicy()

// SantizeStringInput takess a string and santizes the input based on project config.
func sanitizeStringInput(s string) string {
	return html.UnescapeString(bm.Sanitize(s))
}
