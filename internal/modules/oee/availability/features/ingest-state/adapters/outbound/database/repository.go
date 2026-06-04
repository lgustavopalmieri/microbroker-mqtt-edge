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

// IntervalRepository implements application.IntervalStore over SQLite.
// It owns no connection lifecycle — the caller (bootstrap) injects *sql.DB.
type IntervalRepository struct {
	db *sql.DB
	mu sync.Mutex
}

// NewIntervalRepository creates a repository with an injected database connection.
func NewIntervalRepository(db *sql.DB) *IntervalRepository {
	return &IntervalRepository{db: db}
}

// OpenInterval inserts a new state interval with ended_at = NULL.
func (r *IntervalRepository) OpenInterval(ctx context.Context, machineID string, state ooedomain.MachineState, startedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("interval repo: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	_, err = tx.ExecContext(ctx, `
		INSERT INTO state_intervals (machine_id, state, is_downtime, is_planned_stop, started_at)
		VALUES (?, ?, ?, ?, ?)
	`, machineID, string(state), boolToInt(state.IsDowntime()), boolToInt(state.IsPlannedStop()), startedAt.Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("interval repo: insert: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("interval repo: commit: %w", err)
	}
	return nil
}

// CloseOpen sets ended_at on the currently open interval for the given machine.
// It is a no-op if no open interval exists.
func (r *IntervalRepository) CloseOpen(ctx context.Context, machineID string, endedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("interval repo: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	_, err = tx.ExecContext(ctx,
		`UPDATE state_intervals SET ended_at = ? WHERE machine_id = ? AND ended_at IS NULL`,
		endedAt.Format(time.RFC3339Nano), machineID)
	if err != nil {
		return fmt.Errorf("interval repo: update: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("interval repo: commit: %w", err)
	}
	return nil
}

// LastOpen returns the open interval (ended_at IS NULL) for the given machine.
// Returns found=false when none exists.
func (r *IntervalRepository) LastOpen(ctx context.Context, machineID string) (avdomain.StateInterval, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var (
		stateStr  string
		startedAt string
		reason    string
	)

	err := r.db.QueryRowContext(ctx, `
		SELECT state, started_at, reason
		FROM state_intervals
		WHERE machine_id = ? AND ended_at IS NULL
		ORDER BY started_at DESC
		LIMIT 1
	`, machineID).Scan(&stateStr, &startedAt, &reason)

	if err == sql.ErrNoRows {
		return avdomain.StateInterval{}, false, nil
	}
	if err != nil {
		return avdomain.StateInterval{}, false, fmt.Errorf("interval repo: query: %w", err)
	}

	ts, err := time.Parse(time.RFC3339Nano, startedAt)
	if err != nil {
		return avdomain.StateInterval{}, false, fmt.Errorf("interval repo: parse timestamp: %w", err)
	}

	return avdomain.StateInterval{
		MachineID: machineID,
		State:     ooedomain.MachineState(stateStr),
		StartedAt: ts,
		EndedAt:   nil,
		Reason:    reason,
	}, true, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
