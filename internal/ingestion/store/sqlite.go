package store

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"microbroker-mqtt-edge/internal/ingestion/domain"

	_ "modernc.org/sqlite"
)

// SQLiteStore implements the ingestion.Store interface using SQLite.
// It serializes writes via a mutex since SQLite only supports one writer at a time.
type SQLiteStore struct {
	db *sql.DB
	mu sync.Mutex
}

// NewSQLiteStore opens a SQLite database at the given path with WAL mode and busy timeout.
// Use ":memory:" for in-memory databases (testing).
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
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
	db.SetConnMaxLifetime(0) // keep connection alive forever

	// Verify connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("pinging sqlite: %w", err)
	}

	// Enable WAL mode for in-memory too (no-op but consistent)
	if dbPath == ":memory:" {
		if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
			// WAL may not work with :memory:, that's fine
			_ = err
		}
	}

	return &SQLiteStore{db: db}, nil
}

// Migrate creates or updates the database schema using embedded SQL migration files.
// Migrations are versioned and tracked — safe to call on every startup.
func (s *SQLiteStore) Migrate(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	migrator := NewMigrator(s.db)
	return migrator.Run(ctx)
}

// SaveRawData persists a single message in the raw_data table within a transaction.
// Thread-safe: serializes concurrent calls via mutex.
func (s *SQLiteStore) SaveRawData(ctx context.Context, msg domain.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%w: begin tx: %v", domain.ErrStoreFailure, err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx,
		`INSERT INTO raw_data (client, topic, timezone, timestamp, payload)
		 VALUES (?, ?, ?, ?, ?)`,
		msg.ClientID,
		msg.Topic,
		msg.Timezone,
		msg.Timestamp.Format(time.RFC3339Nano),
		string(msg.Payload),
	)
	if err != nil {
		return fmt.Errorf("%w: insert: %v", domain.ErrStoreFailure, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%w: commit: %v", domain.ErrStoreFailure, err)
	}

	return nil
}

// GetByTopic retrieves all messages for a given topic, ordered by insertion (id ASC).
func (s *SQLiteStore) GetByTopic(ctx context.Context, topic string) ([]domain.Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows, err := s.db.QueryContext(ctx,
		`SELECT client, topic, timezone, timestamp, payload
		 FROM raw_data WHERE topic = ? ORDER BY id ASC`, topic)
	if err != nil {
		return nil, fmt.Errorf("querying by topic: %w", err)
	}
	defer rows.Close()

	var messages []domain.Message
	for rows.Next() {
		var msg domain.Message
		var tsStr string
		if err := rows.Scan(&msg.ClientID, &msg.Topic, &msg.Timezone, &tsStr, &msg.Payload); err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}
		msg.Timestamp, err = time.Parse(time.RFC3339Nano, tsStr)
		if err != nil {
			return nil, fmt.Errorf("parsing timestamp: %w", err)
		}
		messages = append(messages, msg)
	}

	return messages, rows.Err()
}

// Close closes the underlying database connection.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}
