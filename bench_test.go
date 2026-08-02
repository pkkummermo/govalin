package govalin

import (
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

// discardWriter is a response writer that keeps nothing but the shape of the
// response, so a benchmark measures the framework rather than the recorder it
// writes into — while still being able to prove it measured what it meant to.
type discardWriter struct {
	header  http.Header
	status  int
	written int
}

func (writer *discardWriter) Header() http.Header {
	return writer.header
}

func (writer *discardWriter) Write(data []byte) (int, error) {
	writer.written += len(data)

	return len(data), nil
}

func (writer *discardWriter) WriteHeader(status int) {
	writer.status = status
}

// observedStatus mirrors net/http: a body written without an explicit header is
// a 200.
func (writer *discardWriter) observedStatus() int {
	if writer.status == 0 && writer.written > 0 {
		return http.StatusOK
	}

	return writer.status
}

func (writer *discardWriter) reset() {
	clear(writer.header)
	writer.status = 0
	writer.written = 0
}

// benchApp builds an app with logging off, ready to serve through
// rootHandlerFunc without binding a listener.
func benchApp(register func(app *App)) *App {
	app := New(func(config *Config) {
		config.EnableAccessLog(false)
		config.EnableStartupLog(false)
	})
	register(app)

	return app
}

// benchServe drives one request shape through the whole request lifecycle.
//
// It checks the response before timing anything: a benchmark that silently
// measures a redirect or an error instead of the work it names is worse than no
// benchmark at all.
func benchServe(b *testing.B, app *App, method string, target string, wantStatus int, wantBody int) {
	b.Helper()

	request := httptest.NewRequest(method, target, nil)
	writer := &discardWriter{header: http.Header{}}

	app.rootHandlerFunc(writer, request)
	if writer.observedStatus() != wantStatus || writer.written < wantBody {
		b.Fatalf(
			"%s %s answered %d with %d body bytes, expected %d with at least %d",
			method, target, writer.observedStatus(), writer.written, wantStatus, wantBody,
		)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		writer.reset()
		app.rootHandlerFunc(writer, request)
	}
}

func BenchmarkText(b *testing.B) {
	app := benchApp(func(app *App) {
		app.Get("/text", func(call *Call) { call.Text("Hello world") })
	})

	benchServe(b, app, http.MethodGet, "/text", http.StatusOK, 11)
}

func BenchmarkJSON(b *testing.B) {
	type payload struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}

	app := benchApp(func(app *App) {
		app.Get("/json", func(call *Call) { call.JSON(payload{Name: "govalin", Count: 42}) })
	})

	benchServe(b, app, http.MethodGet, "/json", http.StatusOK, 25)
}

// BenchmarkStatusOnly is the end-of-lifecycle status flush on its own: no body,
// so the only write is the one the framework makes.
func BenchmarkStatusOnly(b *testing.B) {
	app := benchApp(func(app *App) {
		app.Get("/status", func(call *Call) { call.Status(http.StatusNoContent) })
	})

	benchServe(b, app, http.MethodGet, "/status", http.StatusNoContent, 0)
}

func BenchmarkNotFound(b *testing.B) {
	app := benchApp(func(app *App) {
		app.Get("/text", func(call *Call) { call.Text("Hello world") })
	})

	benchServe(b, app, http.MethodGet, "/missing", http.StatusNotFound, 50)
}

// BenchmarkBeforeAfterHandlers covers the path with the most lifecycle checks.
func BenchmarkBeforeAfterHandlers(b *testing.B) {
	app := benchApp(func(app *App) {
		app.Before("/*", func(_ *Call) bool { return true })
		app.Get("/text", func(call *Call) { call.Text("Hello world") })
		app.After("/*", func(_ *Call) {})
	})

	benchServe(b, app, http.MethodGet, "/text", http.StatusOK, 11)
}

