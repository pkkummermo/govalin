package govalin

import (
	"net/http"
	"strings"

	"github.com/pkkummermo/govalin/internal/http/headers"
)

// NotModified sets the response ETag and reports whether the client already
// holds this version of the resource.
//
// The tag is the caller's own validator — a row version, an updated_at, a digest
// it already has — quoted if it is not already.
//
// A true return means the client's If-None-Match named this version and a 304 is
// buffered. Producing no body is then an optimisation, not an obligation: one
// written anyway is refused by net/http, so the client gets a correct 304 either
// way.
//
// Only GET and HEAD are answered. On any other method If-None-Match is a
// precondition asking a different question, and this does nothing.
func (call *Call) NotModified(tag string) bool {
	if call.req.Method != http.MethodGet && call.req.Method != http.MethodHead {
		return false
	}

	etag := quoteETag(tag)
	call.w.Header().Set(headers.ETag, etag)

	if !etagMatches(call.req.Header.Get(headers.IfNoneMatch), etag) {
		return false
	}

	call.Status(http.StatusNotModified)

	return true
}

// quoteETag makes a caller's value a valid entity-tag. An unquoted one is a tag
// no client will ever match back; one that carries its own quotes is left be.
func quoteETag(tag string) string {
	if strings.HasPrefix(tag, `"`) || strings.HasPrefix(tag, `W/"`) {
		return tag
	}

	return `"` + tag + `"`
}

// etagMatches reports whether an If-None-Match header names the tag, by the weak
// comparison RFC 9110 requires of it: W/"x" and "x" are the same version. Scans
// in place, so the common case — no header at all — costs nothing.
//
// Only a comma outside a quoted tag separates the list. One inside is part of
// the tag, and a caller's version string is quoted as it stands, so a comma in
// one is not the exotic case it would be for a server that only sends digests.
func etagMatches(ifNoneMatch string, etag string) bool {
	if ifNoneMatch == "" {
		return false
	}

	if strings.TrimSpace(ifNoneMatch) == "*" {
		return true
	}

	target := strings.TrimPrefix(etag, "W/")
	quoted := false
	start := 0

	for index := 0; index <= len(ifNoneMatch); index++ {
		switch {
		case index < len(ifNoneMatch) && ifNoneMatch[index] == '"':
			quoted = !quoted
		case index == len(ifNoneMatch) || (ifNoneMatch[index] == ',' && !quoted):
			candidate := strings.TrimSpace(ifNoneMatch[start:index])
			if strings.TrimPrefix(candidate, "W/") == target {
				return true
			}

			start = index + 1
		}
	}

	return false
}
