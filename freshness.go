package govalin

import (
	"strconv"
	"strings"
	"time"

	"github.com/pkkummermo/govalin/internal/http/headers"
)

// CacheFor declares how long the response stays fresh. Until that runs out the
// client reuses its stored copy without asking again.
//
// A duration under a second, or a negative one, declares a lifetime of zero:
// the response may be stored, but is revalidated before every reuse.
//
// Nothing here says who may store the response — a shared cache decides that
// for itself, which is what CachePrivateFor and NoStore are for. Whichever of
// the three is called last is the one that answers.
//
// The exception is a response that sets a cookie, which is narrowed to the
// client that asked for it whatever lifetime it declares.
func (call *Call) CacheFor(duration time.Duration) {
	call.declareCache("max-age=" + maxAgeSeconds(duration))
}

// CachePrivateFor is CacheFor for a response only the client that asked for it
// may store: fresh in that browser, held by no cache in between. This is the
// shape an authenticated response needs.
func (call *Call) CachePrivateFor(duration time.Duration) {
	call.w.Header().Set(headers.CacheControl, "private, max-age="+maxAgeSeconds(duration))
}

// NoCache has the client keep the response but check before every reuse, which
// is what an entry point naming versioned resources needs: never stale, and —
// paired with a validator — answered by a 304 rather than a body.
//
// A response that sets a cookie is kept by that client alone, as under CacheFor.
func (call *Call) NoCache() {
	call.declareCache("no-cache")
}

// declareCache writes a freshness declaration, narrowed if the response is
// already setting a cookie (ADR 0013).
func (call *Call) declareCache(freshness string) {
	if call.w.Header().Get(headers.SetCookie) != "" {
		freshness = "private, " + freshness
	}

	call.w.Header().Set(headers.CacheControl, freshness)
}

// cachePrivate narrows the response to the client that asked for it, keeping the
// freshness already declared (ADR 0013).
func (call *Call) cachePrivate() {
	freshness := call.w.Header().Get(headers.CacheControl)

	switch {
	case freshness == "":
		call.w.Header().Set(headers.CacheControl, "private")
	case !strings.Contains(freshness, "private") && !strings.Contains(freshness, "no-store"):
		call.w.Header().Set(headers.CacheControl, "private, "+freshness)
	}
}

// NoStore forbids storing the response at all, for a body that must not outlive
// the request that asked for it.
func (call *Call) NoStore() {
	call.w.Header().Set(headers.CacheControl, "no-store")
}

func maxAgeSeconds(duration time.Duration) string {
	seconds := int64(duration / time.Second)
	if seconds < 0 {
		seconds = 0
	}

	return strconv.FormatInt(seconds, 10)
}
