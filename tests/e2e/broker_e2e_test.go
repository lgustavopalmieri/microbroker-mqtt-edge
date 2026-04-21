package e2e

import (
	"context"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"microbroker-mqtt-edge/internal/modules/dispatch"
	dispatchdomain "microbroker-mqtt-edge/internal/modules/dispatch/domain"
	"microbroker-mqtt-edge/internal/modules/ingestion/adapters/outbound/database"
	"microbroker-mqtt-edge/internal/modules/ingestion/application"
	ingestiondomain "microbroker-mqtt-edge/internal/modules/ingestion/domain"
	"microbroker-mqtt-edge/internal/modules/protocol"
	"microbroker-mqtt-edge/internal/modules/session"
	sessiondomain "microbroker-mqtt-edge/internal/modules/session/domain"
	platformdb "microbroker-mqtt-edge/internal/platform/database"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

type nopLogger struct{}

func (nopLogger) Info(string, ...any)  {}
func (nopLogger) Error(string, ...any) {}
func (nopLogger) Warn(string, ...any)  {}
func (nopLogger) Debug(string, ...any) {}

// collectWorker is a mock Worker that records every message it receives.
type collectWorker struct {
	mu       sync.Mutex
	received []dispatchdomain.Message
}

func (w *collectWorker) Name() string { return "collector" }
func (w *collectWorker) Process(_ context.Context, msg dispatchdomain.Message) error {
	w.mu.Lock()
	w.received = append(w.received, msg)
	w.mu.Unlock()
	return nil
}
func (w *collectWorker) Close() error { return nil }
func (w *collectWorker) messages() []dispatchdomain.Message {
	w.mu.Lock()
	defer w.mu.Unlock()
	cp := make([]dispatchdomain.Message, len(w.received))
	copy(cp, w.received)
	return cp
}

// broker bundles all components for a running test broker.
type broker struct {
	server *session.Server
	store  *database.SQLiteRepository
	worker *collectWorker
	cancel context.CancelFunc
	addr   string
	topics []string
}

func setupBroker(t *testing.T, topics []string) *broker {
	t.Helper()

	// SQLite :memory: + migrations
	db, err := platformdb.NewSQLiteConnection(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	migrator := platformdb.NewMigrator(db)
	require.NoError(t, migrator.Run(context.Background()))

	store := database.NewSQLiteRepository(db)

	// Channels
	sessionChan := make(chan session.Message, 1000)
	ingestChan := make(chan ingestiondomain.Message, 1000)
	dispatchChan := make(chan dispatchdomain.Message, 1000)

	ctx, cancel := context.WithCancel(context.Background())

	// Bridge: session.Message → ingestion/domain.Message
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

	// Worker
	w := &collectWorker{}

	// Pipeline
	pipeline := application.NewPipeline(topics, store, dispatchChan, 1000, nopLogger{})
	go pipeline.Start(ctx, ingestChan)

	// Dispatcher
	dispatcher := dispatch.NewDispatcher(dispatchChan, []dispatchdomain.Worker{w}, nopLogger{})
	go dispatcher.Start(ctx)

	// Session server
	topicReg, err := sessiondomain.NewTopicRegistry(topics)
	require.NoError(t, err)

	connMgr := session.NewConnectionManager(5)
	auth := session.NewEnvAuthenticator("admin", "secret")
	srv := session.NewServer("127.0.0.1:0", connMgr, auth, topicReg, sessionChan, "UTC", nopLogger{})

	go func() {
		srv.ListenAndServe(ctx)
	}()

	// Wait for listener to be ready via the server's ready channel
	select {
	case <-srv.Ready():
	case <-time.After(3 * time.Second):
		t.Fatal("server did not start in time")
	}
	require.NotNil(t, srv.Addr(), "server did not bind")

	t.Cleanup(func() {
		cancel()
		srv.Close()
		dispatcher.Close()
	})

	return &broker{
		server: srv,
		store:  store,
		worker: w,
		cancel: cancel,
		addr:   srv.Addr().String(),
		topics: topics,
	}
}

// connectClient opens a real TCP connection, sends CONNECT, and reads CONNACK.
func connectClient(t *testing.T, addr, clientID, user, pass string) net.Conn {
	t.Helper()
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	require.NoError(t, err)

	conn.Write(buildConnectPacket(clientID, user, pass, 60))

	_, rc := readConnack(t, conn)
	require.Equal(t, protocol.ConnAccepted, rc, "expected CONNACK accepted")
	return conn
}

// connectClientRaw opens TCP, sends CONNECT, returns conn + return code (no assertion).
func connectClientRaw(t *testing.T, addr, clientID, user, pass string) (net.Conn, protocol.ConnackReturnCode) {
	t.Helper()
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	require.NoError(t, err)

	conn.Write(buildConnectPacket(clientID, user, pass, 60))
	_, rc := readConnack(t, conn)
	return conn, rc
}

// publishMessage sends a PUBLISH packet. If QoS 1, reads and validates PUBACK.
func publishMessage(t *testing.T, conn net.Conn, topic string, payload []byte, qos byte, packetID uint16) {
	t.Helper()
	conn.Write(buildPublishPacket(topic, payload, qos, packetID))
	if qos == 1 {
		readPuback(t, conn, packetID)
	}
}

// --- packet builders (same logic as handler_test.go, using exported protocol funcs) ---

func buildConnectPacket(clientID, username, password string, keepAlive uint16) []byte {
	var payload []byte
	payload = append(payload, 0x00, 0x04, 'M', 'Q', 'T', 'T') // Protocol Name
	payload = append(payload, 0x04)                           // Protocol Level
	payload = append(payload, 0xC2)                           // Flags: Username+Password+CleanSession
	payload = append(payload, byte(keepAlive>>8), byte(keepAlive&0xFF))
	payload = append(payload, protocol.WriteUTF8String(clientID)...)
	payload = append(payload, protocol.WriteUTF8String(username)...)
	passBytes := []byte(password)
	payload = append(payload, byte(len(passBytes)>>8), byte(len(passBytes)&0xFF))
	payload = append(payload, passBytes...)

	var pkt []byte
	pkt = append(pkt, 0x10) // CONNECT
	pkt = append(pkt, protocol.EncodeRemainingLength(len(payload))...)
	pkt = append(pkt, payload...)
	return pkt
}

func buildPublishPacket(topic string, payload []byte, qos byte, packetID uint16) []byte {
	var data []byte
	data = append(data, protocol.WriteUTF8String(topic)...)
	if qos > 0 {
		data = append(data, byte(packetID>>8), byte(packetID&0xFF))
	}
	data = append(data, payload...)

	flags := qos << 1
	var pkt []byte
	pkt = append(pkt, 0x30|flags)
	pkt = append(pkt, protocol.EncodeRemainingLength(len(data))...)
	pkt = append(pkt, data...)
	return pkt
}

func readConnack(t *testing.T, conn net.Conn) (bool, protocol.ConnackReturnCode) {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	buf := make([]byte, 4)
	_, err := io.ReadFull(conn, buf)
	require.NoError(t, err)
	require.Equal(t, byte(0x20), buf[0])
	require.Equal(t, byte(0x02), buf[1])
	sessionPresent := buf[2]&0x01 != 0
	return sessionPresent, protocol.ConnackReturnCode(buf[3])
}

func readPuback(t *testing.T, conn net.Conn, expectedID uint16) {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	buf := make([]byte, 4)
	_, err := io.ReadFull(conn, buf)
	require.NoError(t, err)
	require.Equal(t, byte(0x40), buf[0])
	gotID := uint16(buf[2])<<8 | uint16(buf[3])
	require.Equal(t, expectedID, gotID)
}

// waitForWorkerMessages polls the worker until it has at least n messages or timeout.
func waitForWorkerMessages(t *testing.T, w *collectWorker, n int, timeout time.Duration) []dispatchdomain.Message {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		msgs := w.messages()
		if len(msgs) >= n {
			return msgs
		}
		time.Sleep(20 * time.Millisecond)
	}
	msgs := w.messages()
	require.GreaterOrEqual(t, len(msgs), n, "timed out waiting for %d worker messages, got %d", n, len(msgs))
	return msgs
}

// ---------------------------------------------------------------------------
// E2E Tests
// ---------------------------------------------------------------------------

func TestE2E_HappyPath_FullPipeline(t *testing.T) {
	b := setupBroker(t, []string{"machine/status", "machine/alarm"})

	conn := connectClient(t, b.addr, "device-01", "admin", "secret")
	defer conn.Close()

	// Publish 3 messages to machine/status, 2 to machine/alarm
	publishMessage(t, conn, "machine/status", []byte(`{"temp":20}`), 0, 0)
	publishMessage(t, conn, "machine/status", []byte(`{"temp":21}`), 0, 0)
	publishMessage(t, conn, "machine/alarm", []byte(`{"code":1}`), 1, 1)
	publishMessage(t, conn, "machine/status", []byte(`{"temp":22}`), 0, 0)
	publishMessage(t, conn, "machine/alarm", []byte(`{"code":2}`), 1, 2)

	// Wait for worker to receive all 5
	workerMsgs := waitForWorkerMessages(t, b.worker, 5, 5*time.Second)
	assert.Len(t, workerMsgs, 5)

	// Verify SQLite
	ctx := context.Background()
	statusMsgs, err := b.store.GetByTopic(ctx, "machine/status")
	require.NoError(t, err)
	assert.Len(t, statusMsgs, 3)
	assert.Equal(t, []byte(`{"temp":20}`), statusMsgs[0].Payload)
	assert.Equal(t, []byte(`{"temp":21}`), statusMsgs[1].Payload)
	assert.Equal(t, []byte(`{"temp":22}`), statusMsgs[2].Payload)

	alarmMsgs, err := b.store.GetByTopic(ctx, "machine/alarm")
	require.NoError(t, err)
	assert.Len(t, alarmMsgs, 2)

	// Verify clientID
	for _, m := range statusMsgs {
		assert.Equal(t, "device-01", m.ClientID)
	}
}

func TestE2E_MultipleClientsSimultaneous(t *testing.T) {
	b := setupBroker(t, []string{"data/sensor"})

	const numClients = 3
	const msgsPerClient = 10
	total := numClients * msgsPerClient

	var wg sync.WaitGroup
	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(clientIdx int) {
			defer wg.Done()
			clientID := "client-" + string(rune('A'+clientIdx))
			conn := connectClient(t, b.addr, clientID, "admin", "secret")
			defer conn.Close()

			for j := 0; j < msgsPerClient; j++ {
				payload := []byte(`{"client":` + string(rune('A'+clientIdx)) + `,"seq":` + string(rune('0'+j)) + `}`)
				publishMessage(t, conn, "data/sensor", payload, 0, 0)
			}
		}(i)
	}
	wg.Wait()

	// Wait for all messages to flow through
	workerMsgs := waitForWorkerMessages(t, b.worker, total, 10*time.Second)
	assert.GreaterOrEqual(t, len(workerMsgs), total)

	// Verify SQLite has all messages
	ctx := context.Background()
	dbMsgs, err := b.store.GetByTopic(ctx, "data/sensor")
	require.NoError(t, err)
	assert.Len(t, dbMsgs, total)
}

