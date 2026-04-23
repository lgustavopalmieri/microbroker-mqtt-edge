# Foundation Document: Micro Broker MQTT Architecture

## 1. Architectural Philosophy

This project is not a typical enterprise application (REST API + database). It's a low-level system that deals with binary TCP protocol, heavy concurrency, and real-time data processing. The architecture must balance:

- **Hexagonal/Clean Architecture** → for decoupling and testability
- **Modular Architecture** → for domain separation
- **Pragmatism** → avoid overengineering in an edge system

The golden rule: **interfaces for contracts, modules for domains, channels for communication**.

## 2. Domain Analysis (DDD Strategic Design)

### 2.1 Identified Bounded Contexts

```
┌─────────────────────────────────────────────────────────┐
│                    MICRO BROKER MQTT                     │
├──────────────┬──────────────┬──────────────┬────────────┤
│   PROTOCOL   │  CONNECTION  │  INGESTION   │  DISPATCH  │
│   (mqtt)     │  (session)   │  (pipeline)  │  (worker)  │
├──────────────┼──────────────┼──────────────┼────────────┤
│ Packet parse │ TCP listener │ Topic queues │ Worker     │
│ Packet build │ Auth         │ SQLite store │  registry  │
│ Fixed header │ Client mgmt  │ FIFO order   │ Fan-out    │
│ Var header   │ Keep-alive   │ Backpressure │ Plugins    │
│ Payload      │ Max clients  │ Persistence  │ Retry      │
└──────────────┴──────────────┴──────────────┴────────────┘
```

### 2.2 Domain Justification

| Domain | Responsibility | Why separate? |
|--------|-----------------|---------------------|
| `protocol` | Parsing and building MQTT binary packets | Pure technical logic, no business rules. Can be reused in any MQTT project. Zero external dependencies. |
| `session` | TCP connection management, authentication, keep-alive | Business rules for connections (max clients, auth). Orchestrates client lifecycle. |
| `ingestion` | FIFO queues, SQLite persistence, order guarantee | Core of business: ensure no data loss and order is maintained. |
| `dispatch` | Data distribution to workers/plugins | Extensibility: new destinations (Kafka, S3, OEE) without touching other modules. |

### 2.3 Communication Between Domains

```
protocol ──parse──→ session ──message──→ ingestion ──saved──→ dispatch
   ↑                   │                     │                   │
   │                   │                     │                   │
   └── build ──────────┘                     │                   │
       (CONNACK,                             │                   │
        PUBACK,                              ▼                   ▼
        PINGRESP)                        SQLite              Workers
```

Communication is unidirectional and uses Go channels:
- `session` → `ingestion`: channel of `Message`
- `ingestion` → `dispatch`: channel of `Message` (after persistence)

## 3. Directory Structure

