package database_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	shiftdb "microbroker-mqtt-edge/internal/modules/oee/config/adapters/outbound/database"
	ooedomain "microbroker-mqtt-edge/internal/modules/oee/domain"
	platformdb "microbroker-mqtt-edge/internal/platform/database"
)

func newTestShiftRepo(t *testing.T) (*shiftdb.ShiftRepository, func()) {
	t.Helper()

	db, err := platformdb.NewSQLiteConnection(":memory:")
	require.NoError(t, err)

	err = platformdb.NewMigrator(db).Run(context.Background())
	require.NoError(t, err)

	return shiftdb.NewShiftRepository(db), func() { db.Close() }
}

func shiftFactory(overrides ...func(*ooedomain.Shift)) ooedomain.Shift {
	s := ooedomain.Shift{
		Name:      "Morning",
		MachineID: "machine-1",
		StartMin:  0,
		EndMin:    120, // 00:00 to 02:00
		Weekdays:  nil, // every day
		TZ:        "UTC",
	}
	for _, fn := range overrides {
		fn(&s)
	}
	return s
}

// window00to02 is a fixed 2-hour window on 2026-06-01 00:00–02:00 UTC.
func window00to02() ooedomain.Window {
	from := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	return ooedomain.Window{From: from, To: from.Add(2 * time.Hour)}
}

// window00to04 is a fixed 4-hour window on 2026-06-01 00:00–04:00 UTC.
func window00to04() ooedomain.Window {
	from := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	return ooedomain.Window{From: from, To: from.Add(4 * time.Hour)}
}

func TestShiftRepository_Upsert_InsertsShiftWithBreaks(t *testing.T) {
	repo, cleanup := newTestShiftRepo(t)
	defer cleanup()

	s := shiftFactory(func(s *ooedomain.Shift) {
		s.Breaks = []ooedomain.Break{
			{StartMin: 60, EndMin: 90, Type: "rest"}, // 30-minute break at 01:00
		}
	})
	require.NoError(t, repo.Upsert(context.Background(), s))

	planned, found, err := repo.ForMachineWindow(context.Background(), s.MachineID, window00to02())
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, 90*time.Minute, planned) // 2h shift - 30m break
}

func TestShiftRepository_Upsert_IsIdempotent(t *testing.T) {
	repo, cleanup := newTestShiftRepo(t)
	defer cleanup()

	ctx := context.Background()

	// First upsert: 2-hour shift [00:00, 02:00]
	require.NoError(t, repo.Upsert(ctx, shiftFactory()))

	// Second upsert: same name+machine_id, shrunk to 1 hour [00:00, 01:00]
	require.NoError(t, repo.Upsert(ctx, shiftFactory(func(s *ooedomain.Shift) {
		s.EndMin = 60
	})))

	planned, found, err := repo.ForMachineWindow(ctx, "machine-1", window00to02())
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, time.Hour, planned) // second upsert wins; no duplicate rows
}

func TestShiftRepository_ForMachineWindow_ExactMachineMatch(t *testing.T) {
	repo, cleanup := newTestShiftRepo(t)
	defer cleanup()

	require.NoError(t, repo.Upsert(context.Background(), shiftFactory()))

	planned, found, err := repo.ForMachineWindow(context.Background(), "machine-1", window00to02())
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, 2*time.Hour, planned)
}

func TestShiftRepository_ForMachineWindow_WildcardFallback(t *testing.T) {
	repo, cleanup := newTestShiftRepo(t)
	defer cleanup()

	// Only a wildcard shift exists
	require.NoError(t, repo.Upsert(context.Background(), shiftFactory(func(s *ooedomain.Shift) {
		s.MachineID = "*"
	})))

	planned, found, err := repo.ForMachineWindow(context.Background(), "machine-1", window00to02())
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, 2*time.Hour, planned)
}

func TestShiftRepository_ForMachineWindow_ExactTakesPrecedenceOverWildcard(t *testing.T) {
	repo, cleanup := newTestShiftRepo(t)
	defer cleanup()

	ctx := context.Background()

	// Wildcard covers 4 hours
	require.NoError(t, repo.Upsert(ctx, shiftFactory(func(s *ooedomain.Shift) {
		s.MachineID = "*"
		s.EndMin = 240 // 04:00
	})))

	// Exact machine covers only 2 hours
	require.NoError(t, repo.Upsert(ctx, shiftFactory()))

	planned, found, err := repo.ForMachineWindow(ctx, "machine-1", window00to04())
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, 2*time.Hour, planned) // exact match (2h), not wildcard (4h)
}

func TestShiftRepository_ForMachineWindow_NotFound(t *testing.T) {
	repo, cleanup := newTestShiftRepo(t)
	defer cleanup()

	planned, found, err := repo.ForMachineWindow(context.Background(), "machine-1", window00to02())
	require.NoError(t, err)
	assert.False(t, found)
	assert.Equal(t, time.Duration(0), planned)
}

func TestShiftRepository_Upsert_ContextCancelled(t *testing.T) {
	repo, cleanup := newTestShiftRepo(t)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := repo.Upsert(ctx, shiftFactory())
	assert.Error(t, err)
}
