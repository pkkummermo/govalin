package sqlitesession_test

import (
	"testing"

	"github.com/pkkummermo/govalin/internal/session"
	"github.com/pkkummermo/govalin/plugins/session/sqlitesession"
	"github.com/stretchr/testify/assert"
)

func TestSqliteSessionStore(t *testing.T) {
	session.TestStoreImplementation(t, session.StoreImplementation{Name: "Sqlite", StoreFunc: func() session.Store {
		inMemorySqlite, err := sqlitesession.NewSqliteSessionStore(":memory:", true)
		assert.NoError(t, err)

		return inMemorySqlite
	}})
}
