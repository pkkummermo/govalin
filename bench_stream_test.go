package govalin

import (
	"bytes"
	"net/http"
	"testing"
	"time"
)

// BenchmarkStreamLargeBody measures the new sink against the raw-writer
// workaround it replaces, over a real connection.
func BenchmarkStreamLargeBody(b *testing.B) {
	body := make([]byte, benchBodySize)

	host := startBenchApp(b, func(app *App) {
		app.Get("/stream", func(call *Call) {
			_ = call.Stream("application/octet-stream", newRepeatReader(body))
		})
	})

	benchDownload(b, host, "/stream", benchBodySize)
}

// BenchmarkServeContentLargeBody is the seekable path, which is what static
// mounts now use.
func BenchmarkServeContentLargeBody(b *testing.B) {
	body := make([]byte, benchBodySize)

	host := startBenchApp(b, func(app *App) {
		app.Get("/content", func(call *Call) {
			call.ServeContent("payload.bin", time.Unix(0, 0), bytes.NewReader(body))
		})
	})

	benchDownload(b, host, "/content", benchBodySize)
}

// BenchmarkServeContentSmallBody is the shape a static mount serves most: a
// small file, where per-request overhead dominates throughput.
func BenchmarkServeContentSmallBody(b *testing.B) {
	body := make([]byte, 4<<10)

	app := benchApp(func(app *App) {
		app.Get("/content", func(call *Call) {
			call.ServeContent("payload.bin", time.Unix(0, 0), bytes.NewReader(body))
		})
	})

	benchServe(b, app, http.MethodGet, "/content", http.StatusOK, len(body))
}
