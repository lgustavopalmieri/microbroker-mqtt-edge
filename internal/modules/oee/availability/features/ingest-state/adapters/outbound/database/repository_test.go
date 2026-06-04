package database_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	intervaldb "microbroker-mqtt-edge/internal/modules/oee/availability/features/ingest-state/adapters/outbound/database"
	ooedomain "microbroker-mqtt-edge/internal/modules/oee/domain"
	platformdb "microbroker-mqtt-edge/internal/platform/database"
)

func newTestIntervalRepo(t *testing.T) (*intervaldb.IntervalRepository, func()) {
	t.Helper()

	db, err := platformdb.NewSQLiteConnection(":memory:")
	require.NoError(t, err)

	err = platformdb.NewMigrator(db).Run(context.Background())
	require.NoError(t, err)

	return intervaldb.NewIntervalRepository(db), func() { db.Close() }
}

func TestIntervalRepository_OpenInterval_InsertsOpenRow(t *testing.T) {
	repo, cleanup := newTestIntervalRepo(t)
	defer cleanup()

	startedAt := time.Date(2026, 6, 1, 8, 0, 0, 0, time.UTC)
	err := repo.OpenInterval(context.Background(), "m1", ooedomain.Running, startedAt)
	require.NoError(t, err)

	interval, found, err := repo.LastOpen(context.Background(), "m1")
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "m1", interval.MachineID)
	assert.Equal(t, ooedomain.Running, interval.State)
	assert.Equal(t, startedAt.UTC(), interval.StartedAt.UTC())
	assert.Nil(t, interval.EndedAt)
}

func TestIntervalRepository_CloseOpen_SetsEndedAt(t *testing.T) {
	repo, cleanup := newTestIntervalRepo(t)
	defer cleanup()

	ctx := context.Background()
	startedAt := time.Date(2026, 6, 1, 8, 0, 0, 0, time.UTC)
	require.NoError(t, repo.OpenInterval(ctx, "m1", ooedomain.Running, startedAt))

	endedAt := startedAt.Add(2 * time.Hour)
	require.NoError(t, repo.CloseOpen(ctx, "m1", endedAt))

	_, found, err := repo.LastOpen(ctx, "m1")
	require.NoError(t, err)
	assert.False(t, found)
}

func TestIntervalRepository_LastOpen_NotFound(t *testing.T) {
	repo, cleanup := newTestIntervalRepo(t)
	defer cleanup()

	interval, found, err := repo.LastOpen(context.Background(), "m1")
	require.NoError(t, err)
	assert.False(t, found)
	assert.Empty(t, interval.MachineID)
	assert.Nil(t, interval.EndedAt)
}

func TestIntervalRepository_RestartScenario_OpenIntervalSurvivesAndIsReadBack(t *testing.T) {
	db, err := platformdb.NewSQLiteConnection(":memory:")
	require.NoError(t, err)
	defer db.Close()

	require.NoError(t, platformdb.NewMigrator(db).Run(context.Background()))

	startedAt := time.Date(2026, 6, 1, 8, 0, 0, 0, time.UTC)

	// "before restart" — write an open interval
	repo1 := intervaldb.NewIntervalRepository(db)
	require.NoError(t, repo1.OpenInterval(context.Background(), "m1", ooedomain.Stopped, startedAt))

	// "after restart" — fresh repo instance on the same *sql.DB
	repo2 := intervaldb.NewIntervalRepository(db)
	interval, found, err := repo2.LastOpen(context.Background(), "m1")
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, ooedomain.Stopped, interval.State)
	assert.Equal(t, startedAt.UTC(), interval.StartedAt.UTC())
	assert.Nil(t, interval.EndedAt)
}

func TestIntervalRepository_OpenInterval_IsDowntimeFlagsDerivedFromState(t *testing.T) {
	cases := []struct {
		state        ooedomain.MachineState
		wantDowntime bool
		wantPlanned  bool
	}{
		{ooedomain.Running, false, false},
		{ooedomain.Stopped, true, false},
		{ooedomain.Maintenance, true, true},
		{ooedomain.Setup, true, true},
	}

	for _, tc := range cases {
		t.Run(string(tc.state), func(t *testing.T) {
			repo, cleanup := newTestIntervalRepo(t)
			defer cleanup()

			startedAt := time.Date(2026, 6, 1, 8, 0, 0, 0, time.UTC)
			require.NoError(t, repo.OpenInterval(context.Background(), "m1", tc.state, startedAt))

			interval, found, err := repo.LastOpen(context.Background(), "m1")
			require.NoError(t, err)
			require.True(t, found)
			assert.Equal(t, tc.state, interval.State)
			assert.Equal(t, tc.wantDowntime, interval.State.IsDowntime())
			assert.Equal(t, tc.wantPlanned, interval.State.IsPlannedStop())
		})
	}
}

func TestIntervalRepository_OpenInterval_ContextCancelled(t *testing.T) {
	repo, cleanup := newTestIntervalRepo(t)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := repo.OpenInterval(ctx, "m1", ooedomain.Running, time.Now())
	assert.Error(t, err)
}
