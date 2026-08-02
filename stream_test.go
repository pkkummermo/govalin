package govalin_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pkkummermo/govalin"
	"github.com/pkkummermo/govalin/govalintest"
	"github.com/stretchr/testify/assert"
)

// streamBody builds a deterministic payload of the given size so a ranged
// response can be checked against the bytes it should have returned.
func streamBody(size int) []byte {
	body := make([]byte, size)
	for i := range body {
		body[i] = byte('a' + i%26)
	}

	return body
}

// readerAtOnly exposes a payload as an io.ReaderAt and nothing else, standing
// in for a remote object store: ranged reads work, seeking does not exist.
type readerAtOnly struct {
	reader *bytes.Reader
}

func (r readerAtOnly) ReadAt(p []byte, off int64) (int, error) {
	return r.reader.ReadAt(p, off)
}

// failingReader stands in for content the server cannot read: a remote object
// that stops answering part-way through.
type failingReader struct {
	err error
}

func (r failingReader) Read(_ []byte) (int, error) {
	return 0, r.err
}

// blockingReader emits chunks until it is exhausted, pausing between them so a
// disconnecting client is noticed while the body is still being written.
type blockingReader struct {
	chunks int
	chunk  []byte
}

func (r *blockingReader) Read(p []byte) (int, error) {
	if r.chunks == 0 {
		return 0, io.EOF
	}
	r.chunks--
	time.Sleep(time.Millisecond)

	return copy(p, r.chunk), nil
}

func TestStreamSendsTheWholeBody(t *testing.T) {
	body := streamBody(2 << 20)

	app := newTestApp()
	app.Get("/stream", func(call *govalin.Call) {
		assert.Nil(t, call.Stream("application/octet-stream", bytes.NewReader(body)))
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		response := client.GetResponse("/stream")
		received := readBody(t, response)

		assert.Equal(t, http.StatusOK, response.StatusCode, "A stream defaults to 200 OK")
		assert.Equal(
			t,
			"application/octet-stream",
			response.Header.Get("Content-Type"),
			"Stream should set the given content type",
		)
		assert.Equal(t, len(body), len(received), "The whole body should reach the client")
		assert.Equal(t, string(body), received, "The streamed bytes should arrive unchanged")
	})
}

func TestStreamUsesTheBufferedStatus(t *testing.T) {
	app := newTestApp()
	app.Get("/stream", func(call *govalin.Call) {
		call.Status(http.StatusAccepted)
		assert.Nil(t, call.Stream("text/plain", strings.NewReader("accepted")))
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		response := client.GetResponse("/stream")

		assert.Equal(t, "accepted", readBody(t, response), "The body should be streamed")
		assert.Equal(
			t,
			http.StatusAccepted,
			response.StatusCode,
			"A status set before streaming should be the status sent",
		)
	})
}

// TestStreamClientDisconnectIsNotAnError covers a client that hangs up mid
// body: the header is long gone and there is nothing the server can do, so the
// framework must not report it as an error to the handler or to the log.
func TestStreamClientDisconnectIsNotAnError(t *testing.T) {
	var buf syncBuffer
	var streamErr error
	done := make(chan struct{})

	app := newTestApp()
	app.Get("/slow", func(call *govalin.Call) {
		defer close(done)
		streamErr = call.Stream("application/octet-stream", &blockingReader{
			chunks: 4096,
			chunk:  streamBody(32 << 10),
		})
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		defer captureAccessLog(&buf)()

		response := client.GetResponse("/slow")
		assert.Equal(t, http.StatusOK, response.StatusCode, "The header is sent before the body stalls")
		assert.Nil(t, response.Body.Close(), "Hang up while the body is still being written")

		select {
		case <-done:
		case <-time.After(10 * time.Second):
			t.Fatal("the handler never returned after the client disconnected")
		}
		time.Sleep(accessLogSettleTime)

		assert.Nil(t, streamErr, "A client that hangs up mid-stream is not a failure of the handler")
		assert.NotContains(
			t,
			buf.String(),
			"level=ERROR",
			"A client disconnect should not be logged at error level",
		)
	})
}

// TestStreamSniffsAnEmptyContentType covers the documented contract for an
// empty content type. Committing the status first does not prevent sniffing:
// net/http records the status on WriteHeader but writes (and sniffs) the header
// when the first body bytes are flushed.
func TestStreamSniffsAnEmptyContentType(t *testing.T) {
	app := newTestApp()
	app.Get("/sniff", func(call *govalin.Call) {
		assert.Nil(t, call.Stream("", strings.NewReader("<!DOCTYPE html><html><body>hi</body></html>")))
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		response := client.GetResponse("/sniff")
		_ = readBody(t, response)

		assert.Equal(
			t,
			"text/html; charset=utf-8",
			response.Header.Get("Content-Type"),
			"An empty content type should be sniffed from the body by net/http",
		)
	})
}

