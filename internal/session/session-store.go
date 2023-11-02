package session

import (
	"crypto/rand"
	"database/sql/driver"
	"encoding/base64"
	"errors"
	"io"

	"log/slog"

	"github.com/pkkummermo/govalin/internal/util"
)

// Data is the stored session data.
type Data map[string]interface{}

// Scan implements the sql.Scanner interface.
func (p *Data) Scan(src interface{}) error {
	return util.ScanJSON(p, src)
}

// Value implements the driver.Valuer interface.
func (p *Data) Value() (driver.Value, error) {
	return util.ValueJSON(p)
}

// Session is the stored session data.
type Session struct {
	ID      string
	Expires int64
	Data    Data
}

// Store is the interface for the session back-end store.
type Store interface {
	// CreateSession creates a new session in the store. The expires parameter is the expire
	// time in nanonseconds for the session. Returns the session ID.
	CreateSession(expires int64) (string, error)

	// GetSession returns the session from the session store based
	// on given ID and where expire is higher than provided nanoseconds.
	GetSession(sessionID string, ingnoreOlderNs int64) (Session, error)

	// RemoveSession removes the session from the store.
	RemoveSession(sessionID string) error

	// GetSessions returns sessions with expire time is less than given time in nanoseconds.
	GetSessions(time int64) ([]Session, error)

	// RefreshSession refreshes a session expire time by adding the given nanoseconds.
	RefreshSession(sessionID string, expireAddNs int64) error

	// SetSessionData sets the session data in the store.
	SetSessionData(sessionID string, data Data) error

	// GetSessionData returns the session data from the store.
	GetSessionData(sessionID string) (Data, error)

	// RemoveSessionData removes the session data from the store.
	RemoveSessionData(sessionID string) error
}

// CreateNewSessionID generates a random session id. Also
// check for collisions so we don't accidentally assign an existing session id.
func CreateNewSessionID(s Store) (string, error) {
	numCollisionsBeforeWeGiveUp := 5
	for i := 0; i < numCollisionsBeforeWeGiveUp; i++ {
		randSize := 64
		randomBytes := make([]byte, randSize)
		if _, err := io.ReadFull(rand.Reader, randomBytes); err != nil {
			slog.Error("Unable to generate random data for session ID")
			return "", err
		}

		encoded := base64.URLEncoding.EncodeToString(randomBytes)
		if _, err := s.GetSession(encoded, 0); err != nil {
			// Return the session id if we don't find it in the store.
			//nolint: nilerr // If we don't find the session we consider that a success.
			return encoded, nil
		}
	}

	// We end up here if we have given up creating a session id.
	slog.Error("Unable to create a unique session ID")
	return "", errors.New("unable to create a unique session ID")
}