```
microbroker-mqtt-edge/
├── cmd/
│   └── main.go                          # Bootstrap: wiring, config, start
│
├── internal/
│   ├── config/                          # Configuration (env vars)
│   │   └── config.go
│   │
│   ├── modules/
│   │   ├── protocol/                    # Module: MQTT Protocol (pure, no I/O)
│   │   │   ├── packet.go               # Packet types and constants
│   │   │   ├── decoder.go              # Parsing bytes → structs
│   │   │   ├── encoder.go              # Building structs → bytes
│   │   │   ├── errors.go               # Protocol errors
│   │   │   ├── packet_test.go
│   │   │   ├── decoder_test.go
│   │   │   └── encoder_test.go
│   │   │
│   │   ├── session/                     # Module: Connection/Session Management
│   │   │   ├── domain/
│   │   │   │   ├── client.go            # Client entity
│   │   │   │   ├── errors.go            # Domain errors
│   │   │   │   └── topic_registry.go    # Value Object: topic registry
│   │   │   ├── server.go               # TCP listener + accept loop
│   │   │   ├── handler.go              # Packet handler (orchestrates protocol)
│   │   │   ├── auth.go                 # Authentication
│   │   │   ├── connection_manager.go   # Max clients control
│   │   │   ├── interfaces.go           # Ports (contracts)
│   │   │   ├── server_test.go
│   │   │   ├── handler_test.go
│   │   │   ├── auth_test.go
│   │   │   └── connection_manager_test.go
│   │   │
│   │   ├── ingestion/                   # Module: Data Ingestion Pipeline
│   │   │   ├── domain/
│   │   │   │   ├── message.go           # Message entity
│   │   │   │   └── errors.go           # Domain errors
│   │   │   ├── application/
│   │   │   │   ├── queue.go            # FIFO queue per topic
│   │   │   │   ├── pipeline.go         # Queue orchestrator
│   │   │   │   ├── interfaces.go       # Ports (Store interface)
│   │   │   │   ├── queue_test.go
│   │   │   │   └── pipeline_test.go
│   │   │   └── adapters/
│   │   │       └── outbound/
│   │   │           └── database/
│   │   │               ├── repository.go      # Adapter: SQLite implementation
│   │   │               └── repository_test.go
│   │   │
│   │   └── dispatch/                    # Module: Worker Dispatch
│   │       ├── domain/
│   │       │   └── worker.go            # Worker interface
│   │       ├── dispatcher.go            # Fan-out dispatcher
│   │       ├── interfaces.go            # Ports
│   │       ├── dispatcher_test.go
│   │       └── workers/                 # Worker implementations
│   │           ├── logger_worker.go     # Log worker (debug/dev)
│   │           └── logger_worker_test.go
│   │
│   ├── platform/                        # Infrastructure adapters
│   │   └── database/                    # Database infrastructure (migrations, connection)
│   │       ├── sqlite.go
│   │       ├── migrator.go
│   │       └── migrations/
│   │           └── 001_create_raw_data.sql
│   │
│   └── common/                          # Shared: interfaces and helpers
│       ├── logger.go                    # Logging interface
│       └── closer.go                    # Generic Closer interface
│
├── docs/
│   ├── 01-mqtt-protocol-and-features.md
│   └── 02-architecture.md
│
├── .env
├── .gitignore
├── Dockerfile
├── docker-compose.yml
├── go.mod
├── go.sum
└── README.md
```

## 4. Architectural Principles Applied

### 4.1 Dependency Flow (always inward)

```
protocol (zero deps)
    ↑
session/domain (depends on protocol)
    ↑
session (depends on session/domain, protocol)
    ↑
ingestion/domain (zero external deps)
    ↑
ingestion (depends on ingestion/domain)
    ↑
dispatch (depends on dispatch/domain)
    ↑
cmd/main.go (wiring everything)
```

Strict rules:
- `protocol/` → ZERO imports from other internal packages
- `*/domain/` → ZERO imports from infrastructure
- `session/` → does NOT import `ingestion/` directly (communication via channel)
- `ingestion/` → does NOT import `dispatch/` directly (communication via channel)
- `cmd/main.go` → ONLY place that knows all modules (wiring)

### 4.2 Interfaces as Contracts

Each module defines its interfaces (ports) in `interfaces.go`:

```go
// internal/modules/ingestion/application/interfaces.go
package ingestion

import "context"

// Store defines the persistence contract
type Store interface {
    SaveRawData(ctx context.Context, msg domain.Message) error
    Close() error
}

// MessageSource defines where messages come from
type MessageSource interface {
    Messages() <-chan domain.Message
}
```

```go
// internal/modules/dispatch/interfaces.go
package dispatch

import "context"

// Worker defines the contract for forwarding plugins
type Worker interface {
    Name() string
    Process(ctx context.Context, msg Message) error
    Close() error
}
```

### 4.3 Communication via Channels (not direct imports)

Modules communicate via Go channels, injected at bootstrap:

```go
// cmd/main.go (simplified)
func main() {
    // Communication channels between modules
    ingestChan := make(chan ingestion.Message, cfg.QueueBufferSize)
    dispatchChan := make(chan ingestion.Message, cfg.QueueBufferSize)

    // Wiring
    server := session.NewServer(cfg, ingestChan)      // session → publishes to ingestChan
    pipeline := ingestion.NewPipeline(cfg, store,
                    ingestChan, dispatchChan)           // ingestion → reads from ingestChan, publishes to dispatchChan
    dispatcher := dispatch.NewDispatcher(dispatchChan,
                    workers)                            // dispatch → reads from dispatchChan

    // Start
    go pipeline.Start(ctx)
    go dispatcher.Start(ctx)
    server.ListenAndServe(ctx)
}
```

