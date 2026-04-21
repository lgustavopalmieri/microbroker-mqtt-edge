# Documento de Fundação: Arquitetura do Micro Broker MQTT

## 1. Filosofia Arquitetural

Este projeto não é uma aplicação enterprise típica (REST API + banco). É um sistema de baixo nível que lida com protocolo binário TCP, concorrência pesada e processamento de dados em tempo real. A arquitetura precisa equilibrar:

- **Hexagonal/Clean Architecture** → para desacoplamento e testabilidade
- **Arquitetura Modular** → para separação de domínios
- **Pragmatismo** → evitar overengineering em um sistema edge

A regra de ouro: **interfaces para contratos, módulos para domínios, channels para comunicação**.

## 2. Análise de Domínios (DDD Strategic Design)

### 2.1 Bounded Contexts Identificados

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

### 2.2 Justificativa dos Domínios

| Domínio | Responsabilidade | Por que é separado? |
|---------|-----------------|---------------------|
| `protocol` | Parsing e building de pacotes MQTT binários | Lógica puramente técnica, sem regras de negócio. Pode ser reutilizado em qualquer projeto MQTT. Zero dependências externas. |
| `session` | Gerenciamento de conexões TCP, autenticação, keep-alive | Regras de negócio de conexão (max clients, auth). Orquestra o ciclo de vida do client. |
| `ingestion` | Filas FIFO, persistência SQLite, garantia de ordem | Core do negócio: garantir que nenhum dado se perca e que a ordem seja mantida. |
| `dispatch` | Distribuição de dados para workers/plugins | Extensibilidade: novos destinos (Kafka, S3, OEE) sem tocar nos outros módulos. |

### 2.3 Comunicação entre Domínios

```
protocol ──parse──→ session ──message──→ ingestion ──saved──→ dispatch
   ↑                   │                     │                   │
   │                   │                     │                   │
   └── build ──────────┘                     │                   │
       (CONNACK,                             │                   │
        PUBACK,                              ▼                   ▼
        PINGRESP)                        SQLite              Workers
```

A comunicação é unidirecional e usa Go channels:
- `session` → `ingestion`: channel de `Message`
- `ingestion` → `dispatch`: channel de `Message` (após persistência)

## 3. Estrutura de Diretórios

