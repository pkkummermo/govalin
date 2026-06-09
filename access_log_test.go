package govalin_test

import (
	"bytes"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pkkummermo/govalin"
	"github.com/pkkummermo/govalin/internal/govalintesting"
	"github.com/stretchr/testify/assert"
)

// accessLogSettleTime gives the server goroutine time to flush the deferred
// access log before the test goroutine inspects the captured output.
const accessLogSettleTime = 20 * time.Millisecond

// syncBuffer is a goroutine-safe buffer. The access log is written from the
// server goroutine while assertions read it from the test goroutine.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// captureAccessLog swaps the default slog logger for one writing into buf and
// returns a restore function meant to be deferred.
func captureAccessLog(buf *syncBuffer) func() {
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(buf, nil)))
	return func() { slog.SetDefault(previous) }
}

// accessLogCount counts how many access log lines were emitted.
func accessLogCount(log string) int {
	return strings.Count(log, `msg="incoming request"`)
}

// newAccessLogApp builds an app independent of HTTPTestUtil's defaults so the
// access log setting can be controlled per-test. HTTPTestUtil starts whichever
// app is returned, so returning our own overrides its EnableAccessLog(false).
func newAccessLogApp(enabled bool, register func(app *govalin.App)) govalintesting.TestFunc {
	return func(_ *govalin.App) *govalin.App {
		app := govalin.New(func(config *govalin.Config) {
			config.EnableAccessLog(enabled)
			config.EnableStartupLog(false)
		})
		register(app)
		return app
	}
}

// TestAccessLogHTTPServeIsLogged is the core regression: routes registered via
// HTTPServe set call.bypassLifecycle, which used to short-circuit out of the
// handler before the access log was written. They must now be logged.
func TestAccessLogHTTPServeIsLogged(t *testing.T) {
	var buf syncBuffer
	govalintesting.HTTPTestUtil(
		newAccessLogApp(true, func(app *govalin.App) {
			app.HTTPServe("/httpserve", func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte("httpserve"))
			})
		}),
		func(http govalintesting.GovalinHTTP) {
			defer captureAccessLog(&buf)()

			http.Get("/httpserve")
			time.Sleep(accessLogSettleTime)

			assert.Equal(
				t,
				1,
				accessLogCount(buf.String()),
				"HTTPServe (bypassLifecycle) route should be access-logged exactly once when enabled",
			)
			assert.Contains(
				t,
				buf.String(),
				"path=/httpserve",
				"Access log should contain the request path",
			)
		},
	)
}

// TestAccessLogEnabledLogsOnce ensures a normal request that passes through
// before, endpoint and after handlers is logged exactly once (not duplicated).
func TestAccessLogEnabledLogsOnce(t *testing.T) {
	var buf syncBuffer
	govalintesting.HTTPTestUtil(
		newAccessLogApp(true, func(app *govalin.App) {
			app.Before("/*", func(_ *govalin.Call) bool { return true })
			app.Get("/test", func(call *govalin.Call) { call.Text("govalin") })
			app.After("/*", func(_ *govalin.Call) {})
		}),
		func(http govalintesting.GovalinHTTP) {
			defer captureAccessLog(&buf)()

			http.Get("/test")
			time.Sleep(accessLogSettleTime)

			assert.Equal(
				t,
				1,
				accessLogCount(buf.String()),
				"A normal request should be access-logged exactly once",
			)
		},
	)
}

// TestAccessLogShortCircuitBeforeIsLogged ensures a request short-circuited by
// a before handler is still logged when access logging is enabled.
func TestAccessLogShortCircuitBeforeIsLogged(t *testing.T) {
	var buf syncBuffer
	govalintesting.HTTPTestUtil(
		newAccessLogApp(true, func(app *govalin.App) {
			app.Before("/*", func(call *govalin.Call) bool {
				call.Text("short")
				return false
			})
			app.Get("/test", func(call *govalin.Call) { call.Text("govalin") })
		}),
		func(http govalintesting.GovalinHTTP) {
			defer captureAccessLog(&buf)()

			http.Get("/test")
			time.Sleep(accessLogSettleTime)

			assert.Equal(
				t,
				1,
				accessLogCount(buf.String()),
				"A short-circuiting before handler should still be access-logged exactly once",
			)
		},
	)
}

// TestAccessLogNotFoundIsLogged ensures unmatched requests (404) are logged.
func TestAccessLogNotFoundIsLogged(t *testing.T) {
	var buf syncBuffer
	govalintesting.HTTPTestUtil(
		newAccessLogApp(true, func(_ *govalin.App) {}),
		func(http govalintesting.GovalinHTTP) {
			defer captureAccessLog(&buf)()

			http.Get("/does-not-exist")
			time.Sleep(accessLogSettleTime)

			assert.Equal(
				t,
				1,
				accessLogCount(buf.String()),
				"A 404 request should be access-logged exactly once",
			)
		},
	)
}

// TestAccessLogDisabledLogsNothing ensures that when access logging is disabled
// nothing is logged, including the short-circuit before-handler path that used
// to log unconditionally regardless of the config flag.
func TestAccessLogDisabledLogsNothing(t *testing.T) {
	var buf syncBuffer
	govalintesting.HTTPTestUtil(
		newAccessLogApp(false, func(app *govalin.App) {
			app.Before("/short", func(call *govalin.Call) bool {
				call.Text("short")
				return false
			})
			app.Get("/test", func(call *govalin.Call) { call.Text("govalin") })
			app.HTTPServe("/httpserve", func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte("httpserve"))
			})
		}),
		func(http govalintesting.GovalinHTTP) {
			defer captureAccessLog(&buf)()

			http.Get("/test")
			http.Get("/short")
			http.Get("/httpserve")
			http.Get("/does-not-exist")
			time.Sleep(accessLogSettleTime)

			assert.Equal(
				t,
				0,
				accessLogCount(buf.String()),
				"No access log should be written for any path when access logging is disabled",
			)
		},
	)
}
