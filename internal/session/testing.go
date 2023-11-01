package session

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type StoreImplementation struct {
	Name      string
	StoreFunc func() Store
}

// TestStoreImplementation tests a session store implementation.
//
//nolint:funlen
func TestStoreImplementation(t *testing.T, impl StoreImplementation) {
	t.Run(impl.Name, func(t *testing.T) {
		t.Run("CreatingSessions", func(t *testing.T) {
			store := impl.StoreFunc()
			sessionID, err := store.CreateSession(time.Now().Add(1 * time.Hour).UnixNano())

			assert.NoError(t, err)
			assert.NotEmpty(t, sessionID)

			session, err := store.GetSession(sessionID, time.Now().UnixNano())
			assert.NoError(t, err)
			assert.NotEmpty(t, session.ID)
		})

		t.Run("CreatingSessionsWithExpiredExpiry", func(t *testing.T) {
			store := impl.StoreFunc()
			sessionID, err := store.CreateSession(time.Now().UnixNano())

			assert.NoError(t, err)

			session, err := store.GetSession(sessionID, time.Now().UnixNano())
			assert.Error(t, err)
			assert.Empty(t, session)
		})

		t.Run("CreatingMultipleSessionsAndListing", func(t *testing.T) {
			numOfSessionToCreate := 100

			store := impl.StoreFunc()
			for i := 0; i < 100; i++ {
				_, err := store.CreateSession(time.Now().Add(1 * time.Hour).UnixNano())
				assert.NoError(t, err)
			}

			sessions, err := store.GetSessions(0)
			assert.NoError(t, err)
			assert.Equal(t, numOfSessionToCreate, len(sessions))
		})

		t.Run("Removing session", func(t *testing.T) {
			store := impl.StoreFunc()
			sessionID, err := store.CreateSession(time.Now().Add(1 * time.Hour).UnixNano())

			assert.NoError(t, err)
			assert.NotEmpty(t, sessionID)

			session, err := store.GetSession(sessionID, time.Now().UnixNano())
			assert.NoError(t, err)
			assert.NotEmpty(t, session.ID)

			err = store.RemoveSession(sessionID)
			assert.NoError(t, err)

			session, err = store.GetSession(sessionID, time.Now().UnixNano())
			assert.Error(t, err)
			assert.Empty(t, session)
		})

		t.Run("CreatingSessionData", func(t *testing.T) {
			store := impl.StoreFunc()
			sessionID, err := store.CreateSession(time.Now().Add(1 * time.Hour).UnixNano())

			assert.NoError(t, err)
			assert.NotEmpty(t, sessionID)

			sess, err := store.GetSession(sessionID, time.Now().UnixNano())
			assert.NoError(t, err)
			assert.Equal(t, Data{}, sess.Data)
		})

		t.Run("UpdatingSessionData", func(t *testing.T) {
			store := impl.StoreFunc()
			sessionID, err := store.CreateSession(time.Now().Add(1 * time.Hour).UnixNano())

			assert.NoError(t, err)
			assert.NotEmpty(t, sessionID)

			sess, err := store.GetSession(sessionID, time.Now().UnixNano())
			assert.NoError(t, err)
			assert.Equal(t, Data{}, sess.Data)

			sess.Data["test"] = "test"
			err = store.SetSessionData(sessionID, sess.Data)
			assert.NoError(t, err)

			sess, err = store.GetSession(sessionID, time.Now().UnixNano())
			assert.NoError(t, err)
			assert.Equal(t, "test", sess.Data["test"])
		})

		t.Run("RemovingSessionData", func(t *testing.T) {
			store := impl.StoreFunc()
			sessionID, err := store.CreateSession(time.Now().Add(1 * time.Hour).UnixNano())

			assert.NoError(t, err)
			assert.NotEmpty(t, sessionID)

			sess, err := store.GetSession(sessionID, time.Now().UnixNano())
			assert.NoError(t, err)
			assert.Equal(t, Data{}, sess.Data)

			sess.Data["test"] = "test"
			err = store.SetSessionData(sessionID, sess.Data)
			assert.NoError(t, err)

			sess, err = store.GetSession(sessionID, time.Now().UnixNano())
			assert.NoError(t, err)
			assert.Equal(t, "test", sess.Data["test"])

			err = store.RemoveSessionData(sessionID)
			assert.NoError(t, err)

			sess, err = store.GetSession(sessionID, time.Now().UnixNano())
			assert.NoError(t, err)
			assert.Equal(t, Data{}, sess.Data)
		})
	})
}