func TestE2E_SixthClientRejected(t *testing.T) {
	b := setupBroker(t, []string{"t/1"})

	// Connect 5 clients
	conns := make([]net.Conn, 5)
	for i := 0; i < 5; i++ {
		clientID := "c" + string(rune('0'+i))
		conns[i] = connectClient(t, b.addr, clientID, "admin", "secret")
		defer conns[i].Close()
	}

	// 6th client: TCP connects but server closes it before CONNACK
	conn6, err := net.DialTimeout("tcp", b.addr, 2*time.Second)
	require.NoError(t, err)
	defer conn6.Close()

	conn6.Write(buildConnectPacket("c5", "admin", "secret", 60))

	// Should get EOF or connection reset (server closes before/after CONNACK)
	conn6.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 4)
	n, err := io.ReadFull(conn6, buf)
	if err == nil && n == 4 {
		// Server might send CONNACK with "unavailable" before closing
		rc := protocol.ConnackReturnCode(buf[3])
		assert.Equal(t, protocol.ConnRefusedUnavailable, rc)
	}
	// Either way, the 6th client is not accepted

	// Original 5 clients can still publish
	for i := 0; i < 5; i++ {
		publishMessage(t, conns[i], "t/1", []byte(`{"i":`+string(rune('0'+i))+`}`), 0, 0)
	}

	workerMsgs := waitForWorkerMessages(t, b.worker, 5, 5*time.Second)
	assert.GreaterOrEqual(t, len(workerMsgs), 5)

	ctx := context.Background()
	dbMsgs, err := b.store.GetByTopic(ctx, "t/1")
	require.NoError(t, err)
	assert.Len(t, dbMsgs, 5)
}

