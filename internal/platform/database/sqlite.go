package database

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// NewSQLiteConnection opens a SQLite database connection with sensible defaults.
// Use ":memory:" for in-memory databases (testing).
// The returned *sql.DB is safe for concurrent use and should be shared across the application.
func NewSQLiteConnection(dbPath string) (*sql.DB, error) {
	dsn := dbPath
	if dbPath != ":memory:" {
		dsn = dbPath + "?_journal_mode=WAL&_busy_timeout=5000&_synchronous=NORMAL"
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening sqlite at %s: %w", dbPath, err)
	}

	// SQLite does not support multiple concurrent writers.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("pinging sqlite: %w", err)
	}

	return db, nil
}
