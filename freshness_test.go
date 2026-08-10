package govalin_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/pkkummermo/govalin"
	"github.com/pkkummermo/govalin/govalintest"
	"github.com/stretchr/testify/assert"
)

func TestCacheForDeclaresFreshness(t *testing.T) {
	app := newTestApp()
	app.Get("/asset", func(call *govalin.Call) {
		call.CacheFor(10 * time.Minute)
		call.Text("the asset")
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		response := client.GetResponse("/asset")

		assert.Equal(t, "max-age=600", response.Header.Get("Cache-Control"), "Should declare the lifetime in seconds")
		assert.Equal(t, "the asset", readBody(t, response), "Should serve the representation")
	})
}

// TestCacheForFloorsAtZero covers the two durations that cannot be a lifetime:
// max-age counts whole seconds, and a negative one is not a valid header value.
// Both mean the client has to ask, which is what a zero says.
func TestCacheForFloorsAtZero(t *testing.T) {
	durations := map[string]time.Duration{
		"/sub-second": 900 * time.Millisecond,
		"/negative":   -time.Hour,
	}

	app := newTestApp()

	for path, duration := range durations {
		app.Get(path, func(call *govalin.Call) {
			call.CacheFor(duration)
			call.Text("the asset")
		})
	}

	govalintest.Test(t, app, func(client *govalintest.Client) {
		for path := range durations {
			response := client.GetResponse(path)

			assert.Equal(
				t,
				"max-age=0",
				response.Header.Get("Cache-Control"),
				"A %s duration should be a lifetime of zero", strings.TrimPrefix(path, "/"),
			)
		}
	})
}

// TestCachePrivateForKeepsAResponseOutOfSharedCaches covers the case an
// authenticated response needs: fresh in the browser that asked for it, stored
// by nothing in between.
func TestCachePrivateForKeepsAResponseOutOfSharedCaches(t *testing.T) {
	app := newTestApp()
	app.Get("/me", func(call *govalin.Call) {
		call.CachePrivateFor(time.Minute)
		call.Text("your profile")
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		response := client.GetResponse("/me")

		assert.Equal(t, "private, max-age=60", response.Header.Get("Cache-Control"), "Should name one cache only")
	})
}

func TestNoStoreForbidsStoringTheResponse(t *testing.T) {
	app := newTestApp()
	app.Get("/token", func(call *govalin.Call) {
		call.NoStore()
		call.Text("the token")
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		response := client.GetResponse("/token")

		assert.Equal(t, "no-store", response.Header.Get("Cache-Control"), "Should forbid storing the response")
	})
}

// TestFreshnessReplacesAnEarlierDeclaration covers a group-wide default being
// overridden by the one route that must not be stored. Two Cache-Control values
// on one response are read as one list, so appending would leave the response
// both cacheable and not.
func TestFreshnessReplacesAnEarlierDeclaration(t *testing.T) {
	app := newTestApp()
	app.Before("/api/*", func(call *govalin.Call) bool {
		call.CacheFor(time.Hour)

		return true
	})
	app.Get("/api/token", func(call *govalin.Call) {
		call.NoStore()
		call.Text("the token")
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		response := client.GetResponse("/api/token")

		assert.Equal(
			t,
			[]string{"no-store"},
			response.Header.Values("Cache-Control"),
			"The last declaration should be the only one",
		)
	})
}

// TestFreshnessSurvivesARevalidation covers what a 304 is for once freshness is
// declared: the client stored the response, came back when it went stale, and
// the answer has to renew the lifetime or it revalidates on every request from
// here on.
func TestFreshnessSurvivesARevalidation(t *testing.T) {
	app := newTestApp()
	app.Get("/asset", func(call *govalin.Call) {
		call.CacheFor(10 * time.Minute)

		if call.NotModified("v3") {
			return
		}

		call.Text("the asset")
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		response := revalidate(t, client, http.MethodGet, "/asset", `"v3"`)

		assert.Equal(t, http.StatusNotModified, response.StatusCode, "Should answer 304 on a match")
		assert.Equal(
			t,
			"max-age=600",
			response.Header.Get("Cache-Control"),
			"A 304 should renew the lifetime of the stored response",
		)
	})
}
