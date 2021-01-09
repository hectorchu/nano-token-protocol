package main

import (
	"database/sql"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

var dbLock sync.Mutex

func withDB(cb func(*sql.DB) error) (err error) {
	dbLock.Lock()
	defer dbLock.Unlock()
	db, err := sql.Open("sqlite3", "./chains.db")
	if err != nil {
		return
	}
	defer db.Close()
	return cb(db)
}
