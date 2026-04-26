package bootstrap

import (
	"database/sql"

	"microbroker-mqtt-edge/cmd/broker/config"
	"microbroker-mqtt-edge/internal/common/message"
	"microbroker-mqtt-edge/internal/common/observability"
	"microbroker-mqtt-edge/internal/modules/auth"
	clientmanager "microbroker-mqtt-edge/internal/modules/broker/connection/client_manager"
	"microbroker-mqtt-edge/internal/modules/broker/connection/server"
	"microbroker-mqtt-edge/internal/modules/broker/ingestion/pipeline"
	"microbroker-mqtt-edge/internal/modules/processing/fanout"
	loggerworker "microbroker-mqtt-edge/internal/modules/processing/workers/logger"
	topicdomain "microbroker-mqtt-edge/internal/modules/topic"
	ingestiondb "microbroker-mqtt-edge/internal/platform/database/ingestion"
)

// Modules holds all initialized business modules.
type Modules struct {
	Pipeline  *pipeline.Pipeline
	FanOut    *fanout.FanOut
	Server    *server.Server
	ClientMgr *clientmanager.ClientManager

	// channels owned by bootstrap, passed to modules
	MsgChan     chan message.Message
	ProcessChan chan message.Message
}

// InitModules wires all business modules together.
// Zero business logic — only construction and dependency injection.
func InitModules(cfg *config.Config, db *sql.DB, logger observability.Logger) (*Modules, error) {
	// Channels — no bridge needed, all modules share common/message.Message
	msgChan := make(chan message.Message, cfg.QueueBufferSize)
	processChan := make(chan message.Message, cfg.QueueBufferSize)

	// Store adapter
	store := ingestiondb.NewSQLiteRepository(db)

	// Workers
	lw := loggerworker.NewLoggerWorker(logger)
	allWorkers := []fanout.Worker{lw}

	// Ingestion pipeline
	p := pipeline.NewPipeline(cfg.Topics, store, processChan, cfg.QueueBufferSize, logger)

	// Processing fan-out
	fo := fanout.NewFanOut(processChan, allWorkers, logger)

	// Auth
	authenticator := auth.NewEnvAuthenticator(cfg.Username, cfg.Password)

	// Connection
	topics, err := topicdomain.NewTopicRegistry(cfg.Topics)
	if err != nil {
		return nil, err
	}

	connMgr := clientmanager.NewClientManager(cfg.MaxClients)
	srv := server.NewServer(cfg.Address(), connMgr, authenticator, topics, msgChan, cfg.Timezone, logger)

	return &Modules{
		Pipeline:    p,
		FanOut:      fo,
		Server:      srv,
		ClientMgr:   connMgr,
		MsgChan:     msgChan,
		ProcessChan: processChan,
	}, nil
}
