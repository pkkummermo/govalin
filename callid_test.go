package govalin

import (
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// callIDsPerBuffer is how many IDs one buffer of randomness covers. Tests that
// want to cross a refill boundary ask for more than this.
const callIDsPerBuffer = callIDRandomSize / 16

func TestNewCallIDIsAUUIDv4(t *testing.T) {
	parsed, err := uuid.Parse(newCallID())

	require.NoError(t, err, "A call ID should parse as a UUID")
	assert.Equal(t, uuid.Version(4), parsed.Version(), "A call ID should be a version 4 UUID")
	assert.Equal(t, uuid.RFC4122, parsed.Variant(), "A call ID should carry the RFC 4122 variant")
}

// TestNewCallIDIsUniqueAcrossRefills asks for several buffers' worth, because a
// buffer that refills at the wrong offset would hand the same bytes out twice.
func TestNewCallIDIsUniqueAcrossRefills(t *testing.T) {
	wanted := callIDsPerBuffer * 4

	seen := make(map[string]bool, wanted)
	for range wanted {
		seen[newCallID()] = true
	}

	assert.Len(t, seen, wanted, "Every call ID should be distinct across buffer refills")
}

// TestNewCallIDIsUniqueUnderConcurrency is the check the pool exists for: two
// goroutines must never be reading from the same buffer. Run under -race it also
// catches the sharing itself, not only its visible result.
func TestNewCallIDIsUniqueUnderConcurrency(t *testing.T) {
	const goroutines = 8

	perGoroutine := callIDsPerBuffer * 2

	var waitGroup sync.WaitGroup

	generated := make([][]string, goroutines)

	for goroutine := range goroutines {
		waitGroup.Add(1)

		go func() {
			defer waitGroup.Done()

			ids := make([]string, 0, perGoroutine)
			for range perGoroutine {
				ids = append(ids, newCallID())
			}

			generated[goroutine] = ids
		}()
	}

	waitGroup.Wait()

	seen := map[string]bool{}
	for _, ids := range generated {
		for _, id := range ids {
			seen[id] = true
		}
	}

	assert.Len(t, seen, goroutines*perGoroutine, "Concurrent call IDs should all be distinct")
}
