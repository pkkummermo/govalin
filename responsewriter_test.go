package govalin

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestResponseWriterTracksCommitment pins the wrapper to net/http's own rules
// for when a response is committed. Every divergence is a double-written status
// or a wrong access log, so the cases are spelled out rather than inferred.
func TestResponseWriterTracksCommitment(t *testing.T) {
	tests := []struct {
		name      string
		act       func(writer *responseWriter)
		committed bool
		status    int
	}{
		{
			// net/http may send several 1xx responses before the real status.
			name:      "an informational status does not commit",
			act:       func(writer *responseWriter) { writer.WriteHeader(http.StatusEarlyHints) },
			committed: false,
			status:    0,
		},
		{
			// net/http takes the non-informational path for 101: nothing follows a
			// protocol switch, so it is the final status.
			name:      "switching protocols commits",
			act:       func(writer *responseWriter) { writer.WriteHeader(http.StatusSwitchingProtocols) },
			committed: true,
			status:    http.StatusSwitchingProtocols,
		},
		{
			name:      "a final status commits",
			act:       func(writer *responseWriter) { writer.WriteHeader(http.StatusPartialContent) },
			committed: true,
			status:    http.StatusPartialContent,
		},
		{
			name:      "a body write commits an implicit 200",
			act:       func(writer *responseWriter) { _, _ = writer.Write([]byte("body")) },
			committed: true,
			status:    http.StatusOK,
		},
		{
			// net/http headers a zero-byte write the same as any other.
			name:      "an empty write commits",
			act:       func(writer *responseWriter) { _, _ = writer.Write(nil) },
			committed: true,
			status:    http.StatusOK,
		},
		{
			// net/http's FlushError writes a 200 header when none was sent.
			name:      "a flush commits an implicit 200",
			act:       func(writer *responseWriter) { writer.Flush() },
			committed: true,
			status:    http.StatusOK,
		},
		{
			name:      "a ReadFrom commits an implicit 200",
			act:       func(writer *responseWriter) { _, _ = writer.ReadFrom(strings.NewReader("body")) },
			committed: true,
			status:    http.StatusOK,
		},
		{
			name: "a second status does not overwrite the first",
			act: func(writer *responseWriter) {
				writer.WriteHeader(http.StatusCreated)
				writer.WriteHeader(http.StatusInternalServerError)
			},
			committed: true,
			status:    http.StatusCreated,
		},
		{
			name: "a status after a body write does not overwrite the implicit 200",
			act: func(writer *responseWriter) {
				_, _ = writer.Write([]byte("body"))
				writer.WriteHeader(http.StatusInternalServerError)
			},
			committed: true,
			status:    http.StatusOK,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			writer := &responseWriter{ResponseWriter: httptest.NewRecorder()}

			test.act(writer)

			if writer.committed != test.committed {
				t.Errorf("committed is %v, expected %v", writer.committed, test.committed)
			}
			if writer.status != test.status {
				t.Errorf("written status is %d, expected %d", writer.status, test.status)
			}
		})
	}
}
