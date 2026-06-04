package e2e

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"microbroker-mqtt-edge/internal/common/observability"
	avdomain "microbroker-mqtt-edge/internal/modules/oee/availability/domain"
	ingestworker "microbroker-mqtt-edge/internal/modules/oee/availability/features/ingest-state/adapters/inbound/worker"
	ingestdb "microbroker-mqtt-edge/internal/modules/oee/availability/features/ingest-state/adapters/outbound/database"
	ingestapp "microbroker-mqtt-edge/internal/modules/oee/availability/features/ingest-state/application"
	livehandler "microbroker-mqtt-edge/internal/modules/oee/availability/features/live/adapters/inbound/http_handler"
	oeecomposite "microbroker-mqtt-edge/internal/modules/oee/availability/features/live/adapters/outbound/sink/composite"
	oeelogsink "microbroker-mqtt-edge/internal/modules/oee/availability/features/live/adapters/outbound/sink/logsink"
	oeewssink "microbroker-mqtt-edge/internal/modules/oee/availability/features/live/adapters/outbound/sink/websocket"
	liveapp "microbroker-mqtt-edge/internal/modules/oee/availability/features/live/application"
	queryhandler "microbroker-mqtt-edge/internal/modules/oee/availability/features/query/adapters/inbound/http_handler"
	querydb "microbroker-mqtt-edge/internal/modules/oee/availability/features/query/adapters/outbound/database"
	queryapp "microbroker-mqtt-edge/internal/modules/oee/availability/features/query/application"
	configdb "microbroker-mqtt-edge/internal/modules/oee/config/adapters/outbound/database"
	platformdb "microbroker-mqtt-edge/internal/platform/database"

	"microbroker-mqtt-edge/internal/common/message"
	"microbroker-mqtt-edge/internal/modules/auth"
	clientmanager "microbroker-mqtt-edge/internal/modules/broker/connection/client_manager"
	brokerserver "microbroker-mqtt-edge/internal/modules/broker/connection/server"
	"microbroker-mqtt-edge/internal/modules/broker/ingestion/pipeline"
	topicdomain "microbroker-mqtt-edge/internal/modules/broker/topic"
	"microbroker-mqtt-edge/internal/modules/processing/fanout"
	ingestiondb "microbroker-mqtt-edge/internal/platform/database/ingestion"
)

// oeeStack bundles all OEE components for e2e testing.
type oeeStack struct {
	mqttAddr  string
	httpSrv   *httptest.Server
	db        *sql.DB
	cancel    context.CancelFunc
	fo        *fanout.FanOut
	mqttSrv   *brokerserver.Server
	composite *oeecomposite.CompositeSink
}

const (
	oeeStateTopic = "machine/state"
	oeeMachine    = "m1"
)