func TestE2E_AuthFailureDoesNotPollutePipeline(t *testing.T) {
	b := setupBroker(t, []string{"t/1"})

	// Bad auth
	conn, rc := connectClientRaw(t, b.addr, "bad-client", "admin", "wrong-password")
	assert.Equal(t, protocol.ConnRefusedBadAuth, rc)
	conn.Close()

	// Give time for any accidental pipeline activity
	time.Sleep(200 * time.Millisecond)

	// Verify nothing in DB or worker
	ctx := context.Background()
	dbMsgs, err := b.store.GetByTopic(ctx, "t/1")
	require.NoError(t, err)
	assert.Empty(t, dbMsgs)
	assert.Empty(t, b.worker.messages())

	// Good client works fine after
	conn2 := connectClient(t, b.addr, "good-client", "admin", "secret")
	defer conn2.Close()
	publishMessage(t, conn2, "t/1", []byte(`{"ok":true}`), 0, 0)

	workerMsgs := waitForWorkerMessages(t, b.worker, 1, 5*time.Second)
	assert.Len(t, workerMsgs, 1)

	dbMsgs, err = b.store.GetByTopic(ctx, "t/1")
	require.NoError(t, err)
	assert.Len(t, dbMsgs, 1)
}

func TestE2E_GracefulShutdownUnderLoad(t *testing.T) {
	b := setupBroker(t, []string{"t/1"})

	conn := connectClient(t, b.addr, "loader", "admin", "secret")
	defer conn.Close()

	// Publish 5 messages
	for i := 0; i < 5; i++ {
		publishMessage(t, conn, "t/1", []byte(`{"seq":`+string(rune('0'+i))+`}`), 0, 0)
	}

	// Wait for at least some messages to be processed
	waitForWorkerMessages(t, b.worker, 5, 5*time.Second)

	// Cancel context (graceful shutdown)
	b.cancel()

	// Give time for shutdown
	time.Sleep(200 * time.Millisecond)

	// All 5 messages should be in the DB
	ctx := context.Background()
	dbMsgs, err := b.store.GetByTopic(ctx, "t/1")
	require.NoError(t, err)
	assert.Len(t, dbMsgs, 5)
}

