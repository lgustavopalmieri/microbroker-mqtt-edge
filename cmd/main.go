package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"microbroker-mqtt-edge/internal/common"
	"microbroker-mqtt-edge/internal/config"
	"microbroker-mqtt-edge/internal/modules/dispatch"
	dispatchdomain "microbroker-mqtt-edge/internal/modules/dispatch/domain"
	"microbroker-mqtt-edge/internal/modules/dispatch/workers"
	ingestiondomain "microbroker-mqtt-edge/internal/modules/ingestion/domain"

	"microbroker-mqtt-edge/internal/modules/ingestion/adapters/outbound/database"
	"microbroker-mqtt-edge/internal/modules/ingestion/application"
	"microbroker-mqtt-edge/internal/modules/session"
	sessiondomain "microbroker-mqtt-edge/internal/modules/session/domain"
	platformdb "microbroker-mqtt-edge/internal/platform/database"
)

func main() {
	// 1. Config
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// 2. Logger
	logger := common.NewSlogLogger(slog.Default())

	logger.Info("starting broker",
		"address", cfg.Address(),
		"topics", strings.Join(cfg.Topics, ", "),
		"maxClients", cfg.MaxClients,
	)

	// 3. Context with cancel
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 4. Signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// 5. Ensure DB directory exists
	dbDir := filepath.Dir(cfg.DBPath)
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		logger.Error("failed to create db directory", "path", dbDir, "error", err)
		os.Exit(1)
	}

	// 6. SQLite connection + migrations
	db, err := platformdb.NewSQLiteConnection(cfg.DBPath)
	if err != nil {
		logger.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	migrator := platformdb.NewMigrator(db)
	if err := migrator.Run(ctx); err != nil {
		logger.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}
	logger.Info("database ready", "path", cfg.DBPath)

	// 7. Store adapter
	store := database.NewSQLiteRepository(db)

	// 8. Channels
	sessionChan := make(chan session.Message, cfg.QueueBufferSize)
	ingestChan := make(chan ingestiondomain.Message, cfg.QueueBufferSize)
	dispatchChan := make(chan dispatchdomain.Message, cfg.QueueBufferSize)

	// 9. Bridge: session.Message → ingestion/domain.Message
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-sessionChan:
				if !ok {
					return
				}
				select {
				case ingestChan <- ingestiondomain.Message{
					ClientID:  msg.ClientID,
					Topic:     msg.Topic,
					Payload:   msg.Payload,
					Timezone:  msg.Timezone,
					Timestamp: msg.Timestamp,
				}:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	// 10. Workers
	loggerWorker := workers.NewLoggerWorker(logger)
	allWorkers := []dispatchdomain.Worker{loggerWorker}

	// 11. Pipeline (ingestion)
	pipeline := application.NewPipeline(cfg.Topics, store, dispatchChan, cfg.QueueBufferSize, logger)
	go pipeline.Start(ctx, ingestChan)

	// 12. Dispatcher
	dispatcher := dispatch.NewDispatcher(dispatchChan, allWorkers, logger)
	go dispatcher.Start(ctx)

	// 13. Session server
	topics, err := sessiondomain.NewTopicRegistry(cfg.Topics)
	if err != nil {
		logger.Error("failed to create topic registry", "error", err)
		os.Exit(1)
	}

	connMgr := session.NewConnectionManager(cfg.MaxClients)
	auth := session.NewEnvAuthenticator(cfg.Username, cfg.Password)
	server := session.NewServer(cfg.Address(), connMgr, auth, topics, sessionChan, cfg.Timezone, logger)

	go func() {
		if err := server.ListenAndServe(ctx); err != nil {
			logger.Error("server error", "error", err)
			cancel()
		}
	}()

	logger.Info("broker ready")

	// 14. Wait for signal
	<-sigCh
	logger.Info("shutting down...")
	cancel()

	// 15. Cleanup
	server.Close()
	dispatcher.Close()
	connMgr.CloseAll()

	logger.Info("shutdown complete")
}
