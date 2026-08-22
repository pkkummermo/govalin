package govalin

import (
	"crypto/rand"
	"sync"

	"github.com/google/uuid"
)

// callIDRandomSize is how much randomness one buffer holds. At sixteen bytes per
// call ID it covers thirty-two requests, so the cost of reading from crypto/rand
// is paid once per refill rather than once per request.
const callIDRandomSize = 512

// callIDRandom is a buffer of randomness handed out sixteen bytes at a time.
// It is not safe for concurrent use; callIDRandomPool is what makes it safe,
// by never lending the same buffer to two goroutines at once.
type callIDRandom struct {
	buffer [callIDRandomSize]byte
	offset int
}

func (source *callIDRandom) fill(destination []byte) {
	if source.offset+len(destination) > len(source.buffer) {
		// Documented never to fail, and to fill the slice entirely, since Go 1.24.
		rand.Read(source.buffer[:])
		source.offset = 0
	}

	source.offset += copy(destination, source.buffer[source.offset:])
}

var callIDRandomPool = sync.Pool{
	New: func() any {
		return &callIDRandom{offset: callIDRandomSize}
	},
}

// newCallID returns the UUIDv4 string that identifies one call.
func newCallID() string {
	source, _ := callIDRandomPool.Get().(*callIDRandom)

	var id uuid.UUID
	source.fill(id[:])

	callIDRandomPool.Put(source)

	// Version 4 and variant 10, which randomness alone does not set (RFC 9562 §5.4).
	id[6] = (id[6] & 0x0f) | 0x40
	id[8] = (id[8] & 0x3f) | 0x80

	return id.String()
}