### 4.4 Testability

Each layer is testable in isolation:

| Layer | How to test |
|-------|-------------|
| `protocol/` | Pure unit tests: bytes in → struct out, struct in → bytes out |
| `session/domain/` | Unit tests: validations, business rules |
| `session/` | Tests with `net.Pipe()` to simulate TCP without real network |
| `ingestion/application/queue` | Tests with channels: enqueue/dequeue, order, backpressure |
| `ingestion/adapters/outbound/database` | Integration tests with SQLite in-memory (`:memory:`) |
| `dispatch/` | Tests with mock workers (gomock) |

## 5. Module Breakdown

### 5.1 Module `protocol` — MQTT Packet Engine

This is the lowest-level module. Single responsibility: convert bytes ↔ structs.

**Entities:**

```go
package protocol

// PacketType represents MQTT control packet types
type PacketType byte

const (
    CONNECT     PacketType = 1
    CONNACK     PacketType = 2
    PUBLISH     PacketType = 3
    PUBACK      PacketType = 4
    SUBSCRIBE   PacketType = 8
    SUBACK      PacketType = 9
    PINGREQ     PacketType = 12
    PINGRESP    PacketType = 13
    DISCONNECT  PacketType = 14
)

// ConnackReturnCode represents CONNACK return codes
type ConnackReturnCode byte

const (
    ConnAccepted          ConnackReturnCode = 0x00
    ConnRefusedProtocol   ConnackReturnCode = 0x01
    ConnRefusedIdentifier ConnackReturnCode = 0x02
    ConnRefusedUnavailable ConnackReturnCode = 0x03
    ConnRefusedBadAuth    ConnackReturnCode = 0x04
    ConnRefusedNotAuth    ConnackReturnCode = 0x05
)

// FixedHeader is the header present in all packets
type FixedHeader struct {
    PacketType      PacketType
    Flags           byte
    RemainingLength int
}

// ConnectPacket represents a CONNECT
type ConnectPacket struct {
    ProtocolName  string
    ProtocolLevel byte
    CleanSession  bool
    HasWill       bool
    WillQoS       byte
    WillRetain    bool
    HasUsername    bool
    HasPassword   bool
    KeepAlive     uint16
    ClientID      string
    WillTopic     string
    WillMessage   []byte
    Username      string
    Password      []byte
}

// PublishPacket represents a PUBLISH
type PublishPacket struct {
    DUP       bool
    QoS       byte
    Retain    bool
    TopicName string
    PacketID  uint16
    Payload   []byte
}

// SubscribePacket represents a SUBSCRIBE
type SubscribePacket struct {
    PacketID      uint16
    Subscriptions []Subscription
}

type Subscription struct {
    TopicFilter string
    QoS         byte
}
```

**Decoder (parsing):**

```go
package protocol

// Decoder reads MQTT packets from an io.Reader
type Decoder struct {
    reader io.Reader
}

func NewDecoder(reader io.Reader) *Decoder {
    return &Decoder{reader: reader}
}

// ReadPacket reads the next complete packet
func (d *Decoder) ReadPacket() (FixedHeader, []byte, error) {
    // 1. Read first byte (type + flags)
    // 2. Decode remaining length
    // 3. Read remaining bytes
    // 4. Return header + payload bytes
}

// DecodeConnect parses bytes of a CONNECT packet
func DecodeConnect(data []byte) (*ConnectPacket, error) { ... }

// DecodePublish parses bytes of a PUBLISH packet
func DecodePublish(header FixedHeader, data []byte) (*PublishPacket, error) { ... }

// DecodeSubscribe parses bytes of a SUBSCRIBE packet
func DecodeSubscribe(data []byte) (*SubscribePacket, error) { ... }
```

**Encoder (building):**

```go
package protocol

// EncodeConnack builds a CONNACK packet
func EncodeConnack(sessionPresent bool, returnCode ConnackReturnCode) []byte { ... }

// EncodePuback builds a PUBACK packet
func EncodePuback(packetID uint16) []byte { ... }

// EncodeSuback builds a SUBACK packet
func EncodeSuback(packetID uint16, returnCodes []byte) []byte { ... }

// EncodePingresp builds a PINGRESP packet
func EncodePingresp() []byte { return []byte{0xD0, 0x00} }
```

