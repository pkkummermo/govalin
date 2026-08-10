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

// TestNoCacheKeepsTheStoredCopyButAsksAnyway covers the shape an entry point
// needs: the client keeps the response, and finds out it is still current with
// a 304 rather than a download.
func TestNoCacheKeepsTheStoredCopyButAsksAnyway(t *testing.T) {
	app := newTestApp()
	app.Get("/index", func(call *govalin.Call) {
		call.NoCache()

		if call.NotModified("v3") {
			return
		}

		call.HTML("the shell")
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		response := client.GetResponse("/index")
		assert.Equal(t, "no-cache", response.Header.Get("Cache-Control"), "Should have the client ask every time")

		revalidated := revalidate(t, client, http.MethodGet, "/index", `"v3"`)
		assert.Equal(t, http.StatusNotModified, revalidated.StatusCode, "Asking should cost headers, not a body")
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

// TestASessionMintingResponseIsPrivate covers the response govalin sets a
// cookie on itself: a shared cache that stored it would hand the next visitor
// the same session, so it declares a scope before any handler runs.
func TestASessionMintingResponseIsPrivate(t *testing.T) {
	app := newTestApp(func(config *govalin.Config) {
		config.EnableSessions()
	})
	app.Get("/", func(call *govalin.Call) {
		call.Text("welcome")
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		response := client.GetResponse("/")

		assert.NotEmpty(t, response.Header.Get("Set-Cookie"), "The first visit should mint a session")
		assert.Equal(t, "private", response.Header.Get("Cache-Control"), "A minted session may not be shared")
	})
}

// TestALifetimeNarrowsOnAResponseThatSetsACookie covers the case a scope alone
// does not survive: a mount declaring a lifetime of its own, which last-call-wins
// would otherwise let replace it.
func TestALifetimeNarrowsOnAResponseThatSetsACookie(t *testing.T) {
	app := newTestApp(func(config *govalin.Config) {
		config.EnableSessions()
	})
	app.Get("/asset", func(call *govalin.Call) {
		call.CacheFor(time.Hour)
		call.Text("the asset")
	})
	app.Get("/shell", func(call *govalin.Call) {
		call.NoCache()
		call.HTML("the shell")
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		lifetime := client.GetResponse("/asset")
		assert.Equal(t, "private, max-age=3600", lifetime.Header.Get("Cache-Control"), "A lifetime should narrow")

		shell := client.GetResponse("/shell")
		assert.Equal(t, "private, no-cache", shell.Header.Get("Cache-Control"), "So should a revalidate-always")
	})
}

// TestACookieSetAfterALifetimeNarrowsItToo covers the other order: the handler
// declared how long its response lasts before it knew it would be setting a
// cookie, and a shared cache would hold the response for everyone either way.
func TestACookieSetAfterALifetimeNarrowsItToo(t *testing.T) {
	app := newTestApp()
	app.Get("/preferences", func(call *govalin.Call) {
		call.CacheFor(time.Hour)

		if _, err := call.Cookie("theme", &http.Cookie{Value: "dark", Path: "/"}); err != nil {
			t.Errorf("failed to set the cookie: %v", err)
		}

		call.Text("saved")
	})
	app.Get("/preferences-by-header", func(call *govalin.Call) {
		call.CacheFor(time.Hour)
		call.Header("Set-Cookie", "theme=dark; Path=/")
		call.Text("saved")
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		for _, path := range []string{"/preferences", "/preferences-by-header"} {
			response := client.GetResponse(path)

			assert.Equal(
				t,
				"private, max-age=3600",
				response.Header.Get("Cache-Control"),
				"Neither the order nor the way the cookie was set should decide who holds %s", path,
			)
		}
	})
}

// TestALifetimeStaysSharedForAClientThatHasASession covers what the narrowing is
// keyed on: the response setting a cookie, not the app having sessions. A client
// that already has one is served the shared lifetime the route asked for.
func TestALifetimeStaysSharedForAClientThatHasASession(t *testing.T) {
	app := newTestApp(func(config *govalin.Config) {
		config.EnableSessions()
	})
	app.Get("/asset", func(call *govalin.Call) {
		call.CacheFor(time.Hour)
		call.Text("the asset")
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		minted := client.GetResponse("/asset")

		request, err := http.NewRequest(http.MethodGet, client.Host+"/asset", nil)
		if err != nil {
			t.Fatalf("failed to build a request: %v", err)
		}
		request.Header.Set("Cookie", minted.Header.Get("Set-Cookie"))

		returning := client.Do(request)

		assert.Empty(t, returning.Header.Get("Set-Cookie"), "A client with a session should not be minted another")
		assert.Equal(t, "max-age=3600", returning.Header.Get("Cache-Control"), "Nothing here is one client's own")
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
