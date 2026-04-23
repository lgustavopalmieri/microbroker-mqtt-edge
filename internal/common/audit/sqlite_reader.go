package audit

import (
	"context"
	"database/sql"
	"fmt"
)

// SQLiteReader implements Reader using a shared SQLite connection.
type SQLiteReader struct {
	db *sql.DB
}

// NewSQLiteReader creates a Reader backed by the given database connection.
func NewSQLiteReader(db *sql.DB) *SQLiteReader {
	return &SQLiteReader{db: db}
}

// GetByTopic retrieves all records for a given topic, ordered by insertion.
func (r *SQLiteReader) GetByTopic(ctx context.Context, topic string) ([]Record, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT client, topic, timezone, timestamp, payload
		 FROM raw_data WHERE topic = ? ORDER BY id ASC`, topic)
	if err != nil {
		return nil, fmt.Errorf("audit query: %w", err)
	}
	defer rows.Close()

	var records []Record
	for rows.Next() {
		var rec Record
		if err := rows.Scan(&rec.ClientID, &rec.Topic, &rec.Timezone, &rec.Timestamp, &rec.Payload); err != nil {
			return nil, fmt.Errorf("audit scan: %w", err)
		}
		records = append(records, rec)
	}

	return records, rows.Err()
}

// CountByTopic returns the number of records for a given topic.
func (r *SQLiteReader) CountByTopic(ctx context.Context, topic string) (int64, error) {
	var count int64
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM raw_data WHERE topic = ?`, topic).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("audit count: %w", err)
	}
	return count, nil
}