### 5.2 Module `session` — Connection Management

Orchestrates the lifecycle of TCP connections.

```go
package session

// Server is the main TCP listener
type Server struct {
    cfg        *config.Config
    listener   net.Listener
    connMgr    *ConnectionManager
    auth       Authenticator
    topics     *domain.TopicRegistry
    msgChan    chan<- ingestion.Message // output to ingestion
    logger     common.Logger
}

// ListenAndServe starts the TCP server
func (s *Server) ListenAndServe(ctx context.Context) error {
    var err error
    s.listener, err = net.Listen("tcp", s.cfg.Address())
    if err != nil {
        return err
    }

    s.logger.Info("broker started", "address", s.cfg.Address())

    for {
        conn, err := s.listener.Accept()
        if err != nil {
            select {
            case <-ctx.Done():
                return nil
            default:
                s.logger.Error("accept error", "error", err)
                continue
            }
        }

        // Check client limit BEFORE spawning goroutine
        if !s.connMgr.CanAccept() {
            conn.Close()
            continue
        }

        go s.handleConnection(ctx, conn)
    }
}
```

**Connection handler:**

```go
func (s *Server) handleConnection(ctx context.Context, conn net.Conn) {
    defer conn.Close()

    decoder := protocol.NewDecoder(conn)

    // 1. Wait for CONNECT (with timeout)
    header, data, err := decoder.ReadPacket()
    if err != nil || header.PacketType != protocol.CONNECT {
        return
    }

    // 2. Parse CONNECT
    connectPkt, err := protocol.DecodeConnect(data)
    if err != nil {
        return
    }

    // 3. Authenticate
    if !s.auth.Authenticate(connectPkt.Username, string(connectPkt.Password)) {
        conn.Write(protocol.EncodeConnack(false, protocol.ConnRefusedBadAuth))
        return
    }

    // 4. Register client
    client := domain.NewClient(connectPkt.ClientID, conn, connectPkt.KeepAlive)
    if err := s.connMgr.Add(client); err != nil {
        conn.Write(protocol.EncodeConnack(false, protocol.ConnRefusedUnavailable))
        return
    }
    defer s.connMgr.Remove(client.ID)

    // 5. CONNACK success
    conn.Write(protocol.EncodeConnack(false, protocol.ConnAccepted))

    // 6. Packet read loop
    s.readLoop(ctx, client, decoder)
}

func (s *Server) readLoop(ctx context.Context, client *domain.Client, decoder *protocol.Decoder) {
    for {
        // Reset keep-alive deadline
        client.ResetDeadline()

        header, data, err := decoder.ReadPacket()
        if err != nil {
            return // connection lost
        }

        switch header.PacketType {
        case protocol.PUBLISH:
            s.handlePublish(ctx, client, header, data)
        case protocol.SUBSCRIBE:
            s.handleSubscribe(client, data)
        case protocol.PINGREQ:
            client.Write(protocol.EncodePingresp())
        case protocol.DISCONNECT:
            return // clean disconnection
        }
    }
}

func (s *Server) handlePublish(ctx context.Context, client *domain.Client, header protocol.FixedHeader, data []byte) {
    pkt, err := protocol.DecodePublish(header, data)
    if err != nil {
        return
    }

    // Validate topic
    if !s.topics.IsAllowed(pkt.TopicName) {
        return // silently ignore
    }

    // Send to ingestion queue
    msg := ingestion.Message{
        ClientID:  client.ID,
        Topic:     pkt.TopicName,
        Payload:   pkt.Payload,
        Timezone:  s.cfg.Timezone,
        Timestamp: time.Now(),
    }

    select {
    case s.msgChan <- msg:
    case <-ctx.Done():
        return
    }

    // PUBACK if QoS 1
    if pkt.QoS == 1 {
        client.Write(protocol.EncodePuback(pkt.PacketID))
    }
}
```

### 5.3 Module `ingestion` — Data Pipeline