// TestFlushedResponseIsNotWrittenTwice covers a handler that flushes without
// writing a body: net/http headers the response with an implicit 200, so the
// lifecycle must treat it as committed. Otherwise it flushes a second status —
// the superfluous WriteHeader this change exists to prevent, caught for the
// whole package by TestMain.
func TestFlushedResponseIsNotWrittenTwice(t *testing.T) {
	var buf syncBuffer

	app := newAccessLogApp(true, func(app *govalin.App) {
		app.Get("/flush", func(call *govalin.Call) {
			call.Status(http.StatusAccepted)
			assert.Nil(t, http.NewResponseController(*call.Raw.W).Flush())
		})
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		defer captureAccessLog(&buf)()

		response := client.GetResponse("/flush")
		_ = readBody(t, response)
		time.Sleep(accessLogSettleTime)

		assert.Equal(
			t,
			http.StatusOK,
			response.StatusCode,
			"A flush commits the 200 net/http sends, whatever status was buffered",
		)
		assert.Contains(
			t,
			buf.String(),
			"status=200",
			"The access log should record the status the flush committed, not the buffered one",
		)
	})
}

// TestStreamReportsASourceFailure is the other half of the disconnect case: a
// failure to read the content is the handler's to know about, and must not be
// waved away as a client that hung up.
func TestStreamReportsASourceFailure(t *testing.T) {
	sourceErr := errors.New("the object store said no")

	app := newTestApp()
	app.Get("/stream", func(call *govalin.Call) {
		err := call.Stream("application/octet-stream", io.MultiReader(
			bytes.NewReader(streamBody(16)),
			failingReader{err: sourceErr},
		))

		assert.ErrorIs(t, err, sourceErr, "A source that fails should be reported to the handler")
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		response := client.GetResponse("/stream")
		_ = readBody(t, response)

		assert.Equal(
			t,
			http.StatusOK,
			response.StatusCode,
			"The header is already sent when the source fails, so the status stands",
		)
	})
}

func TestServeContentSupportsRangeRequests(t *testing.T) {
	body := streamBody(4 << 10)

	app := newTestApp()
	app.Get("/content", func(call *govalin.Call) {
		call.ServeContent("payload.bin", time.Unix(0, 0), bytes.NewReader(body))
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		full := client.GetResponse("/content")
		assert.Equal(t, http.StatusOK, full.StatusCode, "An unranged request gets the whole body")
		assert.Equal(t, "bytes", full.Header.Get("Accept-Ranges"), "Ranges should be advertised")
		assert.Equal(t, string(body), readBody(t, full), "The whole body should be served")

		partial := client.GetRange("/content", 0, 1023)
		partialBody := readBody(t, partial)

		assert.Equal(t, http.StatusPartialContent, partial.StatusCode, "A ranged request gets a 206")
		assert.Equal(
			t,
			fmt.Sprintf("bytes 0-1023/%d", len(body)),
			partial.Header.Get("Content-Range"),
			"A 206 should describe the range it carries",
		)
		assert.Len(t, partialBody, 1024, "A 206 should carry exactly the requested bytes")
		assert.Equal(t, string(body[:1024]), partialBody, "A 206 should carry the requested bytes")

		tail := client.GetRange("/content", 4000, -1)
		assert.Equal(t, http.StatusPartialContent, tail.StatusCode, "An open ended range gets a 206")
		assert.Equal(t, string(body[4000:]), readBody(t, tail), "An open ended range runs to the end")
	})
}

func TestServeContentRejectsUnsatisfiableRange(t *testing.T) {
	body := streamBody(1024)

	app := newTestApp()
	app.Get("/content", func(call *govalin.Call) {
		call.ServeContent("payload.bin", time.Unix(0, 0), bytes.NewReader(body))
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		response := client.GetRange("/content", 5000, 6000)
		_ = readBody(t, response)

		assert.Equal(
			t,
			http.StatusRequestedRangeNotSatisfiable,
			response.StatusCode,
			"A range beyond the content should be refused",
		)
	})
}

func TestServeContentHeadHasHeadersButNoBody(t *testing.T) {
	body := streamBody(2048)

	app := newTestApp()
	app.Head("/content", func(call *govalin.Call) {
		call.ServeContent("payload.bin", time.Unix(0, 0), bytes.NewReader(body))
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		response := client.HeadResponse("/content")

		assert.Equal(t, http.StatusOK, response.StatusCode, "A HEAD should answer 200")
		assert.Equal(
			t,
			"2048",
			response.Header.Get("Content-Length"),
			"A HEAD should report the length of the body it would have sent",
		)
		assert.Equal(t, "", readBody(t, response), "A HEAD carries no body")
	})
}

