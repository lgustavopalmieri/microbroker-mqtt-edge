# Foundation Document: MQTT Protocol and Micro Broker Features

## 1. Project Overview

This project implements a micro MQTT broker **from scratch**, built directly on native TCP in Go, designed for edge computing in Industry 4.0. Unlike generic brokers like Mosquitto or HiveMQ, this broker is purposefully limited and specialized:

- Accepts maximum **5 simultaneous clients**
- Accepts maximum **5 topics** (defined by environment variable)
- Clients can **only publish** (publish-only)
- Authentication via **username and password**
- Persistence in **SQLite** with native FIFO queues
- Worker pipeline for data **forwarding** (Kafka, S3, REST, etc.)

## 2. MQTT 3.1.1 Protocol — Byte-by-Byte Analysis

Reference: [OASIS MQTT v3.1.1 Specification](https://docs.oasis-open.org/mqtt/mqtt/v3.1.1/os/mqtt-v3.1.1-os.html)

### 2.1 Control Packet Structure

Every MQTT packet is composed of up to 3 parts, always in this order:

```
┌──────────────────────────┐
│  Fixed Header (mandatory)   │  ← Present in ALL packets
├──────────────────────────┤
│  Variable Header (optional) │  ← Present in SOME packets
├──────────────────────────┤
│  Payload (optional)         │  ← Present in SOME packets
└──────────────────────────┘
```

### 2.2 Fixed Header

```
Byte 1:
  Bits [7-4]: Packet Type (4 bits)
  Bits [3-0]: Flags specific to type

Byte 2+: Remaining Length (variable-length encoding)
  - Each byte uses 7 bits for value + 1 bit for continuation
  - Maximum of 4 bytes → up to 268,435,455 bytes (256 MB)
```

Remaining Length decoding algorithm in Go:

```go
func decodeRemainingLength(reader io.Reader) (int, error) {
    multiplier := 1
    value := 0
    buf := make([]byte, 1)

    for {
        if _, err := io.ReadFull(reader, buf); err != nil {
            return 0, err
        }
        encodedByte := buf[0]
        value += int(encodedByte&0x7F) * multiplier
        multiplier *= 128

        if multiplier > 128*128*128*128 {
            return 0, errors.New("malformed remaining length")
        }
        if encodedByte&0x80 == 0 {
            break
        }
    }
    return value, nil
}
```

Encoding algorithm:

```go
func encodeRemainingLength(length int) []byte {
    var encoded []byte
    for {
        encodedByte := byte(length % 128)
        length /= 128
        if length > 0 {
            encodedByte |= 0x80
        }
        encoded = append(encoded, encodedByte)
        if length == 0 {
            break
        }
    }
    return encoded
}
```

### 2.3 Control Packet Types

For our broker, we need to implement only a subset of the protocol:

| Type | Value | Direction | Required? | Description |
|------|-------|-----------|-----------|-------------|
| CONNECT | 1 | Client → Server | ✅ YES | Connection request |
| CONNACK | 2 | Server → Client | ✅ YES | Connection response |
| PUBLISH | 3 | Client → Server | ✅ YES | Message publication |
| PUBACK | 4 | Server → Client | ✅ YES (QoS 1) | Publication acknowledgment |
| SUBSCRIBE | 8 | Client → Server | ⚠️ PARTIAL | Our broker auto-subscribes internally |
| SUBACK | 9 | Server → Client | ⚠️ PARTIAL | Subscribe response |
| PINGREQ | 12 | Client → Server | ✅ YES | Keep-alive request |
| PINGRESP | 13 | Server → Client | ✅ YES | Keep-alive response |
| DISCONNECT | 14 | Client → Server | ✅ YES | Clean disconnection |

Packets we don't need to implement initially:
- PUBREC/PUBREL/PUBCOMP (QoS 2 — not necessary for our use case)
- UNSUBSCRIBE/UNSUBACK (clients don't subscribe externally)

### 2.4 CONNECT Packet — Complete Breakdown

This is the most complex packet and the first one the client sends.

```
Fixed Header:
  Byte 1: 0x10 (type=1, flags=0000)
  Byte 2+: Remaining Length

Variable Header (10 fixed bytes):
  Bytes 1-2: Protocol Name Length = 0x00, 0x04
  Bytes 3-6: "MQTT" (0x4D, 0x51, 0x54, 0x54)
  Byte 7:    Protocol Level = 0x04 (v3.1.1)
  Byte 8:    Connect Flags
             Bit 7: Username Flag
             Bit 6: Password Flag
             Bit 5: Will Retain
             Bits 4-3: Will QoS
             Bit 2: Will Flag
             Bit 1: Clean Session
             Bit 0: Reserved (must be 0)
  Bytes 9-10: Keep Alive (uint16, big-endian, in seconds)

Payload (in order):
  1. Client Identifier (UTF-8 string, mandatory)
  2. Will Topic (if Will Flag = 1)
  3. Will Message (if Will Flag = 1)
  4. Username (if Username Flag = 1)
  5. Password (if Password Flag = 1)
```

Example parsing in Go:

```go
type ConnectPacket struct {
    ProtocolName  string
    ProtocolLevel byte
    ConnectFlags  byte
    KeepAlive     uint16
    ClientID      string
    WillTopic     string
    WillMessage   []byte
    Username      string
    Password      []byte
}

func (p *ConnectPacket) HasUsername() bool  { return p.ConnectFlags&0x80 != 0 }
func (p *ConnectPacket) HasPassword() bool  { return p.ConnectFlags&0x40 != 0 }
func (p *ConnectPacket) HasWill() bool      { return p.ConnectFlags&0x04 != 0 }
func (p *ConnectPacket) CleanSession() bool { return p.ConnectFlags&0x02 != 0 }
func (p *ConnectPacket) WillQoS() byte      { return (p.ConnectFlags >> 3) & 0x03 }
func (p *ConnectPacket) WillRetain() bool   { return p.ConnectFlags&0x20 != 0 }
```

### 2.5 CONNACK Packet

```
Fixed Header:
  Byte 1: 0x20 (type=2, flags=0000)
  Byte 2: 0x02 (remaining length = 2)

Variable Header:
  Byte 1: Connect Acknowledge Flags
           Bits 7-1: Reserved (0)
           Bit 0: Session Present
  Byte 2: Return Code
           0x00 = Connection Accepted
           0x01 = Unacceptable protocol version
           0x02 = Identifier rejected
           0x03 = Server unavailable
           0x04 = Bad username or password
           0x05 = Not authorized
```

```go
func buildConnack(sessionPresent bool, returnCode byte) []byte {
    packet := []byte{0x20, 0x02}
    if sessionPresent {
        packet = append(packet, 0x01)
    } else {
        packet = append(packet, 0x00)
    }
    packet = append(packet, returnCode)
    return packet
}
```

### 2.6 PUBLISH Packet

```
Fixed Header:
  Byte 1: 0x3X
    Bits [7-4]: 0011 (type=3)
    Bit 3: DUP flag
    Bits 2-1: QoS level (00=QoS0, 01=QoS1, 10=QoS2)
    Bit 0: RETAIN flag
  Byte 2+: Remaining Length

Variable Header:
  Topic Name (UTF-8 string: 2 bytes length + string)
  Packet Identifier (2 bytes, only if QoS > 0)

Payload:
  Application Message (remaining bytes)
```

```go
type PublishPacket struct {
    DUP       bool
    QoS       byte
    Retain    bool
    TopicName string
    PacketID  uint16 // only if QoS > 0
    Payload   []byte
}

func parsePublishFlags(firstByte byte) (dup bool, qos byte, retain bool) {
    dup = firstByte&0x08 != 0
    qos = (firstByte >> 1) & 0x03
    retain = firstByte&0x01 != 0
    return
}
```

### 2.7 PUBACK Packet (QoS 1)

```
Fixed Header:
  Byte 1: 0x40 (type=4, flags=0000)
  Byte 2: 0x02 (remaining length = 2)

Variable Header:
  Bytes 1-2: Packet Identifier (same as PUBLISH)
```

```go
func buildPuback(packetID uint16) []byte {
    return []byte{
        0x40, 0x02,
        byte(packetID >> 8),
        byte(packetID & 0xFF),
    }
}
```

### 2.8 PINGREQ / PINGRESP

```
PINGREQ:  [0xC0, 0x00]  (2 bytes, no variable header, no payload)
PINGRESP: [0xD0, 0x00]  (2 bytes, no variable header, no payload)
```

### 2.9 DISCONNECT

```
DISCONNECT: [0xE0, 0x00]  (2 bytes, no variable header, no payload)
```

### 2.10 Reading UTF-8 Strings (MQTT standard)

All strings in MQTT are prefixed with 2 bytes of length (big-endian):

```go
func readUTF8String(reader io.Reader) (string, error) {
    lenBuf := make([]byte, 2)
    if _, err := io.ReadFull(reader, lenBuf); err != nil {
        return "", err
    }
    length := int(binary.BigEndian.Uint16(lenBuf))
    strBuf := make([]byte, length)
    if _, err := io.ReadFull(reader, strBuf); err != nil {
        return "", err
    }
    return string(strBuf), nil
}

func writeUTF8String(s string) []byte {
    length := len(s)
    buf := make([]byte, 2+length)
    binary.BigEndian.PutUint16(buf, uint16(length))
    copy(buf[2:], s)
    return buf
}
```


## 3. Features — Detailed Analysis and Design Decisions

### 3.1 Feature: Native TCP Connection with Authentication

**Requirements:**
- TCP listener on configurable port (default 1883)
- Maximum of 5 simultaneous clients
- Authentication via username/password in CONNECT packet
- Credentials defined by environment variable
- Client must maintain stable connection (keep-alive)
- 6th client is silently rejected (TCP connection closed)

**Connection Flow:**

```
Client                          Broker
  │                               │
  │──── TCP Connect ─────────────>│  1. Accept TCP connection
  │                               │  2. Check client limit (≤5)
  │                               │     If full → close TCP immediately
  │──── CONNECT Packet ──────────>│  3. Parse CONNECT
  │                               │  4. Validate protocol name/level
  │                               │  5. Validate username/password
  │<──── CONNACK (0x00) ─────────│  6. Accept → CONNACK success
  │                               │     Reject → CONNACK error + close
  │                               │
  │──── PINGREQ ─────────────────>│  7. Periodic keep-alive
  │<──── PINGRESP ───────────────│
  │                               │
  │──── DISCONNECT ──────────────>│  8. Clean disconnection
  │                               │     Free client slot
```

**Implementation decisions:**

1. Connection control uses `sync.Mutex` + atomic counter for thread-safety
2. Each accepted connection spawns a dedicated goroutine for reading
3. Keep-alive timeout: if client sends nothing in 1.5x keep-alive, we disconnect
4. Credentials: `BROKER_USERNAME` and `BROKER_PASSWORD` as env vars

```go
// Example of connection control
type ConnectionManager struct {
    mu         sync.Mutex
    clients    map[string]*Client
    maxClients int
}

func (cm *ConnectionManager) CanAccept() bool {
    cm.mu.Lock()
    defer cm.mu.Unlock()
    return len(cm.clients) < cm.maxClients
}

func (cm *ConnectionManager) Add(client *Client) error {
    cm.mu.Lock()
    defer cm.mu.Unlock()
    if len(cm.clients) >= cm.maxClients {
        return ErrMaxClientsReached
    }
    cm.clients[client.ID] = client
    return nil
}

func (cm *ConnectionManager) Remove(clientID string) {
    cm.mu.Lock()
    defer cm.mu.Unlock()
    delete(cm.clients, clientID)
}
```

### 3.2 Feature: Configurable Topics

**Requirements:**
- Topics defined by environment variable
- Maximum of 5 topics
- Clients can ONLY publish to defined topics
- Publication to disallowed topic → silently reject (or close connection)

**Configuration via env:**

```bash
# Format: comma-separated list
BROKER_TOPICS="machine/status,machine/production,machine/alarm,machine/oee,machine/counter"
```

**Topic validation:**

```go
type TopicRegistry struct {
    allowedTopics map[string]struct{}
}

func NewTopicRegistry(topics []string) (*TopicRegistry, error) {
    if len(topics) == 0 || len(topics) > 5 {
        return nil, fmt.Errorf("topic count must be between 1 and 5, got %d", len(topics))
    }
    allowed := make(map[string]struct{}, len(topics))
    for _, t := range topics {
        t = strings.TrimSpace(t)
        if t == "" {
            return nil, fmt.Errorf("empty topic name not allowed")
        }
        allowed[t] = struct{}{}
    }
    return &TopicRegistry{allowedTopics: allowed}, nil
}

func (tr *TopicRegistry) IsAllowed(topic string) bool {
    _, ok := tr.allowedTopics[topic]
    return ok
}

func (tr *TopicRegistry) Topics() []string {
    topics := make([]string, 0, len(tr.allowedTopics))
    for t := range tr.allowedTopics {
        topics = append(topics, t)
    }
    return topics
}
```

### 3.3 Feature: Data Ingestion with FIFO Queues

**Requirements:**
- 1 FIFO queue per topic (maximum 5 queues)
- Each data received via PUBLISH enters the queue of its respective topic
- Sequential processing per queue (guarantees order)
- After saving to SQLite, data goes to worker channel

**Queue Architecture:**

```
PUBLISH (topic=machine/status)  ──→  FIFO Queue [topic: machine/status]
PUBLISH (topic=machine/alarm)   ──→  FIFO Queue [topic: machine/alarm]
PUBLISH (topic=machine/oee)     ──→  FIFO Queue [topic: machine/oee]
...

Each queue has 1 consumer goroutine:
  Queue → Consumer → SQLite (with lock) → Worker Channel
```

**Native FIFO queue implementation:**

```go
// Message represents data received from the broker
type Message struct {
    ClientID  string
    Topic     string
    Payload   []byte
    Timezone  string
    Timestamp time.Time
}

// TopicQueue is a thread-safe FIFO queue for a topic
type TopicQueue struct {
    topic    string
    messages chan Message
    done     chan struct{}
}

func NewTopicQueue(topic string, bufferSize int) *TopicQueue {
    return &TopicQueue{
        topic:    topic,
        messages: make(chan Message, bufferSize),
        done:     make(chan struct{}),
    }
}

func (q *TopicQueue) Enqueue(msg Message) {
    q.messages <- msg // blocks if buffer full (backpressure)
}

// StartConsumer processes messages sequentially
func (q *TopicQueue) StartConsumer(ctx context.Context, store Store, workerChan chan<- Message) {
    go func() {
        defer close(q.done)
        for {
            select {
            case <-ctx.Done():
                return
            case msg, ok := <-q.messages:
                if !ok {
                    return
                }
                // 1. Save to SQLite with lock
                if err := store.SaveRawData(ctx, msg); err != nil {
                    // log error, but don't lose the message
                    // retry or dead-letter queue
                    continue
                }
                // 2. Forward to workers
                workerChan <- msg
            }
        }
    }()
}
```

### 3.4 Feature: SQLite Persistence (raw_data)

**Requirements:**
- `raw_data` table with columns: client, topic, timezone, timestamp, payload (JSON)
- Lock on table during transaction (SQLite does this natively with WAL mode)
- Guarantee that data was saved before consuming next from queue
- 5 concurrent queues writing → serialization via mutex

**Schema:**

```sql
CREATE TABLE IF NOT EXISTS raw_data (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    client    TEXT    NOT NULL,
    topic     TEXT    NOT NULL,
    timezone  TEXT    NOT NULL,
    timestamp TEXT    NOT NULL,  -- ISO 8601
    payload   TEXT    NOT NULL,  -- JSON
    created_at TEXT   NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_raw_data_topic ON raw_data(topic);
CREATE INDEX idx_raw_data_timestamp ON raw_data(timestamp);
```

**Store implementation with lock:**

```go
type SQLiteStore struct {
    db *sql.DB
    mu sync.Mutex // ensures serialization between 5 queues
}

func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
    db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
    if err != nil {
        return nil, fmt.Errorf("opening sqlite: %w", err)
    }
    // SQLite with WAL mode allows concurrent reads
    // but writes are serialized — our mutex ensures this explicitly
    db.SetMaxOpenConns(1) // SQLite doesn't support multiple simultaneous writes
    return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) SaveRawData(ctx context.Context, msg Message) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("begin tx: %w", err)
    }
    defer tx.Rollback()

    _, err = tx.ExecContext(ctx,
        `INSERT INTO raw_data (client, topic, timezone, timestamp, payload)
         VALUES (?, ?, ?, ?, ?)`,
        msg.ClientID,
        msg.Topic,
        msg.Timezone,
        msg.Timestamp.Format(time.RFC3339Nano),
        string(msg.Payload),
    )
    if err != nil {
        return fmt.Errorf("insert raw_data: %w", err)
    }

    return tx.Commit()
}
```

**Note on CGO and SQLite:**
- `mattn/go-sqlite3` requires CGO — acceptable for edge computing
- Alternative: `modernc.org/sqlite` (pure Go, no CGO) — better for cross-compilation
- Decision: use `modernc.org/sqlite` to maintain zero C dependencies

### 3.5 Feature: Workers (Forwarding Pipeline)

**Requirements:**
- After saving to SQLite, data goes to a shared channel
- Multiple workers consume from this channel simultaneously
- Workers are plugins: S3, Kafka, REST, WebSocket, calculations (OEE, counting)
- Fan-out architecture: each data is sent to ALL registered workers

**Fan-Out Pattern:**

```
                    ┌──→ Worker S3
                    │
SQLite → Channel ─────┼──→ Worker Kafka
                    │
                    ┼──→ Worker REST
                    │
                    └──→ Worker OEE Calculator
```

**Worker Interface:**

```go
// Worker defines the contract for any forwarding plugin
type Worker interface {
    // Name returns the identifier name of the worker
    Name() string
    // Process receives an already-persisted message and processes it
    Process(ctx context.Context, msg Message) error
    // Close releases worker resources
    Close() error
}

// WorkerDispatcher distributes messages to all registered workers
type WorkerDispatcher struct {
    workers []Worker
    input   <-chan Message
    logger  Logger
}

func NewWorkerDispatcher(input <-chan Message, workers []Worker, logger Logger) *WorkerDispatcher {
    return &WorkerDispatcher{
        workers: workers,
        input:   input,
        logger:  logger,
    }
}

func (d *WorkerDispatcher) Start(ctx context.Context) {
    go func() {
        for {
            select {
            case <-ctx.Done():
                return
            case msg, ok := <-d.input:
                if !ok {
                    return
                }
                // Fan-out: send to all workers
                var wg sync.WaitGroup
                for _, w := range d.workers {
                    wg.Add(1)
                    go func(worker Worker) {
                        defer wg.Done()
                        if err := worker.Process(ctx, msg); err != nil {
                            d.logger.Error("worker failed",
                                "worker", worker.Name(),
                                "topic", msg.Topic,
                                "error", err,
                            )
                        }
                    }(w)
                }
                wg.Wait() // wait for all to process before next
            }
        }
    }()
}
```

## 4. Environment Variables

```bash
# TCP Server
BROKER_HOST=0.0.0.0
BROKER_PORT=1883

# Authentication
BROKER_USERNAME=machine01
BROKER_PASSWORD=secret123

# Topics (maximum 5, comma-separated)
BROKER_TOPICS=machine/status,machine/production,machine/alarm,machine/oee,machine/counter

# Limits
BROKER_MAX_CLIENTS=5
BROKER_QUEUE_BUFFER_SIZE=10000

# SQLite
BROKER_DB_PATH=./data/broker.db

# Default timezone
BROKER_TIMEZONE=America/Sao_Paulo
```

## 5. Complete Data Flow

```
┌─────────────┐     TCP      ┌──────────────────┐
│  Device/     │─────────────>│  TCP Listener    │
│  Node-RED/   │   CONNECT    │  (net.Listen)    │
│  MQTT Client │              └────────┬─────────┘
└─────────────┘                        │
                                       ▼
                              ┌──────────────────┐
                              │  Connection      │
                              │  Manager         │
                              │  (max 5 clients) │
                              └────────┬─────────┘
                                       │ PUBLISH
                                       ▼
                              ┌──────────────────┐
                              │  Topic Registry  │
                              │  (validates topic)│
                              └────────┬─────────┘
                                       │
                                       ▼
                              ┌──────────────────┐
                              │  Topic Queue     │
                              │  (FIFO per       │
                              │   topic, max 5)  │
                              └────────┬─────────┘
                                       │ sequential
                                       ▼
                              ┌──────────────────┐
                              │  SQLite Store    │
                              │  (raw_data)      │
                              │  with lock/tx    │
                              └────────┬─────────┘
                                       │
                                       ▼
                              ┌──────────────────┐
                              │  Worker          │
                              │  Dispatcher      │
                              │  (fan-out)       │
                              └────────┬─────────┘
                                       │
                          ┌────────────┼────────────┐
                          ▼            ▼            ▼
                     ┌────────┐  ┌────────┐  ┌────────┐
                     │ Worker │  │ Worker │  │ Worker │
                     │ S3     │  │ Kafka  │  │ OEE    │
                     └────────┘  └────────┘  └────────┘
```

## 6. Important Technical Decisions

### 6.1 QoS Level

For our use case (industrial edge computing), we will implement:
- **QoS 0** (at most once): for high-frequency data where occasional loss is acceptable
- **QoS 1** (at least once): for critical data like alarms and piece counting

We will NOT implement QoS 2 (exactly once) — the complexity of the 4-packet handshake doesn't justify the benefit in our scenario, since SQLite persistence already guarantees durability.

### 6.2 Why not use SUBSCRIBE externally?

In our model, the broker is its own consumer. Clients only publish data. The broker internally "subscribes" to all configured topics and processes the data. This greatly simplifies implementation and eliminates the need to manage subscription trees.

However, we should respond to SUBSCRIBE packets that clients send (some MQTT clients automatically send SUBSCRIBE after CONNECT). The response will be a SUBACK with the requested QoS, but internally we don't need to maintain subscription state.

### 6.3 Backpressure

If SQLite can't keep up with ingestion rate:
1. The FIFO queue channel fills (configurable buffer)
2. `Enqueue` blocks
3. PUBLISH handler blocks
4. Client TCP buffer fills
5. Client perceives slowness

This is natural and desirable backpressure. The queue buffer (`BROKER_QUEUE_BUFFER_SIZE`) should be sized to absorb spikes.

### 6.4 Graceful Shutdown

The broker should perform graceful shutdown:
1. Stop accepting new TCP connections
2. Send DISCONNECT to all connected clients
3. Drain all FIFO queues (process pending messages)
4. Wait for workers to finish
5. Close SQLite connection
6. Exit

```go
func gracefulShutdown(ctx context.Context, cancel context.CancelFunc, server *Server) {
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

    <-sigCh
    log.Println("Shutting down gracefully...")
    cancel() // signal all goroutines via context

    // Wait with timeout
    shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer shutdownCancel()

    if err := server.Shutdown(shutdownCtx); err != nil {
        log.Printf("Forced shutdown: %v", err)
    }
}
```

## 7. References

- [MQTT v3.1.1 OASIS Standard](https://docs.oasis-open.org/mqtt/mqtt/v3.1.1/os/mqtt-v3.1.1-os.html) — Official specification
- [Eclipse Mosquitto](https://github.com/eclipse-mosquitto/mosquitto) — Reference broker in C
- [Mochi MQTT](https://github.com/mochi-mqtt/server) — Embeddable broker in Go (architecture reference)
- [Sol MQTT Broker](https://github.com/codepr/sol) — Minimalist broker in C (low-level reference)
- [Solace MQTT 3.1.1 Conformance Spec](https://docs.solace.com/API/MQTT-311-Prtl-Conformance-Spec/Table%20of%20Contents.htm) — Byte-by-byte breakdown

Content was rephrased for compliance with licensing restrictions.
