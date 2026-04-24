package database

import "database/sql"

// NewSQLiteRepository creates a Repository backed by the given database connection.
// The caller (bootstrap) is responsible for the connection lifecycle.
func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}
