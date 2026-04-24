package bootstrap

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"

	"microbroker-mqtt-edge/internal/common/observability"
	platformdb "microbroker-mqtt-edge/internal/platform/database"
)

// InitDatabase ensures the DB directory exists, opens a SQLite connection,
// and runs all pending migrations. Returns the *sql.DB (caller must close).
func InitDatabase(ctx context.Context, dbPath string, logger observability.Logger) (*sql.DB, error) {
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		return nil, err
	}

	db, err := platformdb.NewSQLiteConnection(dbPath)
	if err != nil {
		return nil, err
	}

	migrator := platformdb.NewMigrator(db)
	if err := migrator.Run(ctx); err != nil {
		db.Close()
		return nil, err
	}

	logger.Info("database ready", "path", dbPath)
	return db, nil
}
