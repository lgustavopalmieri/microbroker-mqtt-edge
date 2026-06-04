package bootstrap

import (
	"context"
	"database/sql"
	"os"
	"time"

	"microbroker-mqtt-edge/cmd/broker/config"
	"microbroker-mqtt-edge/internal/common/message"
	"microbroker-mqtt-edge/internal/common/observability"
	"microbroker-mqtt-edge/internal/modules/auth"
	clientmanager "microbroker-mqtt-edge/internal/modules/broker/connection/client_manager"
	"microbroker-mqtt-edge/internal/modules/broker/connection/server"
	"microbroker-mqtt-edge/internal/modules/broker/ingestion/pipeline"
	topicdomain "microbroker-mqtt-edge/internal/modules/broker/topic"
	"microbroker-mqtt-edge/internal/modules/processing/fanout"
	loggerworker "microbroker-mqtt-edge/internal/modules/processing/workers/logger"
	ingestiondb "microbroker-mqtt-edge/internal/platform/database/ingestion"

	// OEE — config module
	configdb "microbroker-mqtt-edge/internal/modules/oee/config/adapters/outbound/database"
	configapp "microbroker-mqtt-edge/internal/modules/oee/config/application"

	// OEE — ingest-state feature
	ingestworker "microbroker-mqtt-edge/internal/modules/oee/availability/features/ingest-state/adapters/inbound/worker"
	ingestdb "microbroker-mqtt-edge/internal/modules/oee/availability/features/ingest-state/adapters/outbound/database"
	ingestapp "microbroker-mqtt-edge/internal/modules/oee/availability/features/ingest-state/application"

	// OEE — query feature
	queryhandler "microbroker-mqtt-edge/internal/modules/oee/availability/features/query/adapters/inbound/http_handler"
	querydb "microbroker-mqtt-edge/internal/modules/oee/availability/features/query/adapters/outbound/database"
	queryapp "microbroker-mqtt-edge/internal/modules/oee/availability/features/query/application"

	// OEE — live feature (engine + sinks + ws endpoint)
	livehandler "microbroker-mqtt-edge/internal/modules/oee/availability/features/live/adapters/inbound/http_handler"
	oeecomposite "microbroker-mqtt-edge/internal/modules/oee/availability/features/live/adapters/outbound/sink/composite"
	oeelogsink "microbroker-mqtt-edge/internal/modules/oee/availability/features/live/adapters/outbound/sink/logsink"
	oeewssink "microbroker-mqtt-edge/internal/modules/oee/availability/features/live/adapters/outbound/sink/websocket"
	liveapp "microbroker-mqtt-edge/internal/modules/oee/availability/features/live/application"
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

	// OEE fields — nil when OEEEnabled=false
	OEEEngine        *liveapp.Engine
	OEECompositeSink *oeecomposite.CompositeSink
	OEEQueryHandler  *queryhandler.Handler
	OEEWSHandler     *livehandler.Handler
}

// InitModules wires all business modules together.
// Zero business logic — only construction and dependency injection.
func InitModules(cfg *config.Config, db *sql.DB, logger observability.Logger) (*Modules, error) {
	// Channels — no bridge needed, all modules share common/message.Message
	msgChan := make(chan message.Message, cfg.QueueBufferSize)
	processChan := make(chan message.Message, cfg.QueueBufferSize)

	// Store adapter
	store := ingestiondb.NewSQLiteRepository(db)

	// Workers — OEE state worker (if enabled) must be added before fanout creation.
	lw := loggerworker.NewLoggerWorker(logger)
	allWorkers := []fanout.Worker{lw}

	var (
		oeeEngine        *liveapp.Engine
		oeeCompositeSink *oeecomposite.CompositeSink
		oeeQueryHandler  *queryhandler.Handler
		oeeWSHandler     *livehandler.Handler
	)

	if cfg.OEEEnabled {
		var err error
		oeeEngine, oeeCompositeSink, oeeQueryHandler, oeeWSHandler, err =
			buildOEEModules(cfg, db, logger, &allWorkers)
		if err != nil {
			return nil, err
		}
	}

	// Ingestion pipeline
	p := pipeline.NewPipeline(cfg.Topics, store, processChan, cfg.QueueBufferSize, logger)

	// Processing fan-out (built after allWorkers is complete)
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
		Pipeline:         p,
		FanOut:           fo,
		Server:           srv,
		ClientMgr:        connMgr,
		MsgChan:          msgChan,
		ProcessChan:      processChan,
		OEEEngine:        oeeEngine,
		OEECompositeSink: oeeCompositeSink,
		OEEQueryHandler:  oeeQueryHandler,
		OEEWSHandler:     oeeWSHandler,
	}, nil
}

// buildOEEModules wires the full OEE availability pipeline and appends the
// state-change worker to allWorkers. Called only when OEEEnabled=true.
func buildOEEModules(
	cfg *config.Config,
	db *sql.DB,
	logger observability.Logger,
	allWorkers *[]fanout.Worker,
) (*liveapp.Engine, *oeecomposite.CompositeSink, *queryhandler.Handler, *livehandler.Handler, error) {

	// Config module — shift repository + optional seed
	shiftRepo := configdb.NewShiftRepository(db)
	if cfg.OEEShiftsPath != "" {
		if err := seedShifts(shiftRepo, cfg.OEEShiftsPath, logger); err != nil {
			logger.Warn("oee: shift seeding skipped", "error", err)
		}
	}

	// Ingest-state repository (IntervalStore + StateIntervalReader)
	ingestIntervalRepo := ingestdb.NewIntervalRepository(db)

	// Query interval repository (IntervalReader)
	queryIntervalRepo := querydb.NewIntervalRepository(db)

	// Live sinks
	hub := oeewssink.NewHub()
	logSink := oeelogsink.NewLogSink(logger)
	wsSink := oeewssink.NewWebsocketSink(hub, logger)
	compositeSink := oeecomposite.NewCompositeSink(logger, logSink, wsSink)

	// Live engine — shiftRepo satisfies ShiftReader; ingestIntervalRepo satisfies StateIntervalReader
	engine := liveapp.NewEngine(compositeSink, shiftRepo, ingestIntervalRepo, logger, cfg.OEETickInterval, time.Now)

	// Ingest-state use case: engine is the StateObserver
	ingestUC := ingestapp.NewUseCase(ingestIntervalRepo, engine, logger)

	// State-change worker appended before fanout is created
	stateWorker := ingestworker.NewStateChangeWorker(cfg.OEEStateTopic, ingestUC, logger)
	*allWorkers = append(*allWorkers, stateWorker)

	// Query use case + handler
	queryUC := queryapp.NewUseCase(queryIntervalRepo, shiftRepo, logger)
	queryH := queryhandler.NewHandler(queryUC)

	// WebSocket upgrade handler
	wsH := livehandler.NewHandler(hub, logger)

	return engine, compositeSink, queryH, wsH, nil
}

func seedShifts(store *configdb.ShiftRepository, path string, logger observability.Logger) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	shifts, err := configapp.LoadShiftsFromJSON(data)
	if err != nil {
		return err
	}
	uc := configapp.NewUseCase(store, logger)
	return uc.Seed(context.Background(), shifts)
}
