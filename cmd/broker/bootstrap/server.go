package bootstrap

import (
	"context"
	"database/sql"
	"net/http"

	"microbroker-mqtt-edge/cmd/broker/config"
	"microbroker-mqtt-edge/internal/common/observability"

	countHandler "microbroker-mqtt-edge/internal/modules/audit/raw/features/count-by-topic/adapters/inbound/http_handler"
	countDB "microbroker-mqtt-edge/internal/modules/audit/raw/features/count-by-topic/adapters/outbound/database"
	countApp "microbroker-mqtt-edge/internal/modules/audit/raw/features/count-by-topic/application"

	getHandler "microbroker-mqtt-edge/internal/modules/audit/raw/features/get-by-topic/adapters/inbound/http_handler"
	getDB "microbroker-mqtt-edge/internal/modules/audit/raw/features/get-by-topic/adapters/outbound/database"
	getApp "microbroker-mqtt-edge/internal/modules/audit/raw/features/get-by-topic/application"
)

// Servers holds the running TCP and HTTP servers.
type Servers struct {
	httpServer *http.Server
}

// StartServers launches the TCP (MQTT) and HTTP (audit API) servers.
// Returns a Servers handle for shutdown.
func StartServers(ctx context.Context, cancel context.CancelFunc, cfg *config.Config, modules *Modules, db *sql.DB, logger observability.Logger) *Servers {
	// Start ingestion pipeline
	go modules.Pipeline.Start(ctx, modules.MsgChan)

	// Start processing fan-out
	go modules.FanOut.Start(ctx)

	// Start MQTT TCP server
	go func() {
		if err := modules.Server.ListenAndServe(ctx); err != nil {
			logger.Error("server error", "error", err)
			cancel()
		}
	}()

	// Audit features
	mux := http.NewServeMux()

	// Feature: get-by-topic
	getRepo := getDB.NewSQLiteRepository(db)
	getUC := getApp.NewUseCase(getRepo, logger)
	getH := getHandler.NewHandler(getUC)
	getH.RegisterRoutes(mux)

	// Feature: count-by-topic
	countRepo := countDB.NewSQLiteRepository(db)
	countUC := countApp.NewUseCase(countRepo, logger)
	countH := countHandler.NewHandler(countUC)
	countH.RegisterRoutes(mux)

	// OEE availability features (registered only when OEE is enabled)
	if modules.OEEQueryHandler != nil {
		modules.OEEQueryHandler.RegisterRoutes(mux)
	}
	if modules.OEEWSHandler != nil {
		modules.OEEWSHandler.RegisterRoutes(mux)
	}
	if modules.OEEEngine != nil {
		go modules.OEEEngine.Start(ctx, nil)
	}

	httpServer := &http.Server{Addr: cfg.HTTPAddress(), Handler: mux}
	go func() {
		logger.Info("audit API started", "address", cfg.HTTPAddress())
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("http server error", "error", err)
		}
	}()

	logger.Info("broker ready")

	return &Servers{httpServer: httpServer}
}

// HTTPServer returns the underlying *http.Server for shutdown.
func (s *Servers) HTTPServer() *http.Server {
	return s.httpServer
}
