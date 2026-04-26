package database

import "database/sql"

// NewSQLiteRepository creates a Repository backed by the given database connection.
func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}
