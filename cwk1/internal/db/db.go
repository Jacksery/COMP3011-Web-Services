package db

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
)

func OpenDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	// verify connection is usable
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}
