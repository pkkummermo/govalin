<p align="center">
  <img src="docs/govalin-gopher.svg" alt="A Go gopher hugging the Javalin mark" width="180">
</p>

# govalin

[![Unit tests](https://github.com/pkkummermo/govalin/actions/workflows/main.yml/badge.svg)](https://github.com/pkkummermo/govalin/actions/workflows/main.yml)

A simple way of creating efficient HTTP APIs in golang using conventions over configuration.

## Installation

To install govalin run:

```bash
go get -u github.com/pkkummermo/govalin
```

## Hello World

```go
func main() {
	govalin.New().
		Get("/test", func(call *govalin.Call) {
			call.Text("Hello world")
		}).
		Start(7070)
}
```

## Serving files and large bodies

`Text`, `HTML` and `JSON` are for bodies that fit in memory. For anything bigger, `Call` streams:

```go
govalin.New().
	// Any reader, of any length: nothing is buffered.
	Get("/logs", func(call *govalin.Call) {
		if err := call.Stream("text/plain", logReader()); err != nil {
			call.Error(err)
		}
	}).
	// Seekable content, with Range support: resumable downloads and seeking.
	Get("/files/{name}", func(call *govalin.Call) {
		file, err := os.Open(call.PathParam("name"))
		if err != nil {
			call.Error(err)
			return
		}
		defer file.Close()

		info, _ := file.Stat()
		call.ServeContent(info.Name(), info.ModTime(), file)
	}).
	Start(7070)
```

- `Stream(contentType, reader)` copies a reader onto the response. Unknown length, no ranges.
- `ServeContent(name, modTime, readSeeker)` handles `Range`, `If-Range`, `206`, `416`, `304`, `HEAD`
  and `Content-Length` for you, and picks the content type from the name.
- `ServeContentAt(name, modTime, readerAt, size)` is the same for content you can only read at an
  offset — a remote object exposing ranged GETs — so a multi-gigabyte object is served with working
  Range requests and nothing buffered.
- `Download(filename, modTime, readSeeker)` is `ServeContent` offered as an attachment to save.

`Stream` returns a failure to read your content. A failure to write it out — a client that hangs up
mid-body — is not returned: the header is already sent, so there is nothing left to answer with. It
is logged at debug level.

## Testing your app

Govalin ships a test harness, [`govalintest`](govalintest/), for testing your application over real HTTP. Hand it your own app — built by your own constructor, with your config, plugins and routes — and it starts it on an OS-assigned port, waits for it to be ready, and shuts it down when the test finishes:

```go
import (
	"testing"

	"github.com/pkkummermo/govalin/govalintest"
)

func TestMyAPI(t *testing.T) {
	govalintest.Test(t, myapp.New(), func(client *govalintest.Client) {
		if body := client.Get("/health"); body != "ok" {
			t.Errorf("expected ok, got %q", body)
		}
	})
}
```

A few things to know:

- Failures fail the calling test (`t.Fatalf`), never the test process.
- The body-returning verbs (`Get`, `Post`, ...) return the body regardless of status code, so you can assert on error responses. Use the `*Response` variants (`GetResponse`, ...) to assert on status codes and headers.
- `Post`, `Put` and `Patch` send `string`, `[]byte` and `io.Reader` bodies as-is; any other value is JSON-encoded with `Content-Type: application/json`.
- For anything custom (headers, auth, exotic verbs), build an `*http.Request` with a relative path and pass it to `client.Do(...)`, or grab the underlying client with `client.HTTP()`.
- `client.Websocket(path)` connects a websocket to your app.
- `client.GetRange(path, from, to)` asks for a byte range, for asserting on `206` responses.
- Harness knobs live in `govalintest.TestWithOptions(t, app, govalintest.Options{...}, fn)` — the zero value always means sane defaults.

## Motivation

I love how fast and efficient go is. What I don't like, is how it doesn't create an easy way of creating HTTP APIs. Govalin focuses on pleasing those who want to create APIs without too much hassle, with a lean simple API.

Inspired by simple libraries and frameworks such as [Javalin](https://javalin.io), I wanted to see if we could port the simplicity to golang.

## Mascot

The govalin gopher is a modified version of the Go gopher, which was designed by
[Renée French](https://reneefrench.blogspot.com/) and is licensed under
[CC BY 3.0](https://creativecommons.org/licenses/by/3.0/). The mark it is hugging is the
[Javalin](https://javalin.io) logo, used with Javalin's permission.
