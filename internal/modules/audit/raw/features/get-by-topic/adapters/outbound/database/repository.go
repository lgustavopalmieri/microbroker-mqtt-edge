package database

import (
	"context"
	"database/sql"
	"fmt"

	"microbroker-mqtt-edge/internal/modules/audit/raw/domain"
)

// SQLiteRepository implements the get-by-topic Repository port.
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