```
microbroker-mqtt-edge/
├── cmd/
│   └── main.go                          # Bootstrap: wiring, config, start
│
├── internal/
│   ├── config/                          # Configuração (env vars)
│   │   └── config.go
│   │
│   ├── modules/
│   │   ├── protocol/                    # Módulo: MQTT Protocol (puro, sem I/O)
│   │   │   ├── packet.go               # Tipos de pacotes e constantes
│   │   │   ├── decoder.go              # Parsing de bytes → structs
│   │   │   ├── encoder.go              # Building de structs → bytes
│   │   │   ├── errors.go               # Erros do protocolo
│   │   │   ├── packet_test.go
│   │   │   ├── decoder_test.go
│   │   │   └── encoder_test.go
│   │   │
│   │   ├── session/                     # Módulo: Connection/Session Management
│   │   │   ├── domain/
│   │   │   │   ├── client.go            # Entidade Client
│   │   │   │   ├── errors.go            # Erros de domínio
│   │   │   │   └── topic_registry.go    # Value Object: registro de tópicos
│   │   │   ├── server.go               # TCP listener + accept loop
│   │   │   ├── handler.go              # Packet handler (orquestra protocol)
│   │   │   ├── auth.go                 # Autenticação
│   │   │   ├── connection_manager.go   # Controle de max clients
│   │   │   ├── interfaces.go           # Ports (contratos)
│   │   │   ├── server_test.go
│   │   │   ├── handler_test.go
│   │   │   ├── auth_test.go
│   │   │   └── connection_manager_test.go
│   │   │
│   │   ├── ingestion/                   # Módulo: Data Ingestion Pipeline
│   │   │   ├── domain/
│   │   │   │   ├── message.go           # Entidade Message
│   │   │   │   └── errors.go           # Erros de domínio
│   │   │   ├── application/
│   │   │   │   ├── queue.go            # Fila FIFO por tópico
│   │   │   │   ├── pipeline.go         # Orquestrador das filas
│   │   │   │   ├── interfaces.go       # Ports (Store interface)
│   │   │   │   ├── queue_test.go
│   │   │   │   └── pipeline_test.go
│   │   │   └── adapters/
│   │   │       └── outbound/
│   │   │           └── database/
│   │   │               ├── repository.go      # Adapter: SQLite implementation
│   │   │               └── repository_test.go
│   │   │
│   │   └── dispatch/                    # Módulo: Worker Dispatch
│   │       ├── domain/
│   │       │   └── worker.go            # Interface Worker
│   │       ├── dispatcher.go            # Fan-out dispatcher
│   │       ├── interfaces.go            # Ports
│   │       ├── dispatcher_test.go
│   │       └── workers/                 # Implementações de workers
│   │           ├── logger_worker.go     # Worker de log (debug/dev)
│   │           └── logger_worker_test.go
│   │
│   ├── platform/                        # Infrastructure adapters
│   │   └── database/                    # Database infrastructure (migrations, connection)
│   │       ├── sqlite.go
│   │       ├── migrator.go
│   │       └── migrations/
│   │           └── 001_create_raw_data.sql
│   │
│   └── common/                          # Shared: interfaces e helpers
│       ├── logger.go                    # Interface de logging
│       └── closer.go                    # Interface Closer genérica
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

## 4. Princípios Arquiteturais Aplicados

### 4.1 Dependency Flow (sempre para dentro)

```
protocol (zero deps)
    ↑
session/domain (depende de protocol)
    ↑
session (depende de session/domain, protocol)
    ↑
ingestion/domain (zero deps externas)
    ↑
ingestion (depende de ingestion/domain)
    ↑
dispatch (depende de dispatch/domain)
    ↑
cmd/main.go (wiring de tudo)
```

Regras rígidas:
- `protocol/` → ZERO imports de outros pacotes internos
- `*/domain/` → ZERO imports de infraestrutura
- `session/` → NÃO importa `ingestion/` diretamente (comunicação via channel)
- `ingestion/` → NÃO importa `dispatch/` diretamente (comunicação via channel)
- `cmd/main.go` → ÚNICO lugar que conhece todos os módulos (wiring)

### 4.2 Interfaces como Contratos

Cada módulo define suas interfaces (ports) em `interfaces.go`:

```go
// internal/modules/ingestion/application/interfaces.go
package ingestion

import "context"

// Store define o contrato de persistência
type Store interface {
    SaveRawData(ctx context.Context, msg domain.Message) error
    Close() error
}

// MessageSource define de onde vêm as mensagens
type MessageSource interface {
    Messages() <-chan domain.Message
}
```

```go
// internal/modules/dispatch/interfaces.go
package dispatch

import "context"

