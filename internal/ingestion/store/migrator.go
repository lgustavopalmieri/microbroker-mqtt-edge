package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"sort"
	"strings"

	"microbroker-mqtt-edge/internal/ingestion/domain"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Migrator handles database schema migrations using embedded SQL files.
// Migrations are executed in alphabetical order (use numeric prefixes: 001_, 002_, etc.).
// Each migration runs inside a transaction. Already-applied migrations are tracked
// in a `schema_migrations` table to ensure idempotency.
type Migrator struct {
	db *sql.DB
}

// NewMigrator creates a Migrator for the given database connection.
func NewMigrator(db *sql.DB) *Migrator {
	return &Migrator{db: db}
}

// Run executes all pending migrations in order.
func (m *Migrator) Run(ctx context.Context) error {
	// Ensure the tracking table exists
	if err := m.ensureTrackingTable(ctx); err != nil {
		return err
	}

	// List applied migrations
	applied, err := m.getApplied(ctx)
	if err != nil {
		return err
	}

	// Read all migration files
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("%w: reading migrations dir: %v", domain.ErrMigrationFailed, err)
	}

	// Sort by name (alphabetical = numeric order with 001_ prefix)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	// Execute pending migrations
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}

		if applied[name] {
			continue // already applied
		}

		content, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("%w: reading %s: %v", domain.ErrMigrationFailed, name, err)
		}

		if err := m.applyMigration(ctx, name, string(content)); err != nil {
			return err
		}
	}

	return nil
}

func (m *Migrator) ensureTrackingTable(ctx context.Context) error {
	_, err := m.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			name       TEXT PRIMARY KEY,
			applied_at TEXT NOT NULL DEFAULT (datetime('now'))
		)
	`)
	if err != nil {
		return fmt.Errorf("%w: creating schema_migrations table: %v", domain.ErrMigrationFailed, err)
	}
	return nil
}

func (m *Migrator) getApplied(ctx context.Context) (map[string]bool, error) {
	rows, err := m.db.QueryContext(ctx, "SELECT name FROM schema_migrations")
	if err != nil {
		return nil, fmt.Errorf("%w: querying applied migrations: %v", domain.ErrMigrationFailed, err)
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("%w: scanning migration name: %v", domain.ErrMigrationFailed, err)
		}
		applied[name] = true
	}
	return applied, rows.Err()
}

func (m *Migrator) applyMigration(ctx context.Context, name, content string) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%w: begin tx for %s: %v", domain.ErrMigrationFailed, name, err)
	}
	defer tx.Rollback()

	// Execute the migration SQL (may contain multiple statements)
	statements := splitStatements(content)
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("%w: executing %s: %v", domain.ErrMigrationFailed, name, err)
		}
	}

	// Record the migration
	if _, err := tx.ExecContext(ctx,
		"INSERT INTO schema_migrations (name) VALUES (?)", name); err != nil {
		return fmt.Errorf("%w: recording %s: %v", domain.ErrMigrationFailed, name, err)
	}

	return tx.Commit()
}

// splitStatements splits SQL content by semicolons, handling basic cases.
func splitStatements(content string) []string {
	return strings.Split(content, ";")
}