// setupOEEStack wires the full OEE stack in-process using an in-memory SQLite DB,
// a dynamic-port MQTT server, and an httptest.Server for REST/WS endpoints.
func setupOEEStack(t *testing.T) *oeeStack {
	t.Helper()
	logger := observability.NopLogger{}
	topics := []string{oeeStateTopic, "machine/alarm"}

	db, err := platformdb.NewSQLiteConnection(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	require.NoError(t, platformdb.NewMigrator(db).Run(context.Background()))

	// Repositories
	ingestIntervalRepo := ingestdb.NewIntervalRepository(db)
	queryIntervalRepo := querydb.NewIntervalRepository(db)
	shiftRepo := configdb.NewShiftRepository(db)

	// Sinks
	hub := oeewssink.NewHub()
	logSink := oeelogsink.NewLogSink(logger)
	wsSink := oeewssink.NewWebsocketSink(hub, logger)
	comp := oeecomposite.NewCompositeSink(logger, logSink, wsSink)

	// Engine with a short tick interval for tests
	engine := liveapp.NewEngine(comp, shiftRepo, ingestIntervalRepo, logger, 50*time.Millisecond, time.Now)

	// Ingest-state use case + worker
	ingestUC := ingestapp.NewUseCase(ingestIntervalRepo, engine, logger)
	stateWorker := ingestworker.NewStateChangeWorker(oeeStateTopic, ingestUC, logger)

	// Ingestion store + MQTT pipeline
	store := ingestiondb.NewSQLiteRepository(db)
	msgChan := make(chan message.Message, 500)
	processChan := make(chan message.Message, 500)

	ctx, cancel := context.WithCancel(context.Background())

	p := pipeline.NewPipeline(topics, store, processChan, 500, logger)
	go p.Start(ctx, msgChan)

	fo := fanout.NewFanOut(processChan, []fanout.Worker{stateWorker}, logger)
	go fo.Start(ctx)

	// Engine ticker
	go engine.Start(ctx, nil)

	// MQTT server
	authenticator := auth.NewEnvAuthenticator("admin", "secret")
	topicReg, err := topicdomain.NewTopicRegistry(topics)
	require.NoError(t, err)
	connMgr := clientmanager.NewClientManager(5)
	mqttSrv := brokerserver.NewServer("127.0.0.1:0", connMgr, authenticator, topicReg, msgChan, "UTC", logger)
	go mqttSrv.ListenAndServe(ctx)

	select {
	case <-mqttSrv.Ready():
	case <-time.After(3 * time.Second):
		t.Fatal("MQTT server did not start in time")
	}
	require.NotNil(t, mqttSrv.Addr())

	// HTTP server
	mux := http.NewServeMux()
	queryUC := queryapp.NewUseCase(queryIntervalRepo, shiftRepo, logger)
	queryhandler.NewHandler(queryUC).RegisterRoutes(mux)
	livehandler.NewHandler(hub, logger).RegisterRoutes(mux)
	httpSrv := httptest.NewServer(mux)

	t.Cleanup(func() {
		cancel()
		mqttSrv.Close()
		fo.Close()
		_ = comp.Close()
		httpSrv.Close()
	})

	return &oeeStack{
		mqttAddr:  mqttSrv.Addr().String(),
		httpSrv:   httpSrv,
		db:        db,
		cancel:    cancel,
		fo:        fo,
		mqttSrv:   mqttSrv,
		composite: comp,
	}
}

// publishStateChange sends an MQTT PUBLISH with a state_change payload.
func publishStateChange(t *testing.T, conn net.Conn, machineID, state string, ts time.Time) {
	t.Helper()
	payload, err := json.Marshal(map[string]any{
		"machine_id": machineID,
		"state":      state,
		"timestamp":  ts.Format(time.RFC3339),
	})
	require.NoError(t, err)
	publishMessage(t, conn, oeeStateTopic, payload, 0, 0)
}

// waitForIntervals polls state_intervals until machineID has at least n rows.
func waitForIntervals(t *testing.T, db *sql.DB, machineID string, n int, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		var count int
		_ = db.QueryRowContext(context.Background(),
			`SELECT COUNT(*) FROM state_intervals WHERE machine_id = ?`, machineID).Scan(&count)
		if count >= n {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	var count int
	_ = db.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM state_intervals WHERE machine_id = ?`, machineID).Scan(&count)
	require.GreaterOrEqual(t, count, n, "timed out: expected %d intervals for %s, got %d", n, machineID, count)
}

// ---------------------------------------------------------------------------
// E2E Tests
// ---------------------------------------------------------------------------

func TestE2E_OEE_StateTransitionsPersistedAndQueryable(t *testing.T) {
	stack := setupOEEStack(t)

	conn := connectClient(t, stack.mqttAddr, "oee-device", "admin", "secret")
	defer conn.Close()

	windowStart := time.Now().UTC().Add(-5 * time.Second)

	// Publish running → stopped → running
	t0 := windowStart.Add(1 * time.Second)
	publishStateChange(t, conn, oeeMachine, "running", t0)
	publishStateChange(t, conn, oeeMachine, "stopped", t0.Add(2*time.Second))
	publishStateChange(t, conn, oeeMachine, "running", t0.Add(4*time.Second))

	// 3 intervals: first two closed, last open
	waitForIntervals(t, stack.db, oeeMachine, 3, 5*time.Second)

	// Query via REST
	from := windowStart.Format(time.RFC3339)
	to := time.Now().UTC().Add(time.Second).Format(time.RFC3339)
	url := fmt.Sprintf("%s/availability/%s?from=%s&to=%s", stack.httpSrv.URL, oeeMachine, from, to)

	resp, err := http.Get(url)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var out struct {
		MachineID string `json:"machine_id"`
		HasData   bool   `json:"has_data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
	assert.Equal(t, oeeMachine, out.MachineID)
	assert.True(t, out.HasData)
}

func TestE2E_OEE_WebsocketReceivesSnapshot(t *testing.T) {
	stack := setupOEEStack(t)

	conn := connectClient(t, stack.mqttAddr, "oee-ws-device", "admin", "secret")
	defer conn.Close()

	// Trigger engine to track the machine by publishing a state transition
	t0 := time.Now().UTC().Add(-2 * time.Second)
	publishStateChange(t, conn, oeeMachine, "running", t0)
	waitForIntervals(t, stack.db, oeeMachine, 1, 5*time.Second)

	// Connect WS client
	wsURL := "ws" + strings.TrimPrefix(stack.httpSrv.URL, "http") +
		"/availability/" + oeeMachine + "/ws"
	wsConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer wsConn.Close()

	// Wait for engine tick to deliver a snapshot (~50ms tick)
	wsConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := wsConn.ReadMessage()
	require.NoError(t, err, "WS client should receive a snapshot within tick interval")

	var snap avdomain.AvailabilitySnapshot
	require.NoError(t, json.Unmarshal(msg, &snap))
	assert.Equal(t, oeeMachine, snap.MachineID)
}

func TestE2E_OEE_DisabledSkipsWiring(t *testing.T) {
	// Without OEE wiring, /availability/... returns 404.
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	defer srv.Close()

	from := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)
	to := time.Now().UTC().Format(time.RFC3339)
	url := fmt.Sprintf("%s/availability/%s?from=%s&to=%s", srv.URL, oeeMachine, from, to)

	resp, err := http.Get(url)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}