```go
package ingestion

// Pipeline orchestrates FIFO queues and persistence
type Pipeline struct {
    queues       map[string]*Queue
    store        Store
    dispatchChan chan<- domain.Message
    logger       common.Logger
}

func NewPipeline(topics []string, store Store, input <-chan domain.Message,
    output chan<- domain.Message, bufSize int, logger common.Logger) *Pipeline {

    queues := make(map[string]*Queue, len(topics))
    for _, topic := range topics {
        queues[topic] = NewQueue(topic, bufSize)
    }

    return &Pipeline{
        queues:       queues,
        store:        store,
        dispatchChan: output,
        logger:       logger,
    }
}

// Start initiates message routing to queues and their consumers
func (p *Pipeline) Start(ctx context.Context, input <-chan domain.Message) {
    // Start consumers (1 per queue)
    for _, q := range p.queues {
        go q.StartConsumer(ctx, p.store, p.dispatchChan, p.logger)
    }

    // Route messages to correct queue
    for {
        select {
        case <-ctx.Done():
            return
        case msg, ok := <-input:
            if !ok {
                return
            }
            if q, exists := p.queues[msg.Topic]; exists {
                q.Enqueue(msg)
            }
        }
    }
}
```

### 5.4 Module `dispatch` — Worker Fan-Out

```go
package dispatch

// Dispatcher distributes messages to registered workers
type Dispatcher struct {
    workers []Worker
    input   <-chan Message
    logger  common.Logger
}

func (d *Dispatcher) Start(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return
        case msg, ok := <-d.input:
            if !ok {
                return
            }
            d.fanOut(ctx, msg)
        }
    }
}

func (d *Dispatcher) fanOut(ctx context.Context, msg Message) {
    var wg sync.WaitGroup
    for _, w := range d.workers {
        wg.Add(1)
        go func(worker Worker) {
            defer wg.Done()
            if err := worker.Process(ctx, msg); err != nil {
                d.logger.Error("worker process failed",
                    "worker", worker.Name(),
                    "error", err,
                )
            }
        }(w)
    }
    wg.Wait()
}
```

## 6. Concurrency Patterns Used

### 6.1 Goroutine per Connection

Each client TCP has its dedicated goroutine. With maximum of 5 clients, this is trivial.

### 6.2 Channel Pipeline

```
session goroutines → [ingestChan] → pipeline router → [queue channels] → consumers → [dispatchChan] → dispatcher
```

### 6.3 Fan-Out to Workers

Each persisted message is sent to all workers simultaneously via goroutines.

### 6.4 Graceful Shutdown via Context

`context.WithCancel` propagated to all goroutines. `signal.Notify` captures SIGINT/SIGTERM.

## 7. Testing Strategy

### 7.1 Unit Tests

| Package | What to test | Technique |
|---------|-------------|---------|
| `protocol/` | Encode/decode of each packet type | Literal bytes → struct → bytes |
| `session/domain/` | Client validations, topic registry | Table-driven tests |
| `session/auth` | Valid/invalid authentication | Table-driven tests |
| `session/connection_manager` | Add/Remove/CanAccept, client limit | Concurrency tests |
| `ingestion/application/queue` | Enqueue/dequeue, FIFO order, backpressure | Channel-based tests |
| `ingestion/adapters/outbound/database` | Insert + query of raw_data | SQLite `:memory:` |
| `dispatch/dispatcher` | Fan-out to N workers, error handling | Mock workers (gomock) |

### 7.2 Integration Tests

| Scenario | What to test | Technique |
|----------|-------------|---------|
| TCP + MQTT | Client connects, publishes, disconnects | `net.Pipe()` or TCP localhost |
| SQLite | Insert + query of raw_data | SQLite `:memory:` |
| Complete pipeline | Publish → Queue → SQLite → Worker | Channels + mock store |

### 7.3 Example Protocol Decoder Test

```go
func TestDecodeConnect(t *testing.T) {
    tests := []struct {
        name     string
        input    []byte
        expected *ConnectPacket
        wantErr  bool
    }{
        {
            name: "valid connect with username and password",
            input: buildConnectBytes("client1", "user", "pass", 60),
            expected: &ConnectPacket{
                ProtocolName:  "MQTT",
                ProtocolLevel: 4,
                CleanSession:  true,
                HasUsername:    true,
                HasPassword:   true,
                KeepAlive:     60,
                ClientID:      "client1",
                Username:      "user",
                Password:      []byte("pass"),
            },
        },
        {
            name:    "invalid protocol name",
            input:   buildInvalidProtocolBytes(),
            wantErr: true,
        },
        // ... more cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := DecodeConnect(tt.input)
            if tt.wantErr {
                assert.Error(t, err)
                return
            }
            require.NoError(t, err)
            assert.Equal(t, tt.expected, got)
        })
    }
}
```