func TestE2E_DisallowedTopicNotPersisted(t *testing.T) {
	b := setupBroker(t, []string{"allowed/topic"})

	conn := connectClient(t, b.addr, "device-01", "admin", "secret")
	defer conn.Close()

	// Publish to allowed topic
	publishMessage(t, conn, "allowed/topic", []byte(`{"ok":1}`), 0, 0)

	// Publish to disallowed topic
	publishMessage(t, conn, "forbidden/topic", []byte(`{"bad":1}`), 0, 0)

	// Wait for the allowed message to arrive
	waitForWorkerMessages(t, b.worker, 1, 5*time.Second)

	// Give extra time for any accidental processing of the disallowed message
	time.Sleep(200 * time.Millisecond)

	ctx := context.Background()

	// Allowed topic: 1 message
	allowedMsgs, err := b.store.GetByTopic(ctx, "allowed/topic")
	require.NoError(t, err)
	assert.Len(t, allowedMsgs, 1)
	assert.Equal(t, []byte(`{"ok":1}`), allowedMsgs[0].Payload)

	// Disallowed topic: 0 messages
	forbiddenMsgs, err := b.store.GetByTopic(ctx, "forbidden/topic")
	require.NoError(t, err)
	assert.Empty(t, forbiddenMsgs)

	// Client still connected — publish another allowed message
	publishMessage(t, conn, "allowed/topic", []byte(`{"ok":2}`), 0, 0)
	waitForWorkerMessages(t, b.worker, 2, 5*time.Second)

	allowedMsgs, err = b.store.GetByTopic(ctx, "allowed/topic")
	require.NoError(t, err)
	assert.Len(t, allowedMsgs, 2)
}
