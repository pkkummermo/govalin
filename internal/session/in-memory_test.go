package session_test

import (
	"testing"

	"github.com/pkkummermo/govalin/internal/session"
)

func TestInMemorySessionStore(t *testing.T) {
	session.TestStoreImplementation(t, session.StoreImplementation{Name: "InMemory", StoreFunc: session.NewInMemoryStore})
}
