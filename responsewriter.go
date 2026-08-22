package govalin

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
)

// responseWriter records the committed response: whether the header has reached
// the wire, and with which written status.
//
// Everything writes through it — a Call body sink, http.ServeContent, an
// HTTPServe handler, third-party middleware — so govalin knows a response has
// been committed without asking callers to declare it.
type responseWriter struct {
	http.ResponseWriter
	status    int
	committed bool
}

// WriteHeader records the status and passes it on unchanged. A repeated call is
// still forwarded, so net/http keeps reporting a genuine double write; the
// framework's own status flush is guarded by the committed flag instead.
func (writer *responseWriter) WriteHeader(status int) {
	// 1xx is informational and the real status still follows; 101 is net/http's own exception.
	if !writer.committed && (status >= http.StatusOK || status == http.StatusSwitchingProtocols) {
		writer.status = status
		writer.committed = true
	}

	writer.ResponseWriter.WriteHeader(status)
}

// Write records the implicit 200 net/http commits for an unheadered body.
func (writer *responseWriter) Write(data []byte) (int, error) {
	writer.markCommitted()

	return writer.ResponseWriter.Write(data)
}

// WriteString keeps net/http's string path: *http.response implements
// io.StringWriter, and without this a string body would be copied into a byte
// slice of its own before every write.
func (writer *responseWriter) WriteString(data string) (int, error) {
	writer.markCommitted()

	if stringWriter, ok := writer.ResponseWriter.(io.StringWriter); ok {
		return stringWriter.WriteString(data)
	}

	return writer.ResponseWriter.Write([]byte(data))
}

// ReadFrom keeps net/http's sendfile path: io.Copy prefers an io.ReaderFrom, and
// without this every streamed body would fall back to a buffered copy.
func (writer *responseWriter) ReadFrom(reader io.Reader) (int64, error) {
	writer.markCommitted()

	if readerFrom, ok := writer.ResponseWriter.(io.ReaderFrom); ok {
		return readerFrom.ReadFrom(reader)
	}

	return io.Copy(writer.ResponseWriter, reader)
}

// Flush pushes buffered bytes to the client. net/http headers a response that
// is flushed before anything was written, so a flush commits it.
func (writer *responseWriter) Flush() {
	flusher, ok := writer.ResponseWriter.(http.Flusher)
	if !ok {
		return
	}

	writer.markCommitted()
	flusher.Flush()
}

// Hijack hands over the connection — a websocket upgrade type-asserts on
// http.Hijacker directly, so the wrapper has to carry it. What goes onto the
// connection from here is no longer net/http's to header, so the response counts
// as committed.
func (writer *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := writer.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("the underlying response writer does not support hijacking")
	}

	conn, readWriter, err := hijacker.Hijack()
	if err == nil {
		writer.committed = true
	}

	return conn, readWriter, err
}

// Unwrap exposes the wrapped writer to http.ResponseController, which is how a
// handler reaches capabilities the wrapper does not implement itself.
func (writer *responseWriter) Unwrap() http.ResponseWriter {
	return writer.ResponseWriter
}

// markCommitted records the implicit 200 that a body write commits. net/http
// headers a zero-byte write the same way, so no length check is needed.
func (writer *responseWriter) markCommitted() {
	if writer.committed {
		return
	}

	writer.status = http.StatusOK
	writer.committed = true
}
