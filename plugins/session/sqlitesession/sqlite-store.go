package sqlitesession

import (
	"database/sql"
	"errors"
	"sync"

	// SQLite3 driver.
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkkummermo/govalin/internal/session"
	"github.com/pkkummermo/govalin/plugins/session/internal"
)

type sqliteSessionStore struct {
	mutex *sync.Mutex
	db    *sql.DB
	s     internal.Statements
}

const sqliteDriver = "sqlite3"

func NewSqliteSessionStore(connectionString string, useWAL bool) (*sqliteSessionStore, error) {
	var err error

	ret := sqliteSessionStore{
		mutex: &sync.Mutex{},
	}

	if ret.db, err = sql.Open(sqliteDriver, connectionString); err != nil {
		return &ret, err
	}

	if pingErr := ret.db.Ping(); pingErr != nil {
		return &ret, pingErr
	}

	if useWAL {
		if _, pragmaErr := ret.db.Exec("PRAGMA journal_mode=WAL;"); pragmaErr != nil {
			return &ret, pragmaErr
		}
	}

	if createSchemaErr := ret.createSchema(); createSchemaErr != nil {
		return &ret, createSchemaErr
	}

	if initErr := ret.init(); initErr != nil {
		return &ret, initErr
	}

	return &ret, nil
}

func (s *sqliteSessionStore) createSchema() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	_, err := s.db.Exec(internal.CreateGovalinSessionTable)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(internal.CreateGovalinSessionTableIndex)
	if err != nil {
		return err
	}

	return nil
}

func (s *sqliteSessionStore) init() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	initiatedStatements, err := internal.InitStatements(s.db)

	if err != nil {
		return err
	}

	s.s = initiatedStatements

	return nil
}

func (s *sqliteSessionStore) CreateSession(expires int64) (string, error) {
	sessionID, err := session.CreateNewSessionID(s)

	if err != nil {
		return "", err
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()
	rows, err := s.s.CreateSession.Exec(&sessionID, &expires, &session.Data{})
	if err != nil {
		return "", err
	}
	count, err := rows.RowsAffected()
	if err != nil {
		return "", err
	}
	if count == 0 {
		return "", errors.New("no session created")
	}

	return sessionID, nil
}

func (s *sqliteSessionStore) GetSession(sessionID string, expireTime int64) (session.Session, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	row := s.s.RetrieveSession.QueryRow(sessionID, expireTime)
	if row == nil {
		return session.Session{}, errors.New("not found")
	}
	ret := session.Session{}
	return ret, row.Scan(&ret.ID, &ret.Expires, &ret.Data)
}

func (s *sqliteSessionStore) RemoveSession(sessionID string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	res, err := s.s.RemoveSession.Exec(sessionID)
	if err != nil {
		return err
	}
	count, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return errors.New("session not found")
	}

	return nil
}

func (s *sqliteSessionStore) GetSessions(time int64) ([]session.Session, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	rows, err := s.s.ListSessions.Query(time)
	if err != nil {
		return nil, err
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	defer rows.Close()
	ret := make([]session.Session, 0)
	for rows.Next() {
		sess := session.Session{}
		if scanErr := rows.Scan(&sess.ID, &sess.Expires, &sess.Data); scanErr != nil {
			return ret, scanErr
		}
		ret = append(ret, sess)
	}

	return ret, nil
}

func (s *sqliteSessionStore) RefreshSession(sessionID string, checkInterval int64) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	res, err := s.s.RefreshSession.Exec(checkInterval, sessionID)
	if err != nil {
		return err
	}
	count, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return errors.New("session not found")
	}

	return nil
}

func (s *sqliteSessionStore) SetSessionData(sessionID string, data session.Data) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	res, err := s.s.SetSessionData.Exec(&data, sessionID)
	if err != nil {
		return err
	}
	count, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return errors.New("session not found")
	}

	return nil
}

func (s *sqliteSessionStore) GetSessionData(sessionID string) (session.Data, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	row := s.s.GetSessionData.QueryRow(sessionID)
	if row == nil {
		return nil, errors.New("not found")
	}
	var data session.Data
	return data, row.Scan(&data)
}

func (s *sqliteSessionStore) RemoveSessionData(sessionID string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	res, err := s.s.SetSessionData.Exec(&session.Data{}, sessionID)
	if err != nil {
		return err
	}
	count, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return errors.New("session not found")
	}

	return nil
}