// Worker define o contrato para plugins de forwarding
type Worker interface {
    Name() string
    Process(ctx context.Context, msg Message) error
    Close() error
}
```

### 4.3 Comunicação via Channels (não imports diretos)

Os módulos se comunicam via Go channels, injetados no bootstrap:

```go
// cmd/main.go (simplificado)
func main() {
    // Channels de comunicação entre módulos
    ingestChan := make(chan ingestion.Message, cfg.QueueBufferSize)
    dispatchChan := make(chan ingestion.Message, cfg.QueueBufferSize)

    // Wiring
    server := session.NewServer(cfg, ingestChan)      // session → publica em ingestChan
    pipeline := ingestion.NewPipeline(cfg, store,
                    ingestChan, dispatchChan)           // ingestion → lê de ingestChan, publica em dispatchChan
    dispatcher := dispatch.NewDispatcher(dispatchChan,
                    workers)                            // dispatch → lê de dispatchChan

    // Start
    go pipeline.Start(ctx)
    go dispatcher.Start(ctx)
    server.ListenAndServe(ctx)
}
```

### 4.4 Testabilidade

Cada camada é testável isoladamente:

| Camada | Como testar |
|--------|-------------|
| `protocol/` | Testes unitários puros: bytes in → struct out, struct in → bytes out |
| `session/domain/` | Testes unitários: validações, regras de negócio |
| `session/` | Testes com `net.Pipe()` para simular TCP sem rede real |
| `ingestion/application/queue` | Testes com channels: enqueue/dequeue, ordem, backpressure |
| `ingestion/adapters/outbound/database` | Testes de integração com SQLite in-memory (`:memory:`) |
| `dispatch/` | Testes com mock workers (gomock) |

## 5. Detalhamento dos Módulos

### 5.1 Módulo `protocol` — MQTT Packet Engine

Este é o módulo mais baixo nível. Responsabilidade única: converter bytes ↔ structs.

**Entidades:**

```go
package protocol

// PacketType representa os tipos de control packets MQTT
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

// ConnackReturnCode representa os códigos de retorno do CONNACK
type ConnackReturnCode byte

const (
    ConnAccepted          ConnackReturnCode = 0x00
    ConnRefusedProtocol   ConnackReturnCode = 0x01
    ConnRefusedIdentifier ConnackReturnCode = 0x02
    ConnRefusedUnavailable ConnackReturnCode = 0x03
    ConnRefusedBadAuth    ConnackReturnCode = 0x04
    ConnRefusedNotAuth    ConnackReturnCode = 0x05
)

// FixedHeader é o header presente em todos os pacotes
type FixedHeader struct {
    PacketType      PacketType
    Flags           byte
    RemainingLength int
}

// ConnectPacket representa um CONNECT
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

// PublishPacket representa um PUBLISH
type PublishPacket struct {
    DUP       bool
    QoS       byte
    Retain    bool
    TopicName string
    PacketID  uint16
    Payload   []byte
}

// SubscribePacket representa um SUBSCRIBE
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

// Decoder lê pacotes MQTT de um io.Reader
type Decoder struct {
    reader io.Reader
}

func NewDecoder(reader io.Reader) *Decoder {
    return &Decoder{reader: reader}
}

// ReadPacket lê o próximo pacote completo
func (d *Decoder) ReadPacket() (FixedHeader, []byte, error) {
    // 1. Lê primeiro byte (tipo + flags)
    // 2. Decodifica remaining length
    // 3. Lê remaining bytes
    // 4. Retorna header + payload bytes
}

// DecodeConnect parseia os bytes de um CONNECT packet
func DecodeConnect(data []byte) (*ConnectPacket, error) { ... }

// DecodePublish parseia os bytes de um PUBLISH packet
func DecodePublish(header FixedHeader, data []byte) (*PublishPacket, error) { ... }

// DecodeSubscribe parseia os bytes de um SUBSCRIBE packet
func DecodeSubscribe(data []byte) (*SubscribePacket, error) { ... }
```

**Encoder (building):**

```go
package protocol

// EncodeConnack constrói um CONNACK packet
func EncodeConnack(sessionPresent bool, returnCode ConnackReturnCode) []byte { ... }

// EncodePuback constrói um PUBACK packet
func EncodePuback(packetID uint16) []byte { ... }

// EncodeSuback constrói um SUBACK packet
func EncodeSuback(packetID uint16, returnCodes []byte) []byte { ... }

// EncodePingresp constrói um PINGRESP packet
func EncodePingresp() []byte { return []byte{0xD0, 0x00} }
```

### 5.2 Módulo `session` — Connection Management

Orquestra o ciclo de vida das conexões TCP.

```go
package session

