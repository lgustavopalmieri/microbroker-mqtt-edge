package database

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	ooedomain "microbroker-mqtt-edge/internal/modules/oee/domain"
)

// ShiftRepository implements application.ShiftStore over SQLite.
// It owns no connection lifecycle — the caller (bootstrap) injects *sql.DB.
type ShiftRepository struct {
	db *sql.DB
	mu sync.Mutex
}

// NewShiftRepository creates a repository with an injected database connection.
func NewShiftRepository(db *sql.DB) *ShiftRepository {
	return &ShiftRepository{db: db}
}

// Upsert replaces the shift (keyed by machine_id+name) and its breaks atomically.
func (r *ShiftRepository) Upsert(ctx context.Context, s ooedomain.Shift) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("shift repo: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	_, err = tx.ExecContext(ctx, `
		DELETE FROM shift_breaks WHERE shift_id IN (
			SELECT id FROM shifts WHERE machine_id = ? AND name = ?
		)
	`, s.MachineID, s.Name)
	if err != nil {
		return fmt.Errorf("shift repo: delete breaks: %w", err)
	}

	_, err = tx.ExecContext(ctx,
		`DELETE FROM shifts WHERE machine_id = ? AND name = ?`,
		s.MachineID, s.Name)
	if err != nil {
		return fmt.Errorf("shift repo: delete shift: %w", err)
	}

	tz := s.TZ
	if tz == "" {
		tz = "UTC"
	}

	result, err := tx.ExecContext(ctx, `
		INSERT INTO shifts (name, machine_id, start_minute, end_minute, weekdays, timezone, active)
		VALUES (?, ?, ?, ?, ?, ?, 1)
	`, s.Name, s.MachineID, s.StartMin, s.EndMin, weekdaysToCSV(s.Weekdays), tz)
	if err != nil {
		return fmt.Errorf("shift repo: insert shift: %w", err)
	}

	shiftID, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("shift repo: last insert id: %w", err)
	}

	for _, b := range s.Breaks {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO shift_breaks (shift_id, start_minute, end_minute, type) VALUES (?, ?, ?, ?)`,
			shiftID, b.StartMin, b.EndMin, b.Type)
		if err != nil {
			return fmt.Errorf("shift repo: insert break: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("shift repo: commit: %w", err)
	}
	return nil
}

// ForMachineWindow returns the total planned production time for the given machine
// and window. It tries an exact machine_id match first; if none is found, it falls
// back to the wildcard machine_id ("*"). Returns found=false when no config exists.
func (r *ShiftRepository) ForMachineWindow(ctx context.Context, machineID string, w ooedomain.Window) (time.Duration, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	shifts, err := r.queryShifts(ctx, machineID)
	if err != nil {
		return 0, false, err
	}
	if len(shifts) == 0 && machineID != "*" {
		shifts, err = r.queryShifts(ctx, "*")
		if err != nil {
			return 0, false, err
		}
	}
	if len(shifts) == 0 {
		return 0, false, nil
	}

	var total time.Duration
	for _, s := range shifts {
		total += s.PlannedProductionTime(w)
	}
	return total, true, nil
}

// queryShifts fetches all active shifts for a given machine_id. It closes the
// row cursor before issuing nested break queries (single-connection SQLite).
func (r *ShiftRepository) queryShifts(ctx context.Context, machineID string) ([]ooedomain.Shift, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, machine_id, start_minute, end_minute, weekdays, timezone
		FROM shifts WHERE machine_id = ? AND active = 1
	`, machineID)
	if err != nil {
		return nil, fmt.Errorf("shift repo: query shifts: %w", err)
	}

	type shiftRow struct {
		id          int64
		name        string
		machineID   string
		startMin    int
		endMin      int
		weekdaysCSV string
		tz          string
	}

	var collected []shiftRow
	for rows.Next() {
		var sr shiftRow
		if err := rows.Scan(&sr.id, &sr.name, &sr.machineID, &sr.startMin, &sr.endMin, &sr.weekdaysCSV, &sr.tz); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("shift repo: scan shift: %w", err)
		}
		collected = append(collected, sr)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, fmt.Errorf("shift repo: rows error: %w", err)
	}
	// Close explicitly before nested queries — single-connection SQLite pool.
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("shift repo: close rows: %w", err)
	}

	shifts := make([]ooedomain.Shift, 0, len(collected))
	for _, sr := range collected {
		breaks, err := r.queryBreaks(ctx, sr.id)
		if err != nil {
			return nil, err
		}
		shifts = append(shifts, ooedomain.Shift{
			Name:      sr.name,
			MachineID: sr.machineID,
			StartMin:  sr.startMin,
			EndMin:    sr.endMin,
			Weekdays:  csvToWeekdays(sr.weekdaysCSV),
			TZ:        sr.tz,
			Breaks:    breaks,
		})
	}
	return shifts, nil
}

func (r *ShiftRepository) queryBreaks(ctx context.Context, shiftID int64) ([]ooedomain.Break, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT start_minute, end_minute, type FROM shift_breaks WHERE shift_id = ?`,
		shiftID)
	if err != nil {
		return nil, fmt.Errorf("shift repo: query breaks: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	var breaks []ooedomain.Break
	for rows.Next() {
		var b ooedomain.Break
		if err := rows.Scan(&b.StartMin, &b.EndMin, &b.Type); err != nil {
			return nil, fmt.Errorf("shift repo: scan break: %w", err)
		}
		breaks = append(breaks, b)
	}
	return breaks, rows.Err()
}

func weekdaysToCSV(weekdays []time.Weekday) string {
	if len(weekdays) == 0 {
		return ""
	}
	parts := make([]string, len(weekdays))
	for i, wd := range weekdays {
		parts[i] = strconv.Itoa(int(wd))
	}
	return strings.Join(parts, ",")
}

func csvToWeekdays(csv string) []time.Weekday {
	if csv == "" {
		return nil
	}
	parts := strings.Split(csv, ",")
	weekdays := make([]time.Weekday, 0, len(parts))
	for _, p := range parts {
		n, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			continue
		}
		weekdays = append(weekdays, time.Weekday(n))
	}
	return weekdays
}
