package bootstrap

import (
	"context"
	"database/sql"
	"net/http"

	"microbroker-mqtt-edge/cmd/broker/config"
	"microbroker-mqtt-edge/internal/common/observability"
	"microbroker-mqtt-edge/internal/modules/audit/features/query/adapters/inbound/http_handler"
	auditdb "microbroker-mqtt-edge/internal/modules/audit/features/query/adapters/outbound/database"
	"microbroker-mqtt-edge/internal/modules/audit/features/query/application"
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

	// Audit — wire feature: query
	auditRepo := auditdb.NewSQLiteRepository(db)
	auditUseCase := application.NewUseCase(auditRepo, logger)
	auditHandler := http_handler.NewHandler(auditUseCase)

	mux := http.NewServeMux()
	auditHandler.RegisterRoutes(mux)

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