// Server é o TCP listener principal
type Server struct {
    cfg        *config.Config
    listener   net.Listener
    connMgr    *ConnectionManager
    auth       Authenticator
    topics     *domain.TopicRegistry
    msgChan    chan<- ingestion.Message // output para ingestion
    logger     common.Logger
}

// ListenAndServe inicia o servidor TCP
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

        // Verifica limite de clients ANTES de criar goroutine
        if !s.connMgr.CanAccept() {
            conn.Close()
            continue
        }

        go s.handleConnection(ctx, conn)
    }
}
```

**Handler de conexão:**

```go
func (s *Server) handleConnection(ctx context.Context, conn net.Conn) {
    defer conn.Close()

    decoder := protocol.NewDecoder(conn)

    // 1. Espera CONNECT (com timeout)
    header, data, err := decoder.ReadPacket()
    if err != nil || header.PacketType != protocol.CONNECT {
        return
    }

    // 2. Parse CONNECT
    connectPkt, err := protocol.DecodeConnect(data)
    if err != nil {
        return
    }

    // 3. Autentica
    if !s.auth.Authenticate(connectPkt.Username, string(connectPkt.Password)) {
        conn.Write(protocol.EncodeConnack(false, protocol.ConnRefusedBadAuth))
        return
    }

    // 4. Registra client
    client := domain.NewClient(connectPkt.ClientID, conn, connectPkt.KeepAlive)
    if err := s.connMgr.Add(client); err != nil {
        conn.Write(protocol.EncodeConnack(false, protocol.ConnRefusedUnavailable))
        return
    }
    defer s.connMgr.Remove(client.ID)

    // 5. CONNACK success
    conn.Write(protocol.EncodeConnack(false, protocol.ConnAccepted))

    // 6. Loop de leitura de pacotes
    s.readLoop(ctx, client, decoder)
}

func (s *Server) readLoop(ctx context.Context, client *domain.Client, decoder *protocol.Decoder) {
    for {
        // Reset keep-alive deadline
        client.ResetDeadline()

        header, data, err := decoder.ReadPacket()
        if err != nil {
            return // conexão perdida
        }

        switch header.PacketType {
        case protocol.PUBLISH:
            s.handlePublish(ctx, client, header, data)
        case protocol.SUBSCRIBE:
            s.handleSubscribe(client, data)
        case protocol.PINGREQ:
            client.Write(protocol.EncodePingresp())
        case protocol.DISCONNECT:
            return // desconexão limpa
        }
    }
}

func (s *Server) handlePublish(ctx context.Context, client *domain.Client, header protocol.FixedHeader, data []byte) {
    pkt, err := protocol.DecodePublish(header, data)
    if err != nil {
        return
    }

    // Valida tópico
    if !s.topics.IsAllowed(pkt.TopicName) {
        return // ignora silenciosamente
    }

    // Envia para fila de ingestão
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

    // PUBACK se QoS 1
    if pkt.QoS == 1 {
        client.Write(protocol.EncodePuback(pkt.PacketID))
    }
}
```

### 5.3 Módulo `ingestion` — Data Pipeline

```go
package ingestion

// Pipeline orquestra as filas FIFO e a persistência
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