// BenchmarkAccessLog includes the log line, which now reads the written status.
func BenchmarkAccessLog(b *testing.B) {
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	b.Cleanup(func() { slog.SetDefault(previous) })

	app := New(func(config *Config) {
		config.EnableAccessLog(true)
		config.EnableStartupLog(false)
	})
	app.Get("/text", func(call *Call) { call.Text("Hello world") })

	benchServe(b, app, http.MethodGet, "/text", http.StatusOK, 11)
}

func BenchmarkStaticFile(b *testing.B) {
	app := benchApp(func(app *App) {
		app.Static("/static", func(_ *Call, staticConfig *StaticConfig) {
			staticConfig.WithStaticPath("internal/testdata/static")
		})
	})

	benchServe(b, app, http.MethodGet, "/static/sub/test.html", http.StatusOK, 100)
}

func BenchmarkHTTPServe(b *testing.B) {
	app := benchApp(func(app *App) {
		app.HTTPServe("/raw", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("Hello world"))
		})
	})

	benchServe(b, app, http.MethodGet, "/raw", http.StatusOK, 11)
}

// startBenchApp starts an app on an OS-assigned port and returns its base URL,
// so a benchmark can measure the real net/http write path rather than a
// stand-in writer.
func startBenchApp(b *testing.B, register func(app *App)) string {
	b.Helper()

	app := New(func(config *Config) {
		config.EnableAccessLog(false)
		config.EnableStartupLog(false)
	})
	register(app)

	ready := make(chan struct{})
	app.Events(func(events *ServerEvents) {
		events.AddOnServerStartup(func() { close(ready) })
	})

	go func() { _ = app.Start(0) }()
	<-ready

	b.Cleanup(func() { _ = app.Shutdown() })

	return fmt.Sprintf("http://localhost:%d", app.Port())
}

// benchDownload measures throughput of a large body over a real connection,
// which is where the writer wrapper could cost the sendfile path.
func benchDownload(b *testing.B, host string, path string, size int) {
	b.Helper()

	client := &http.Client{Transport: &http.Transport{
		MaxIdleConnsPerHost: 1,
		DialContext:         (&net.Dialer{}).DialContext,
	}}

	b.SetBytes(int64(size))
	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		response, err := client.Get(host + path)
		if err != nil {
			b.Fatalf("request failed: %v", err)
		}
		written, copyErr := io.Copy(io.Discard, response.Body)
		_ = response.Body.Close()
		if copyErr != nil {
			b.Fatalf("failed to read body: %v", copyErr)
		}
		if written != int64(size) {
			b.Fatalf("expected %d bytes, got %d", size, written)
		}
	}
}

const benchBodySize = 8 << 20

// BenchmarkRawWriteLargeBody is the workaround this change replaces: a handler
// copying to the raw writer itself.
func BenchmarkRawWriteLargeBody(b *testing.B) {
	body := make([]byte, benchBodySize)

	host := startBenchApp(b, func(app *App) {
		app.Get("/raw", func(call *Call) {
			call.Status(http.StatusOK)
			_, _ = io.Copy(*call.Raw.W, newRepeatReader(body))
		})
	})

	benchDownload(b, host, "/raw", benchBodySize)
}

// BenchmarkHTTPServeLargeBody is the same copy through a bare http.Handler, the
// only supported escape hatch before this change.
func BenchmarkHTTPServeLargeBody(b *testing.B) {
	body := make([]byte, benchBodySize)

	host := startBenchApp(b, func(app *App) {
		app.HTTPServe("/httpserve", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.Copy(w, newRepeatReader(body))
		})
	})

	benchDownload(b, host, "/httpserve", benchBodySize)
}

// repeatReader serves a fixed payload once, without the seeking a bytes.Reader
// offers, so every large-body benchmark reads the same way.
type repeatReader struct {
	data   []byte
	offset int
}

func newRepeatReader(data []byte) *repeatReader {
	return &repeatReader{data: data}
}

func (reader *repeatReader) Read(buffer []byte) (int, error) {
	if reader.offset >= len(reader.data) {
		return 0, io.EOF
	}

	read := copy(buffer, reader.data[reader.offset:])
	reader.offset += read

	return read, nil
}
