package database_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	querydb "microbroker-mqtt-edge/internal/modules/oee/availability/features/query/adapters/outbound/database"
	platformdb "microbroker-mqtt-edge/internal/platform/database"
)

func newTestRepo(t *testing.T) (*querydb.IntervalRepository, *sql.DB) {
	t.Helper()
	db, err := platformdb.NewSQLiteConnection(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	require.NoError(t, platformdb.NewMigrator(db).Run(context.Background()))
	return querydb.NewIntervalRepository(db), db
}

func seedInterval(t *testing.T, db *sql.DB, machineID, state string, startedAt time.Time, endedAt *time.Time) {
	t.Helper()
	if endedAt == nil {
		_, err := db.ExecContext(context.Background(),
			`INSERT INTO state_intervals (machine_id, state, is_downtime, is_planned_stop, started_at) VALUES (?, ?, 0, 0, ?)`,
			machineID, state, startedAt.UTC().Format(time.RFC3339Nano))
		require.NoError(t, err)
	} else {
		_, err := db.ExecContext(context.Background(),
			`INSERT INTO state_intervals (machine_id, state, is_downtime, is_planned_stop, started_at, ended_at) VALUES (?, ?, 0, 0, ?, ?)`,
			machineID, state, startedAt.UTC().Format(time.RFC3339Nano), endedAt.UTC().Format(time.RFC3339Nano))
		require.NoError(t, err)
	}
}

func ts(hour, min int) time.Time {
	return time.Date(2026, 6, 1, hour, min, 0, 0, time.UTC)
}

func ptr(t time.Time) *time.Time { return &t }

func TestIntervalRepository_ByMachineRange_ReturnsOverlappingClosedIntervals(t *testing.T) {
	repo, db := newTestRepo(t)

	// overlapping: [06:00, 10:00] intersects window [08:00, 12:00]
	seedInterval(t, db, "m1", "stopped", ts(6, 0), ptr(ts(10, 0)))
	// non-overlapping: [04:00, 07:59] ends before window start
	seedInterval(t, db, "m1", "running", ts(4, 0), ptr(ts(7, 59)))

	from, to := ts(8, 0), ts(12, 0)
	got, err := repo.ByMachineRange(context.Background(), "m1", from, to)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "m1", got[0].MachineID)
	assert.Equal(t, ts(6, 0), got[0].StartedAt)
	assert.NotNil(t, got[0].EndedAt)
}

func TestIntervalRepository_ByMachineRange_ExcludesNonOverlappingIntervals(t *testing.T) {
	repo, db := newTestRepo(t)

	from, to := ts(8, 0), ts(12, 0)

	// ended exactly at from — not overlapping (ended_at > from is false)
	seedInterval(t, db, "m1", "running", ts(4, 0), ptr(from))
	// started exactly at to — not overlapping (started_at < to is false)
	seedInterval(t, db, "m1", "running", to, ptr(ts(14, 0)))

	got, err := repo.ByMachineRange(context.Background(), "m1", from, to)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestIntervalRepository_ByMachineRange_IncludesOpenInterval(t *testing.T) {
	repo, db := newTestRepo(t)

	from, to := ts(8, 0), ts(12, 0)

	// open interval started inside window — included
	seedInterval(t, db, "m1", "stopped", ts(10, 0), nil)
	// open interval started after window end — excluded
	seedInterval(t, db, "m1", "running", ts(13, 0), nil)

	got, err := repo.ByMachineRange(context.Background(), "m1", from, to)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, ts(10, 0), got[0].StartedAt)
	assert.Nil(t, got[0].EndedAt)
}

func TestIntervalRepository_ByMachineRange_ReturnsEmptyWhenNoMatch(t *testing.T) {
	repo, _ := newTestRepo(t)

	got, err := repo.ByMachineRange(context.Background(), "m1", ts(8, 0), ts(12, 0))
	require.NoError(t, err)
	assert.NotNil(t, got)
	assert.Empty(t, got)
}

func TestIntervalRepository_ByMachineRange_OrdersByStartedAtAsc(t *testing.T) {
	repo, db := newTestRepo(t)

	from, to := ts(8, 0), ts(18, 0)

	// Insert out of order
	seedInterval(t, db, "m1", "stopped", ts(14, 0), ptr(ts(15, 0)))
	seedInterval(t, db, "m1", "running", ts(9, 0), ptr(ts(10, 0)))
	seedInterval(t, db, "m1", "idle", ts(11, 0), ptr(ts(12, 0)))

	got, err := repo.ByMachineRange(context.Background(), "m1", from, to)
	require.NoError(t, err)
	require.Len(t, got, 3)
	assert.Equal(t, ts(9, 0), got[0].StartedAt)
	assert.Equal(t, ts(11, 0), got[1].StartedAt)
	assert.Equal(t, ts(14, 0), got[2].StartedAt)
}
