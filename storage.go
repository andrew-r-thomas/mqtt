package mqtt

import (
	"database/sql"
)

type Storage struct {
	dbConn      *sql.DB
	getUserStmt *sql.Stmt
}

func NewStorage(path string) (Storage, error) {
	var storage Storage
	var err error
	storage.dbConn, err = sql.Open("sqlite3", path)
	if err != nil {
		return storage, err
	}
	storage.getUserStmt, err = storage.dbConn.Prepare("select * from users where id = ?")
	if err != nil {
		return storage, err
	}

	return storage, nil
}

func (s *Storage) GetUser(clientId string) (User, error) {
	var user User
	var err error

	err = s.getUserStmt.QueryRow(clientId).Scan(&user)
	if err != nil {
		return user, err
	}

	return user, nil
}
