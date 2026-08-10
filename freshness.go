package govalin

import (
	"strconv"
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
func (call *Call) CacheFor(duration time.Duration) {
	call.w.Header().Set(headers.CacheControl, "max-age="+maxAgeSeconds(duration))
}

// CachePrivateFor is CacheFor for a response only the client that asked for it
// may store: fresh in that browser, held by no cache in between. This is the
// shape an authenticated response needs.
func (call *Call) CachePrivateFor(duration time.Duration) {
	call.w.Header().Set(headers.CacheControl, "private, max-age="+maxAgeSeconds(duration))
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
