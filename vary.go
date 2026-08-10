package govalin

import (
	"net/http"
	"slices"
	"strings"

	"github.com/pkkummermo/govalin/internal/http/headers"
)

// VaryOn names the request headers this response was selected from, so a cache
// stores one copy per distinct value of them rather than one per URL. Declare
// it wherever the response depends on something the client asked for: a
// negotiated body, a language, anything read off a cookie or Authorization.
//
// Unlike a lifetime, this adds rather than replaces. Two layers can each have a
// claim on the same response — a CORS plugin varying on Origin, the handler
// below it on Accept-Language — and both are declared. A header named twice is
// declared once.
//
// Declare it before answering a revalidation: a 304 carries the Vary the 200
// would have, and NotModified returning true is the point the handler stops.
//
// For a response no cache may reuse at all, NoStore is the one to call.
func (call *Call) VaryOn(names ...string) {
	header := call.w.Header()
	selecting := declaredVary(header)

	for _, name := range names {
		if name = http.CanonicalHeaderKey(name); name != "" && !slices.Contains(selecting, name) {
			selecting = append(selecting, name)
		}
	}

	if len(selecting) == 0 {
		return
	}

	header.Set(headers.Vary, strings.Join(selecting, ", "))
}

// declaredVary reads the response's Vary as the one list several field lines
// are, per RFC 9110 §5.3.
func declaredVary(header http.Header) []string {
	var declared []string

	for _, line := range header.Values(headers.Vary) {
		for _, name := range strings.Split(line, ",") {
			name = http.CanonicalHeaderKey(strings.TrimSpace(name))
			if name != "" && !slices.Contains(declared, name) {
				declared = append(declared, name)
			}
		}
	}

	return declared
}
