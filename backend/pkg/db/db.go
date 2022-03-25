package db

import (
	"database/sql"

	_ "github.com/lib/pq"
)

// DB contains client database pool
type DB struct {
	Client *sql.DB
}

// Get opens a sql connection to postgres and returns DB instance
func Get(connStr string) (*DB, error) {
	db, err := get(connStr)
	if err != nil {
		return nil, err
	}

	return &DB{
		Client: db,
	}, nil
}

// Close calls sql.DB Close function. Returns an error on failure
func (db *DB) Close() error {
	return db.Client.Close()
}

func get(connStr string) (*sql.DB, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}
