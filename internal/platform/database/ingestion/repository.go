package ingestion

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"microbroker-mqtt-edge/internal/common/message"
)

// SQLiteRepository is the outbound adapter that implements ingestion.Store
// using a shared SQLite database connection. The *sql.DB is injected from
// the platform layer — this adapter does NOT own the connection lifecycle.
type SQLiteRepository struct {
	db *sql.DB
	mu sync.Mutex
}

// NewSQLiteRepository creates a repository with an injected database connection.
// The caller (bootstrap) is responsible for opening, migrating, and closing the connection.
func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

// SaveRawData persists a single message in the raw_data table within a transaction.
// Thread-safe: serializes concurrent calls via mutex (SQLite single-writer constraint).
func (r *SQLiteRepository) SaveRawData(ctx context.Context, msg message.Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%w: begin tx: %v", ErrStoreFailure, err)
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
		return fmt.Errorf("%w: insert: %v", ErrStoreFailure, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%w: commit: %v", ErrStoreFailure, err)
	}

	return nil
}

// GetByTopic retrieves all messages for a given topic, ordered by insertion (id ASC).
func (r *SQLiteRepository) GetByTopic(ctx context.Context, topic string) ([]message.Message, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	rows, err := r.db.QueryContext(ctx,
		`SELECT client, topic, timezone, timestamp, payload
		 FROM raw_data WHERE topic = ? ORDER BY id ASC`, topic)
	if err != nil {
		return nil, fmt.Errorf("querying by topic: %w", err)
	}
	defer rows.Close()

	var messages []message.Message
	for rows.Next() {
		var msg message.Message
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

// Close is a no-op — the database connection lifecycle is managed by the platform layer.
func (r *SQLiteRepository) Close() error {
	return nil
}
