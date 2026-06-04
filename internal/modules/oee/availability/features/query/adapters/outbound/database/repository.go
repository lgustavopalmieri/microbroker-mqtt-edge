package database

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	avdomain "microbroker-mqtt-edge/internal/modules/oee/availability/domain"
	ooedomain "microbroker-mqtt-edge/internal/modules/oee/domain"
)

// IntervalRepository implements query/application.IntervalReader over SQLite.
// It owns no connection lifecycle — the caller (bootstrap) injects *sql.DB.
type IntervalRepository struct {
	db *sql.DB
	mu sync.Mutex
}

// NewIntervalRepository creates a repository with an injected database connection.
func NewIntervalRepository(db *sql.DB) *IntervalRepository {
	return &IntervalRepository{db: db}
}

// ByMachineRange returns all state intervals for machineID that overlap [from, to).
// An open interval (ended_at IS NULL) is included if it started before to.
// Results are ordered by started_at ASC.
func (r *IntervalRepository) ByMachineRange(ctx context.Context, machineID string, from, to time.Time) ([]avdomain.StateInterval, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	rows, err := r.db.QueryContext(ctx, `
		SELECT state, started_at, ended_at, reason
		FROM state_intervals
		WHERE machine_id = ?
		  AND started_at < ?
		  AND (ended_at IS NULL OR ended_at > ?)
		ORDER BY started_at ASC
	`, machineID, to.UTC().Format(time.RFC3339Nano), from.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return nil, fmt.Errorf("interval reader: query: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	intervals := make([]avdomain.StateInterval, 0)
	for rows.Next() {
		var (
			state     string
			startedAt string
			endedAt   sql.NullString
			reason    string
		)
		if err := rows.Scan(&state, &startedAt, &endedAt, &reason); err != nil {
			return nil, fmt.Errorf("interval reader: scan: %w", err)
		}

		ts, err := time.Parse(time.RFC3339Nano, startedAt)
		if err != nil {
			return nil, fmt.Errorf("interval reader: parse started_at: %w", err)
		}

		var endedAtPtr *time.Time
		if endedAt.Valid {
			t, err := time.Parse(time.RFC3339Nano, endedAt.String)
			if err != nil {
				return nil, fmt.Errorf("interval reader: parse ended_at: %w", err)
			}
			endedAtPtr = &t
		}

		intervals = append(intervals, avdomain.StateInterval{
			MachineID: machineID,
			State:     ooedomain.MachineState(state),
			StartedAt: ts.UTC(),
			EndedAt:   endedAtPtr,
			Reason:    reason,
		})
	}

	return intervals, rows.Err()
}
