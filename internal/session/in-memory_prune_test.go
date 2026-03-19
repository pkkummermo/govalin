package session

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// newStoreForTesting creates an inMemoryStore without spawning the background
// prune goroutine, making it safe and deterministic for unit tests.
func newStoreForTesting() *inMemoryStore {
	return &inMemoryStore{
		mutex: &sync.Mutex{},
		store: make(map[string]Session),
	}
}

func TestSessionPruneNoDeadlock(t *testing.T) {
	store := newStoreForTesting()

	// Insert an already-expired session directly into the store map.
	expiredTime := time.Now().Add(-1 * time.Hour).UnixNano()
	store.store["expired-session"] = Session{
		ID:      "expired-session",
		Expires: expiredTime,
		Data:    Data{},
	}

	// sessionPrune must complete without deadlocking.
	done := make(chan error, 1)
	go func() {
		done <- store.sessionPrune()
	}()

	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("sessionPrune deadlocked")
	}

	// The expired session must have been removed.
	assert.Empty(t, store.store)
}

func TestSessionPruneKeepsValidSessions(t *testing.T) {
	store := newStoreForTesting()

	validTime := time.Now().Add(1 * time.Hour).UnixNano()
	store.store["valid-session"] = Session{
		ID:      "valid-session",
		Expires: validTime,
		Data:    Data{},
	}

	err := store.sessionPrune()
	assert.NoError(t, err)

	// The valid session must still be present.
	assert.Len(t, store.store, 1)
}