// Start inicia o roteamento de mensagens para filas e os consumers
func (p *Pipeline) Start(ctx context.Context, input <-chan domain.Message) {
    // Inicia consumers (1 por fila)
    for _, q := range p.queues {
        go q.StartConsumer(ctx, p.store, p.dispatchChan, p.logger)
    }

    // Roteia mensagens para a fila correta
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

### 5.4 Módulo `dispatch` — Worker Fan-Out

```go
package dispatch

// Dispatcher distribui mensagens para workers registrados
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

## 6. Padrões de Concorrência Utilizados

### 6.1 Goroutine por Conexão

Cada client TCP tem sua goroutine dedicada. Com máximo de 5 clients, isso é trivial.

### 6.2 Channel Pipeline

```
session goroutines → [ingestChan] → pipeline router → [queue channels] → consumers → [dispatchChan] → dispatcher
```

### 6.3 Fan-Out para Workers

Cada mensagem persistida é enviada para todos os workers simultaneamente via goroutines.

### 6.4 Graceful Shutdown via Context

`context.WithCancel` propagado para todas as goroutines. `signal.Notify` captura SIGINT/SIGTERM.

## 7. Estratégia de Testes

### 7.1 Testes Unitários

| Pacote | O que testar | Técnica |
|--------|-------------|---------|
| `protocol/` | Encode/decode de cada tipo de pacote | Bytes literais → struct → bytes |
| `session/domain/` | Validações de client, topic registry | Table-driven tests |
| `session/auth` | Autenticação válida/inválida | Table-driven tests |
| `session/connection_manager` | Add/Remove/CanAccept, limite de clients | Concurrency tests |
| `ingestion/application/queue` | Enqueue/dequeue, ordem FIFO, backpressure | Channel-based tests |
| `ingestion/adapters/outbound/database` | Insert + query de raw_data | SQLite `:memory:` |
| `dispatch/dispatcher` | Fan-out para N workers, error handling | Mock workers (gomock) |

### 7.2 Testes de Integração

| Cenário | O que testar | Técnica |
|---------|-------------|---------|
| TCP + MQTT | Client conecta, publica, desconecta | `net.Pipe()` ou TCP localhost |
| SQLite | Insert + query de raw_data | SQLite `:memory:` |
| Pipeline completo | Publish → Queue → SQLite → Worker | Channels + mock store |

### 7.3 Exemplo de Teste do Protocol Decoder

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
        // ... mais casos
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

## 8. Evolução Futura (fora do escopo inicial)

Estes itens NÃO serão implementados agora, mas a arquitetura os suporta:

| Feature | Módulo afetado | Como adicionar |
|---------|---------------|----------------|
| Worker Kafka | `dispatch/workers/` | Novo arquivo `kafka_worker.go` implementando `Worker` |
| Worker S3 | `dispatch/workers/` | Novo arquivo `s3_worker.go` implementando `Worker` |
| Worker REST | `dispatch/workers/` | Novo arquivo `rest_worker.go` implementando `Worker` |
| Cálculo OEE | `dispatch/workers/` | Worker que acumula estado e calcula OEE |
| Contagem de peças | `dispatch/workers/` | Worker com contador atômico por tópico |
| WebSocket | `dispatch/workers/` | Worker que publica em hub WebSocket |
| TLS/SSL | `session/` | `tls.NewListener` wrapping o `net.Listener` |
| Métricas Prometheus | `common/` | Interface de métricas + adapter Prometheus |

## 9. Configuração e Bootstrap

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

    // 3. Context com cancelamento
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
        // workers registrados aqui
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

    // 11. Aguarda sinal
    <-sigCh
    slog.Info("shutting down...")
    cancel()

    // 12. Cleanup
    server.Close()
}
```

## 10. Resumo das Decisões

| Decisão | Escolha | Justificativa |
|---------|---------|---------------|
| Protocolo | MQTT 3.1.1 | Mais amplamente suportado por devices industriais |
| QoS | 0 e 1 apenas | QoS 2 é complexo demais para o benefício no edge |
| SQLite driver | `modernc.org/sqlite` | Pure Go, sem CGO, cross-compilation fácil |
| Comunicação entre módulos | Go channels | Nativo, zero overhead, backpressure natural |
| Concorrência | 1 goroutine/client + 1 goroutine/fila | Simples, previsível, máximo 10 goroutines de trabalho |
| Logging | `log/slog` (stdlib) | Zero dependências externas |
| Testes | `testing` + `testify` + `gomock` | Padrão do projeto |
| Arquitetura | Modular com interfaces | Meio-termo entre hexagonal e pragmatismo |
