package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"microbroker-mqtt-edge/cmd/broker/bootstrap"
	"microbroker-mqtt-edge/cmd/broker/config"
	"microbroker-mqtt-edge/internal/common/observability"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	logger := observability.NewSlogLogger(slog.Default())
	logger.Info("starting broker",
		"address", cfg.Address(),
		"topics", strings.Join(cfg.Topics, ", "),
		"maxClients", cfg.MaxClients,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	db, err := bootstrap.InitDatabase(ctx, cfg.DBPath, logger)
	if err != nil {
		logger.Error("database init failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	modules, err := bootstrap.InitModules(cfg, db, logger)
	if err != nil {
		logger.Error("module init failed", "error", err)
		os.Exit(1)
	}

	servers := bootstrap.StartServers(ctx, cancel, cfg, modules, db, logger)

	<-sigCh
	bootstrap.GracefulShutdown(modules, servers, logger)
}