// TestServeContentAtServesRangesWithoutSeeking is the remote-object case: an
// object store hands out ranged reads, never a seekable handle, and must not
// have to be buffered to be served.
func TestServeContentAtServesRangesWithoutSeeking(t *testing.T) {
	body := streamBody(8 << 10)

	app := newTestApp()
	app.Get("/object", func(call *govalin.Call) {
		call.ServeContentAt(
			"object.bin",
			time.Unix(0, 0),
			readerAtOnly{reader: bytes.NewReader(body)},
			int64(len(body)),
		)
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		partial := client.GetRange("/object", 1024, 2047)

		assert.Equal(t, http.StatusPartialContent, partial.StatusCode, "A ranged read gets a 206")
		assert.Equal(
			t,
			fmt.Sprintf("bytes 1024-2047/%d", len(body)),
			partial.Header.Get("Content-Range"),
			"A 206 should describe the range it carries",
		)
		assert.Equal(t, string(body[1024:2048]), readBody(t, partial), "The requested bytes should be served")
	})
}

func TestDownloadSendsAnAttachment(t *testing.T) {
	body := streamBody(512)

	app := newTestApp()
	app.Get("/download", func(call *govalin.Call) {
		call.Download("my report.csv", time.Unix(0, 0), bytes.NewReader(body))
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		response := client.GetResponse("/download")

		assert.Equal(t, http.StatusOK, response.StatusCode, "A download answers 200")
		assert.Contains(
			t,
			response.Header.Get("Content-Disposition"),
			"attachment",
			"A download should be offered as an attachment",
		)
		assert.Contains(
			t,
			response.Header.Get("Content-Disposition"),
			`filename="my report.csv"`,
			"A download should carry the given file name",
		)
		assert.Equal(t, string(body), readBody(t, response), "The whole file should be served")
	})
}

// TestAccessLogRecordsTheWrittenStatus covers the log reporting what actually
// went out: a handler that writes the header itself used to be logged with
// whatever buffered status happened to be set, so a 206 was logged as a 200.
func TestAccessLogRecordsTheWrittenStatus(t *testing.T) {
	var buf syncBuffer
	body := streamBody(4 << 10)

	app := newAccessLogApp(true, func(app *govalin.App) {
		app.Get("/content", func(call *govalin.Call) {
			call.ServeContent("payload.bin", time.Unix(0, 0), bytes.NewReader(body))
		})
		app.HTTPServe("/raw", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusTeapot)
		})
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		defer captureAccessLog(&buf)()

		_ = readBody(t, client.GetRange("/content", 0, 1023))
		client.Get("/raw")
		time.Sleep(accessLogSettleTime)

		assert.Contains(t, buf.String(), "status=206", "A 206 should be logged as a 206")
		assert.Contains(
			t,
			buf.String(),
			"status=418",
			"A status written by a raw handler should be the status logged",
		)
	})
}

// TestRawWriterDoesNotGetANotFoundAppended covers a handler that writes to the
// raw writer without setting a status: the response is written, so the
// lifecycle must not treat the request as unhandled and append a 404 body.
func TestRawWriterDoesNotGetANotFoundAppended(t *testing.T) {
	app := newTestApp()
	app.Get("/raw", func(call *govalin.Call) {
		_, err := (*call.Raw.W).Write([]byte("written by hand"))
		assert.Nil(t, err)
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		response := client.GetResponse("/raw")

		assert.Equal(t, http.StatusOK, response.StatusCode, "An implicit 200 should stand")
		assert.Equal(
			t,
			"written by hand",
			readBody(t, response),
			"A hand-written response should not have a 404 body appended to it",
		)
	})
}

// TestConcurrentStreamsAreIndependent guards the response wrapper against
// shared state between requests.
func TestConcurrentStreamsAreIndependent(t *testing.T) {
	body := streamBody(64 << 10)

	app := newTestApp()
	app.Get("/content", func(call *govalin.Call) {
		call.ServeContent("payload.bin", time.Unix(0, 0), bytes.NewReader(body))
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		var waitGroup sync.WaitGroup
		for i := range 8 {
			waitGroup.Add(1)
			go func() {
				defer waitGroup.Done()
				request, err := http.NewRequest(http.MethodGet, client.Host+"/content", nil)
				if !assert.Nil(t, err) {
					return
				}
				request.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", i*1024, i*1024+1023))
				response, doErr := client.HTTP().Do(request) //nolint:bodyclose // closed below
				if !assert.Nil(t, doErr) {
					return
				}
				defer func() { _ = response.Body.Close() }()
				received, _ := io.ReadAll(response.Body)
				assert.Equal(t, http.StatusPartialContent, response.StatusCode)
				assert.Equal(t, string(body[i*1024:i*1024+1024]), string(received))
			}()
		}
		waitGroup.Wait()
	})
}
