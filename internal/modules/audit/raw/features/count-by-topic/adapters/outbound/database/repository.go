package database

import (
	"context"
	"database/sql"
	"fmt"

	"microbroker-mqtt-edge/internal/modules/audit/raw/domain"
)

// SQLiteRepository implements the count-by-topic Repository port.
type SQLiteRepository struct {
	db *sql.DB
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
