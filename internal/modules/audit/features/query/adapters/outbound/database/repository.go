package database

import (
	"context"
	"database/sql"
	"fmt"

	"microbroker-mqtt-edge/internal/modules/audit/domain"
)

// SQLiteRepository implements the query Repository port using SQLite.
type SQLiteRepository struct {
	db *sql.DB
}

// GetByTopic retrieves all records for a given topic, ordered by insertion.
func (r *SQLiteRepository) GetByTopic(ctx context.Context, topic string) ([]domain.Record, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT client, topic, timezone, timestamp, payload
		 FROM raw_data WHERE topic = ? ORDER BY id ASC`, topic)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrQueryFailed, err)
	}
	defer rows.Close()

	var records []domain.Record
	for rows.Next() {
		var rec domain.Record
		if err := rows.Scan(&rec.ClientID, &rec.Topic, &rec.Timezone, &rec.Timestamp, &rec.Payload); err != nil {
			return nil, fmt.Errorf("%w: scan: %v", domain.ErrQueryFailed, err)
		}
		records = append(records, rec)
	}

	return records, rows.Err()
}

// CountByTopic returns the number of records for a given topic.
func (r *SQLiteRepository) CountByTopic(ctx context.Context, topic string) (int64, error) {
	var count int64
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM raw_data WHERE topic = ?`, topic).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("%w: count: %v", domain.ErrQueryFailed, err)
	}
	return count, nil
}