## 8. Future Evolution (out of initial scope)

These items will NOT be implemented now, but the architecture supports them:

| Feature | Affected Module | How to add |
|---------|---------------|----------------|
| Worker Kafka | `dispatch/workers/` | New file `kafka_worker.go` implementing `Worker` |
| Worker S3 | `dispatch/workers/` | New file `s3_worker.go` implementing `Worker` |
| Worker REST | `dispatch/workers/` | New file `rest_worker.go` implementing `Worker` |
| OEE Calculation | `dispatch/workers/` | Worker that accumulates state and calculates OEE |
| Piece Counting | `dispatch/workers/` | Worker with atomic counter per topic |
| WebSocket | `dispatch/workers/` | Worker that publishes to WebSocket hub |
| TLS/SSL | `session/` | `tls.NewListener` wrapping the `net.Listener` |
| Prometheus Metrics | `common/` | Metrics interface + Prometheus adapter |

## 9. Configuration and Bootstrap

```go
// cmd/main.go
package main

import (
    "context"
    "log/slog"
    "os"
    "os/signal"
    "syscall"

    "microbroker-mqtt-edge/internal/config"
    "microbroker-mqtt-edge/internal/modules/dispatch"
    "microbroker-mqtt-edge/internal/modules/ingestion"
    "microbroker-mqtt-edge/internal/modules/ingestion/adapters/outbound/database"
    "microbroker-mqtt-edge/internal/modules/session"
)

func main() {
    // 1. Config
    cfg, err := config.Load()
    if err != nil {
        slog.Error("failed to load config", "error", err)
        os.Exit(1)
    }

    // 2. Logger
    logger := slog.Default()

    // 3. Context with cancellation
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // 4. Signal handling
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

    // 5. SQLite Store
    sqliteStore, err := database.NewSQLiteStore(cfg.DBPath)
    if err != nil {
        slog.Error("failed to create store", "error", err)
        os.Exit(1)
    }
    defer sqliteStore.Close()

    // 6. Channels
    ingestChan := make(chan ingestion.Message, cfg.QueueBufferSize)
    dispatchChan := make(chan ingestion.Message, cfg.QueueBufferSize)

    // 7. Workers
    workers := []dispatch.Worker{
        // workers registered here
    }

    // 8. Pipeline
    pipeline := ingestion.NewPipeline(cfg.Topics, sqliteStore, dispatchChan, cfg.QueueBufferSize, logger)
    go pipeline.Start(ctx, ingestChan)

    // 9. Dispatcher
    dispatcher := dispatch.NewDispatcher(dispatchChan, workers, logger)
    go dispatcher.Start(ctx)

    // 10. Server
    server := session.NewServer(cfg, ingestChan, logger)
    go func() {
        if err := server.ListenAndServe(ctx); err != nil {
            slog.Error("server error", "error", err)
            cancel()
        }
    }()

    // 11. Wait for signal
    <-sigCh
    slog.Info("shutting down...")
    cancel()

    // 12. Cleanup
    server.Close()
}
```

## 10. Summary of Decisions

| Decision | Choice | Justification |
|----------|--------|---------------|
| Protocol | MQTT 3.1.1 | Most widely supported by industrial devices |
| QoS | 0 and 1 only | QoS 2 is too complex for the benefit in edge |
| SQLite driver | `modernc.org/sqlite` | Pure Go, no CGO, easy cross-compilation |
| Inter-module communication | Go channels | Native, zero overhead, natural backpressure |
| Concurrency | 1 goroutine/client + 1 goroutine/queue | Simple, predictable, maximum 10 work goroutines |
| Logging | `log/slog` (stdlib) | Zero external dependencies |
| Testing | `testing` + `testify` + `gomock` | Project standard |
| Architecture | Modular with interfaces | Middle ground between hexagonal and pragmatism |
