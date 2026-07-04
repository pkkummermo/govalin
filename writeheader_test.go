package govalin_test

import (
	"bytes"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/pkkummermo/govalin"
	"github.com/pkkummermo/govalin/govalintest"
)

// writeHeaderWarnings are the net/http log messages that signal a handler took
// over the raw writer without bypassing the govalin lifecycle, so the buffered
// status got flushed a second time. Catching them guards against regressions:
//   - "superfluous response.WriteHeader call" — a second WriteHeader on a
//     normal response (e.g. a handler delegating to a standard-library file
//     server without bypassing the lifecycle).
//   - "response.WriteHeader on hijacked connection" — a WriteHeader on a
//     connection a handler already hijacked (e.g. a websocket upgrade without a
//     lifecycle bypass).
var writeHeaderWarnings = []string{
	"superfluous response.WriteHeader call",
	"response.WriteHeader on hijacked connection",
}

// safeBuffer is a concurrency-safe buffer. net/http logs warnings from the
// per-connection goroutines that serve requests, so writes to the capture
// buffer can happen concurrently with each other.
type safeBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *safeBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *safeBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// logCapture mirrors everything logged during the test run into memory. The
// govalin server leaves http.Server.ErrorLog unset, so net/http logs its
// warnings — including the superfluous WriteHeader warning — through the
// standard log package, which slog.SetDefault re-routes through the slog
// default handler. We therefore capture at the slog layer (see TestMain).
var logCapture = &safeBuffer{}

// TestMain installs a package-wide guard: it makes the slog default handler tee
// every record into an in-memory buffer for the duration of the run, then fails
// the whole run if any test triggered one of the writeHeaderWarnings (a second
// WriteHeader on a normal response, or a WriteHeader on an already-hijacked
// connection). Capturing at the slog layer (rather than via log.SetOutput) is what
// makes this robust: slog.SetDefault re-routes the standard log package — where
// net/http emits the warning — through the current slog handler, and the
// access-log tests that swap the slog default restore it to whatever was
// default when they started, i.e. this teeing handler. This way the regression
// is caught no matter which test exercises the offending code path, not just
// the dedicated test below.
func TestMain(m *testing.M) {
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(io.MultiWriter(os.Stderr, logCapture), nil)))

	code := m.Run()

	slog.SetDefault(previous)

	failed := false
	for _, line := range strings.Split(logCapture.String(), "\n") {
		for _, warning := range writeHeaderWarnings {
			if strings.Contains(line, warning) {
				_, _ = os.Stderr.WriteString("FAIL: " + line + "\n")
				failed = true
			}
		}
	}
	if failed {
		os.Exit(1)
	}

	os.Exit(code)
}

// TestStaticServingDoesNotDoubleWriteHeader explicitly drives every static
// file-serving path so the lifecycle/file-server interaction is covered even if
// no other test happens to hit it. The actual assertion lives in TestMain,
// which inspects the captured net/http warnings for the whole package.
func TestStaticServingDoesNotDoubleWriteHeader(t *testing.T) {
	staticRoot, _ := fs.Sub(staticTestFiles, "internal/testdata/static")

	// Embedded FS, both plain and SPA mode.
	fsApp := newTestApp()
	fsApp.Static("/fs", func(_ *govalin.Call, staticConfig *govalin.StaticConfig) {
		staticConfig.WithFS(staticRoot)
	})
	fsApp.Static("/fs-spa", func(_ *govalin.Call, staticConfig *govalin.StaticConfig) {
		staticConfig.WithFS(staticRoot).EnableSPAMode(true)
	})
	govalintest.Test(t, fsApp, func(client *govalintest.Client) {
		client.Get("/fs/index.html")         // served via http.FileServer
		client.Get("/fs/")                   // index via serveIndexFS
		client.Get("/fs/sub/test.html")      // subfolder file via http.FileServer
		client.Get("/fs/non-existing-path")  // 404 via http.FileServer
		client.Get("/fs-spa/does-not-exist") // SPA fallback to index
	})

	// On-disk static directory, both plain and SPA mode.
	staticApp := newTestApp()
	staticApp.Static("/static", func(_ *govalin.Call, staticConfig *govalin.StaticConfig) {
		staticConfig.WithStaticPath("internal/testdata/static")
	})
	staticApp.Static("/static-spa", func(_ *govalin.Call, staticConfig *govalin.StaticConfig) {
		staticConfig.WithStaticPath("internal/testdata/static").EnableSPAMode(true)
	})
	govalintest.Test(t, staticApp, func(client *govalintest.Client) {
		client.Get("/static/index.html")         // served via http.FileServer
		client.Get("/static/")                   // index via serveIndexStatic (http.ServeFile)
		client.Get("/static/sub/test.html")      // subfolder file via http.FileServer
		client.Get("/static/non-existing-path")  // 404 via http.FileServer
		client.Get("/static-spa/does-not-exist") // SPA fallback to index
	})
}
