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
	"github.com/pkkummermo/govalin/govalintest"
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

// newAccessLogApp builds an app independent of the quiet test defaults so the
// access log setting can be controlled per-test.
func newAccessLogApp(enabled bool, register func(app *govalin.App)) *govalin.App {
	app := govalin.New(func(config *govalin.Config) {
		config.EnableAccessLog(enabled)
		config.EnableStartupLog(false)
	})
	register(app)
	return app
}

// TestAccessLogHTTPServeIsLogged is the core regression: routes registered via
// HTTPServe set call.bypassLifecycle, which used to short-circuit out of the
// handler before the access log was written. They must now be logged.
func TestAccessLogHTTPServeIsLogged(t *testing.T) {
	var buf syncBuffer
	app := newAccessLogApp(true, func(app *govalin.App) {
		app.HTTPServe("/httpserve", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("httpserve"))
		})
	})
	govalintest.Test(t, app, func(client *govalintest.Client) {
		defer captureAccessLog(&buf)()

		client.Get("/httpserve")
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
	})
}

// TestAccessLogEnabledLogsOnce ensures a normal request that passes through
// before, endpoint and after handlers is logged exactly once (not duplicated).
func TestAccessLogEnabledLogsOnce(t *testing.T) {
	var buf syncBuffer
	app := newAccessLogApp(true, func(app *govalin.App) {
		app.Before("/*", func(_ *govalin.Call) bool { return true })
		app.Get("/test", func(call *govalin.Call) { call.Text("govalin") })
		app.After("/*", func(_ *govalin.Call) {})
	})
	govalintest.Test(t, app, func(client *govalintest.Client) {
		defer captureAccessLog(&buf)()

		client.Get("/test")
		time.Sleep(accessLogSettleTime)

		assert.Equal(
			t,
			1,
			accessLogCount(buf.String()),
			"A normal request should be access-logged exactly once",
		)
	})
}

// TestAccessLogShortCircuitBeforeIsLogged ensures a request short-circuited by
// a before handler is still logged when access logging is enabled.
func TestAccessLogShortCircuitBeforeIsLogged(t *testing.T) {
	var buf syncBuffer
	app := newAccessLogApp(true, func(app *govalin.App) {
		app.Before("/*", func(call *govalin.Call) bool {
			call.Text("short")
			return false
		})
		app.Get("/test", func(call *govalin.Call) { call.Text("govalin") })
	})
	govalintest.Test(t, app, func(client *govalintest.Client) {
		defer captureAccessLog(&buf)()

		client.Get("/test")
		time.Sleep(accessLogSettleTime)

		assert.Equal(
			t,
			1,
			accessLogCount(buf.String()),
			"A short-circuiting before handler should still be access-logged exactly once",
		)
	})
}

// TestAccessLogNotFoundIsLogged ensures unmatched requests (404) are logged.
func TestAccessLogNotFoundIsLogged(t *testing.T) {
	var buf syncBuffer
	app := newAccessLogApp(true, func(_ *govalin.App) {})
	govalintest.Test(t, app, func(client *govalintest.Client) {
		defer captureAccessLog(&buf)()

		client.Get("/does-not-exist")
		time.Sleep(accessLogSettleTime)

		assert.Equal(
			t,
			1,
			accessLogCount(buf.String()),
			"A 404 request should be access-logged exactly once",
		)
	})
}

// TestAccessLogSanitizesRequestDerivedValues covers the access log recording
// values a client controls: nothing else caps them, so a request must not get
// to decide how many bytes its log line costs.
func TestAccessLogSanitizesRequestDerivedValues(t *testing.T) {
	var buf syncBuffer
	app := newAccessLogApp(true, func(app *govalin.App) {
		app.Get("/test", func(call *govalin.Call) { call.Text("govalin") })
	})
	govalintest.Test(t, app, func(client *govalintest.Client) {
		defer captureAccessLog(&buf)()

		req, err := http.NewRequest(http.MethodGet, "/test", nil)
		assert.Nil(t, err)
		// A tab is the only control character net/http will put on the wire, and
		// it is enough to show control characters are dropped rather than logged.
		req.Header.Set("User-Agent", "evil\tagent"+strings.Repeat("a", 1000))
		response := client.Do(req)
		_ = readBody(t, response)
		time.Sleep(accessLogSettleTime)

		log := buf.String()

		assert.Contains(
			t,
			log,
			"evilagent",
			"Control characters should be dropped from the logged user agent",
		)
		assert.NotContains(
			t,
			log,
			strings.Repeat("a", 1000),
			"An oversized user agent should be truncated before it reaches the log",
		)
		assert.Contains(
			t,
			log,
			"…",
			"A truncated value should be marked as truncated",
		)
	})
}

// TestAccessLogDisabledLogsNothing ensures that when access logging is disabled
// nothing is logged, including the short-circuit before-handler path that used
// to log unconditionally regardless of the config flag.
func TestAccessLogDisabledLogsNothing(t *testing.T) {
	var buf syncBuffer
	app := newAccessLogApp(false, func(app *govalin.App) {
		app.Before("/short", func(call *govalin.Call) bool {
			call.Text("short")
			return false
		})
		app.Get("/test", func(call *govalin.Call) { call.Text("govalin") })
		app.HTTPServe("/httpserve", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("httpserve"))
		})
	})
	govalintest.Test(t, app, func(client *govalintest.Client) {
		defer captureAccessLog(&buf)()

		client.Get("/test")
		client.Get("/short")
		client.Get("/httpserve")
		client.Get("/does-not-exist")
		time.Sleep(accessLogSettleTime)

		assert.Equal(
			t,
			0,
			accessLogCount(buf.String()),
			"No access log should be written for any path when access logging is disabled",
		)
	})
}
