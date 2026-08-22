package govalin

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"time"

	"github.com/pkkummermo/govalin/internal/http/headers"
)

// Stream copies the reader into the response body, buffering nothing.
//
// It is the sink for a body too large to hold in memory. The buffered status is
// committed first (200 OK by default), so govalin owns the response and no
// lifecycle bypass is needed. An empty contentType leaves the content type to
// net/http, which sniffs it from the first bytes. The length is unknown and the
// reader cannot seek, so the body is chunked and carries no ranges — use
// ServeContent for content that can seek.
//
// A failure to read the source is returned. A failure to write it out is not:
// the header is long gone, so there is no error left to send and nothing for the
// caller to do. It is logged at debug level and reported as success.
func (call *Call) Stream(contentType string, reader io.Reader) error {
	if contentType != "" {
		call.w.Header().Set(headers.ContentType, contentType)
	}

	call.sendStatusOrDefault()

	// io.Copy cannot say which side failed, and the two sides call for opposite answers.
	source := &sourceReader{reader: reader}

	if _, err := io.Copy(&call.w, source); err != nil {
		if source.err != nil {
			return fmt.Errorf("failed to read the streamed body. %w", source.err)
		}

		slog.Debug("Failed to write the streamed body, the response is already on the wire",
			"err", err, "id", call.ID())
	}

	return nil
}

// sourceReader remembers why the content stopped reading, so a failure of the
// source is not mistaken for a client that hung up.
type sourceReader struct {
	reader io.Reader
	err    error
}

func (source *sourceReader) Read(buffer []byte) (int, error) {
	read, err := source.reader.Read(buffer)
	if err != nil && !errors.Is(err, io.EOF) {
		source.err = err
	}

	return read, err
}

// ServeContent sends seekable content as the response body, with Range support.
//
// It delegates to http.ServeContent, which resolves Range and If-Range and
// answers 206 (including multipart ranges), 416, 304 and HEAD, sets
// Content-Length, and derives the content type from the name's extension unless
// the handler set one. A zero modTime leaves out Last-Modified.
//
// http.ServeContent picks the status itself, so a status buffered with Status is
// not used. The response is committed here and the lifecycle leaves it alone.
func (call *Call) ServeContent(name string, modTime time.Time, content io.ReadSeeker) {
	http.ServeContent(&call.w, call.req, name, modTime, content)

	// After handlers read the buffered status, so it has to match what went out.
	call.status = call.w.status
}

// ServeContentAt is ServeContent for content that reads at an offset but cannot
// seek — a remote object exposing ranged GETs rather than a handle.
//
// The size is the full length of the content, and is what makes a Range request
// resolvable without reading anything ahead of it: a seek becomes offset
// arithmetic that the next ranged read resolves.
func (call *Call) ServeContentAt(name string, modTime time.Time, content io.ReaderAt, size int64) {
	call.ServeContent(name, modTime, io.NewSectionReader(content, 0, size))
}

// Download is ServeContent offered to the client as a file to save, by setting
// Content-Disposition: attachment with the given filename.
func (call *Call) Download(filename string, modTime time.Time, content io.ReadSeeker) {
	call.w.Header().Set(headers.ContentDisposition, attachmentDisposition(filename))
	call.ServeContent(filename, modTime, content)
}

// attachmentDisposition builds the Content-Disposition value for a download.
// mime.FormatMediaType quotes and escapes the name and encodes a non-ASCII one
// per RFC 2231; a name it cannot represent falls back to a bare attachment
// rather than an unparseable header.
func attachmentDisposition(filename string) string {
	disposition := mime.FormatMediaType("attachment", map[string]string{"filename": filename})
	if disposition == "" {
		return "attachment"
	}

	return disposition
}
