package db

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

func OpenDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	// verify connection is usable
	if err := db.Ping(); err != nil {
		if cerr := db.Close(); cerr != nil {
			// return a wrapped error so callers can see both ping and close results
			return nil, fmt.Errorf("ping error: %w; close error: %v", err, cerr)
		}
		return nil, err
	}
	return db, nil
}
