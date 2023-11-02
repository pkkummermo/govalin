// Package session provides an interface for managing user sessions.
// This file contains an implementation of the Store interface using an in-memory store.
package session

import (
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

const pruneInterval = 1 * time.Minute

// inMemoryStore is an implementation of the Store interface using an in-memory store.
type inMemoryStore struct {
	mutex *sync.Mutex
	store map[string]Session
}

// NewInMemoryStore returns a new instance of inMemoryStore.
func NewInMemoryStore() Store {
	initiatedStore := inMemoryStore{
		mutex: &sync.Mutex{},
		store: make(map[string]Session),
	}

	go func() {
		for {
			time.Sleep(pruneInterval)
			err := initiatedStore.sessionPrune()
			if err != nil {
				slog.Error(fmt.Sprintf("Failed to prune sessions: %v", err))
			}
		}
	}()

	return &initiatedStore
}

func (s *inMemoryStore) sessionPrune() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for sessionID, session := range s.store {
		if session.Expires < time.Now().UnixNano() {
			slog.Debug(fmt.Sprintf("Pruning session: %s", sessionID))
			deleteErr := s.RemoveSession(sessionID)
			if deleteErr != nil {
				return deleteErr
			}
		}
	}
	return nil
}

// CreateSession creates a new session with the given expiration time and returns its ID.
func (s *inMemoryStore) CreateSession(expires int64) (string, error) {
	sessionID, err := CreateNewSessionID(s)

	if err != nil {
		return "", err
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.store[sessionID] = Session{
		ID:      sessionID,
		Expires: expires,
		Data:    Data{},
	}

	return sessionID, nil
}

// GetSession returns the session with the given ID if it exists and has not expired.
func (s *inMemoryStore) GetSession(sessionID string, expireTime int64) (Session, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if session, ok := s.store[sessionID]; ok {
		if session.Expires > expireTime {
			return session, nil
		}
	}

	return Session{}, errors.New("not found")
}

// RemoveSession removes the session with the given ID if it exists.
func (s *inMemoryStore) RemoveSession(sessionID string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if _, ok := s.store[sessionID]; !ok {
		return errors.New("not found")
	}

	delete(s.store, sessionID)
	return nil
}

// GetSessions returns all sessions that have not expired.
func (s *inMemoryStore) GetSessions(time int64) ([]Session, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var sessions = make([]Session, 0)

	for _, session := range s.store {
		if session.Expires > time {
			sessions = append(sessions, session)
		}
	}

	return sessions, nil
}

// RefreshSession extends the expiration time of the session with the given ID.
func (s *inMemoryStore) RefreshSession(sessionID string, checkInterval int64) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if session, ok := s.store[sessionID]; ok {
		session.Expires += checkInterval
		s.store[sessionID] = session

		return nil
	}

	return errors.New("not found")
}

// SetSessionData sets the data associated with the session with the given ID.
func (s *inMemoryStore) SetSessionData(sessionID string, data Data) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if session, ok := s.store[sessionID]; ok {
		session.Data = data
		s.store[sessionID] = session
		return nil
	}

	return errors.New("not found")
}

// GetSessionData returns the data associated with the session with the given ID.
func (s *inMemoryStore) GetSessionData(sessionID string) (Data, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if session, ok := s.store[sessionID]; ok {
		return session.Data, nil
	}

	return Data{}, errors.New("not found")
}

// RemoveSessionData removes the data associated with the session with the given ID.
func (s *inMemoryStore) RemoveSessionData(sessionID string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if session, ok := s.store[sessionID]; ok {
		session.Data = Data{}
		s.store[sessionID] = session

		return nil
	}

	return errors.New("not found")
}
