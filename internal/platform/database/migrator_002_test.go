package database_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"microbroker-mqtt-edge/internal/platform/database"
)

func newMigratedDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	db, err := database.NewSQLiteConnection(":memory:")
	require.NoError(t, err)
	require.NoError(t, database.NewMigrator(db).Run(context.Background()))
	return db, func() { db.Close() }
}

func tableColumns(t *testing.T, db *sql.DB, table string) map[string]bool {
	t.Helper()
	rows, err := db.QueryContext(context.Background(), "PRAGMA table_info("+table+")")
	require.NoError(t, err)
	defer rows.Close()
	cols := make(map[string]bool)
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull int
		var dfltVal sql.NullString
		var pk int
		require.NoError(t, rows.Scan(&cid, &name, &colType, &notNull, &dfltVal, &pk))
		cols[name] = true
	}
	require.NoError(t, rows.Err())
	return cols
}

func tableIndexes(t *testing.T, db *sql.DB, table string) map[string]bool {
	t.Helper()
	rows, err := db.QueryContext(context.Background(), "PRAGMA index_list("+table+")")
	require.NoError(t, err)
	defer rows.Close()
	names := make(map[string]bool)
	for rows.Next() {
		var seq, unique int
		var name, origin, partial string
		require.NoError(t, rows.Scan(&seq, &name, &unique, &origin, &partial))
		names[name] = true
	}
	require.NoError(t, rows.Err())
	return names
}

func TestMigrator_002_TablesCreated(t *testing.T) {
	tests := []struct {
		name    string
		table   string
		columns []string
	}{
		{
			name:    "creates shifts table with all required columns",
			table:   "shifts",
			columns: []string{"id", "name", "machine_id", "start_minute", "end_minute", "weekdays", "timezone", "active", "created_at"},
		},
		{
			name:    "creates shift_breaks table with all required columns",
			table:   "shift_breaks",
			columns: []string{"id", "shift_id", "start_minute", "end_minute", "type"},
		},
		{
			name:    "creates state_intervals table with all required columns",
			table:   "state_intervals",
			columns: []string{"id", "machine_id", "state", "is_downtime", "is_planned_stop", "started_at", "ended_at", "reason", "created_at"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, cleanup := newMigratedDB(t)
			defer cleanup()

			cols := tableColumns(t, db, tc.table)
			require.NotEmpty(t, cols, "table %q should exist", tc.table)
			for _, col := range tc.columns {
				assert.True(t, cols[col], "column %q missing from %q", col, tc.table)
			}
		})
	}
}

func TestMigrator_002_IndexesCreated(t *testing.T) {
	tests := []struct {
		name      string
		table     string
		indexName string
	}{
		{
			name:      "idx_shifts_machine_id exists on shifts",
			table:     "shifts",
			indexName: "idx_shifts_machine_id",
		},
		{
			name:      "idx_shift_breaks_shift_id exists on shift_breaks",
			table:     "shift_breaks",
			indexName: "idx_shift_breaks_shift_id",
		},
		{
			name:      "idx_state_intervals_machine_started exists on state_intervals",
			table:     "state_intervals",
			indexName: "idx_state_intervals_machine_started",
		},
		{
			name:      "idx_state_intervals_machine_ended exists on state_intervals",
			table:     "state_intervals",
			indexName: "idx_state_intervals_machine_ended",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, cleanup := newMigratedDB(t)
			defer cleanup()

			indexes := tableIndexes(t, db, tc.table)
			assert.True(t, indexes[tc.indexName], "index %q missing from %q", tc.indexName, tc.table)
		})
	}
}

func TestMigrator_002_Idempotent(t *testing.T) {
	t.Run("re-running migration 002 is a no-op (tracked in schema_migrations)", func(t *testing.T) {
		db, cleanup := newMigratedDB(t)
		defer cleanup()

		err := database.NewMigrator(db).Run(context.Background())
		require.NoError(t, err, "second Run should be a no-op")

		var count int
		row := db.QueryRowContext(context.Background(),
			"SELECT COUNT(*) FROM schema_migrations WHERE name = '002_create_oee_availability.sql'")
		require.NoError(t, row.Scan(&count))
		assert.Equal(t, 1, count, "migration should be recorded exactly once")
	})
}

func TestMigrator_002_StaticConstraints(t *testing.T) {
	t.Run("state_intervals allows NULL ended_at (open interval)", func(t *testing.T) {
		db, cleanup := newMigratedDB(t)
		defer cleanup()

		_, err := db.ExecContext(context.Background(),
			`INSERT INTO state_intervals (machine_id, state, is_downtime, is_planned_stop, started_at)
			 VALUES ('m1', 'running', 0, 0, '2026-06-03T08:00:00Z')`)
		assert.NoError(t, err, "NULL ended_at must be allowed")
	})

	t.Run("state_intervals enforces NOT NULL on machine_id, state, started_at", func(t *testing.T) {
		db, cleanup := newMigratedDB(t)
		defer cleanup()

		_, err := db.ExecContext(context.Background(),
			`INSERT INTO state_intervals (machine_id, state, is_downtime, is_planned_stop, started_at)
			 VALUES (NULL, 'running', 0, 0, '2026-06-03T08:00:00Z')`)
		assert.Error(t, err, "NULL machine_id should be rejected")
	})

	t.Run("shifts active column defaults to 1", func(t *testing.T) {
		db, cleanup := newMigratedDB(t)
		defer cleanup()

		_, err := db.ExecContext(context.Background(),
			`INSERT INTO shifts (name, machine_id, start_minute, end_minute, weekdays, timezone)
			 VALUES ('Morning', '*', 360, 960, '1,2,3,4,5', 'UTC')`)
		require.NoError(t, err)

		var active int
		row := db.QueryRowContext(context.Background(), "SELECT active FROM shifts LIMIT 1")
		require.NoError(t, row.Scan(&active))
		assert.Equal(t, 1, active, "active should default to 1")
	})
}
