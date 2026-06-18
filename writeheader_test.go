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
	"github.com/pkkummermo/govalin/internal/govalintesting"
)

// superfluousWriteHeaderWarning is the exact message net/http logs when a
// response header is written more than once. Catching it guards against
// handlers that delegate to the standard library file servers (or otherwise
// take over the raw writer) without bypassing the govalin lifecycle, which
// would flush the buffered status a second time.
const superfluousWriteHeaderWarning = "superfluous response.WriteHeader call"

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
// the whole run if any test triggered a superfluous response.WriteHeader
// warning. Capturing at the slog layer (rather than via log.SetOutput) is what
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

	if captured := logCapture.String(); strings.Contains(captured, superfluousWriteHeaderWarning) {
		for _, line := range strings.Split(captured, "\n") {
			if strings.Contains(line, superfluousWriteHeaderWarning) {
				_, _ = os.Stderr.WriteString("FAIL: " + line + "\n")
			}
		}
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
	govalintesting.HTTPTestUtil(func(app *govalin.App) *govalin.App {
		app.Static("/fs", func(_ *govalin.Call, staticConfig *govalin.StaticConfig) {
			staticConfig.WithFS(staticRoot)
		})
		app.Static("/fs-spa", func(_ *govalin.Call, staticConfig *govalin.StaticConfig) {
			staticConfig.WithFS(staticRoot).EnableSPAMode(true)
		})
		return app
	}, func(http govalintesting.GovalinHTTP) {
		http.Get("/fs/index.html")         // served via http.FileServer
		http.Get("/fs/")                   // index via serveIndexFS
		http.Get("/fs/sub/test.html")      // subfolder file via http.FileServer
		http.Get("/fs/non-existing-path")  // 404 via http.FileServer
		http.Get("/fs-spa/does-not-exist") // SPA fallback to index
	})

	// On-disk static directory, both plain and SPA mode.
	govalintesting.HTTPTestUtil(func(app *govalin.App) *govalin.App {
		app.Static("/static", func(_ *govalin.Call, staticConfig *govalin.StaticConfig) {
			staticConfig.WithStaticPath("internal/testdata/static")
		})
		app.Static("/static-spa", func(_ *govalin.Call, staticConfig *govalin.StaticConfig) {
			staticConfig.WithStaticPath("internal/testdata/static").EnableSPAMode(true)
		})
		return app
	}, func(http govalintesting.GovalinHTTP) {
		http.Get("/static/index.html")         // served via http.FileServer
		http.Get("/static/")                   // index via serveIndexStatic (http.ServeFile)
		http.Get("/static/sub/test.html")      // subfolder file via http.FileServer
		http.Get("/static/non-existing-path")  // 404 via http.FileServer
		http.Get("/static-spa/does-not-exist") // SPA fallback to index
	})
}
