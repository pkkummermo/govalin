package internal

import (
	"database/sql"
)

const (
	CreateGovalinSessionTable = `
	CREATE TABLE IF NOT EXISTS govalin_session (
		session_id     VARCHAR(128)  NOT NULL,
		expires        BIGINT        NOT NULL,
		data           JSON          NOT NULL,
		CONSTRAINT govaline_session_pk PRIMARY KEY (session_id)
	)`
	CreateGovalinSessionTableIndex = `CREATE INDEX IF NOT EXISTS oauthsession_expires ON govalin_session(expires)`
)

type Statements struct {
	CreateSession   *sql.Stmt
	RetrieveSession *sql.Stmt
	RemoveSession   *sql.Stmt
	ListSessions    *sql.Stmt
	RefreshSession  *sql.Stmt
	GetSessionData  *sql.Stmt
	SetSessionData  *sql.Stmt
}

func InitStatements(db *sql.DB) (Statements, error) {
	var err error
	var sqlStatements Statements

	if sqlStatements.CreateSession, err = db.Prepare(`
		INSERT INTO govalin_session (session_id, expires, data)
			VALUES ($1, $2, $3)
	`); err != nil {
		return sqlStatements, err
	}

	if sqlStatements.RetrieveSession, err = db.Prepare(`
		SELECT session_id, expires, data
			FROM govalin_session
			WHERE session_id = $1 AND expires > $2`); err != nil {
		return sqlStatements, err
	}
	if sqlStatements.RemoveSession, err = db.Prepare(`
		DELETE FROM govalin_session
			WHERE session_id = $1`); err != nil {
		return sqlStatements, err
	}
	if sqlStatements.ListSessions, err = db.Prepare(`
		SELECT session_id, expires, data
			FROM govalin_session WHERE expires > $1`); err != nil {
		return sqlStatements, err
	}
	if sqlStatements.RefreshSession, err = db.Prepare(`
		UPDATE govalin_session
			SET expires = expires + $1
			WHERE session_id = $2`); err != nil {
		return sqlStatements, err
	}
	if sqlStatements.GetSessionData, err = db.Prepare(`
	   SELECT data 
	   		FROM govalin_session WHERE session_id = $1`); err != nil {
		return sqlStatements, err
	}
	if sqlStatements.SetSessionData, err = db.Prepare(`
		UPDATE govalin_session
			SET data = $1
			WHERE session_id = $2`); err != nil {
		return sqlStatements, err
	}

	return sqlStatements, nil
}
