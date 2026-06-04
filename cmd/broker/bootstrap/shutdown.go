package bootstrap

import (
	"context"

	"microbroker-mqtt-edge/internal/common/observability"
)

// GracefulShutdown stops all components in the correct order.
func GracefulShutdown(modules *Modules, servers *Servers, logger observability.Logger) {
	logger.Info("shutting down...")

	modules.Server.Close()
	_ = servers.HTTPServer().Shutdown(context.Background())
	modules.FanOut.Close()
	modules.ClientMgr.CloseAll()

	if modules.OEECompositeSink != nil {
		_ = modules.OEECompositeSink.Close()
	}

	logger.Info("shutdown complete")
}
