# MicroBroker MQTT Edge — Architecture Review & Restructuring Proposal

## 1. Executive Summary

This document presents a deep domain analysis of the `microbroker-mqtt-edge` project, identifies bounded context violations, naming issues, and structural misalignments, and proposes a clean restructuring following DDD Strategic Design and Hexagonal Architecture principles.

**Key finding**: The code quality is excellent — functions are well-scoped, concurrency is handled safely, tests are solid, and the broker works correctly. The issues are purely structural: mixed bounded contexts, misleading module names, and unnecessary coupling between domains.

**Guiding constraint**: Zero logic changes. Only structural reorganization, renaming, and introduction of explicit contracts (interfaces) where needed.

---

## 2. Domain Analysis

### 2.1 Problem Space — What Does This System Do?

This is an **MQTT edge broker** for Industry 4.0 environments. It:

1. Accepts TCP connections from MQTT clients (machines, sensors, PLCs)
2. Authenticates clients and manages their lifecycle
3. Receives PUBLISH messages on allowed topics
4. Persists raw data (audit trail / edge storage)
5. Fans out persisted messages to independent workers (plugins)
6. Exposes an HTTP API for querying persisted data

### 2.2 Concept Extraction

**Entities & Value Objects**:
- `Client` — an authenticated MQTT device connection
- `TopicRegistry` — allowed topic whitelist
- `Message` — a data point from PUBLISH (clientID, topic, payload, timestamp)
- `Record` — a persisted message for audit queries
- `Worker` — a processing plugin that consumes messages
- `Queue` — per-topic FIFO buffer
- `FixedHeader`, `ConnectPacket`, `PublishPacket`, `SubscribePacket` — MQTT wire protocol structures

**Services / Orchestrators**:
- `Server` — TCP listener, connection accept loop
- `ConnectionManager` — tracks active clients with capacity limit
- `EnvAuthenticator` — validates credentials
- `Pipeline` — routes messages to per-topic queues, persists, forwards
- `Dispatcher` — fan-out to workers
- `Handler` (audit) — HTTP REST for querying persisted data
- `Migrator` — database schema management

### 2.3 Language Groups (Ubiquitous Language)

| Language Group | Key Terms | Current Location |
|---|---|---|
| **Connection & Transport** | TCP listener, accept, connection, address, server | `session/server.go` |
| **Authentication** | authenticate, credentials, username, password, CONNECT | `session/auth.go` |
| **Client Lifecycle** | client, keep-alive, deadline, connected, disconnected, max clients | `session/domain/client.go`, `session/connection_manager.go` |
| **Topic Management** | topic, allowed, registry, subscribe, filter | `session/domain/topic_registry.go` |
| **MQTT Protocol** | packet, CONNECT, PUBLISH, SUBSCRIBE, PINGREQ, QoS, encode, decode | `protocol/*` |
| **Message Handling** | message, PUBLISH handler, routing | `session/handler.go` |
| **Ingestion & Persistence** | pipeline, queue, FIFO, persist, store, save, raw data | `ingestion/*` |
| **Fan-out & Processing** | worker, process, fan-out, plugin, dispatch | `dispatch/*` |
| **Audit & Query** | audit, record, query, count, HTTP API | `common/audit/*` |

### 2.4 Identified Subdomains

#### 1. MQTT Protocol (Generic Subdomain)
- **Type**: Generic — standard MQTT v3.1.1 wire protocol implementation
- **Language**: packet, encode, decode, remaining length, QoS, fixed header
- **Concepts**: `FixedHeader`, `ConnectPacket`, `PublishPacket`, `SubscribePacket`, codec, reader
- **Cohesion**: 10/10 ✅
- **Current module**: `protocol/` — **perfectly isolated, no changes needed**

#### 2. Connection (Core Domain)
- **Type**: Core — the TCP server, client lifecycle, and MQTT session handling
- **Language**: server, connection, client, keep-alive, topic, subscribe, publish
- **Concepts**: `Server`, `Client`, `ClientManager`, `TopicRegistry`, `handler`
- **Cohesion**: 4/10 ❌ (currently mixes 3+ bounded contexts under "session")
- **Current module**: `session/` — **needs decomposition**

#### 2b. Auth (Supporting Subdomain)
- **Type**: Supporting — authentication and credential validation, highly extensible
- **Language**: authenticate, credentials, username, password, provider, token
- **Concepts**: `Authenticator` (port), `EnvAuthenticator` (adapter)
- **Cohesion**: 9/10 ✅ (when isolated — today it's a single strategy, but the bounded context is clear)
- **Current module**: embedded in `session/auth.go` — **needs extraction to own module**
- **Growth potential**: External credential stores (Kafka auth, mTLS), refresh tokens for frontend clients, multiple auth providers, token-based auth for HTTP API. This justifies a dedicated bounded context from day one.

#### 3. Ingestion Pipeline (Core Domain)
- **Type**: Core — the data persistence pipeline that guarantees message durability at the edge
- **Language**: pipeline, queue, persist, store, message, FIFO, backpressure
- **Concepts**: `Pipeline`, `Queue`, `Store`, `Message`
- **Cohesion**: 6/10 ⚠️ (domain layer disconnected from application, `Message` duplicated)
- **Current module**: `ingestion/` — **needs flattening and clarification**

#### 4. Fan-out Processing (Core Domain)
- **Type**: Core — the extensible plugin system that processes persisted messages
- **Language**: worker, process, fan-out, plugin, broadcast
- **Concepts**: `Dispatcher` (fan-out engine), `Worker` (plugin contract), `LoggerWorker`
- **Cohesion**: 8/10 ✅ (good isolation, bad naming)
- **Current module**: `dispatch/` — **needs renaming**

#### 5. Audit (Supporting Subdomain)
- **Type**: Supporting — HTTP query API for persisted data
- **Language**: audit, record, query, count, topic
- **Concepts**: `Handler`, `Reader`, `Record`, `SQLiteReader`
- **Cohesion**: 8/10 ✅ (well isolated)
- **Current module**: `common/audit/` — **misplaced, should be a module**


---

## 3. Cohesion Analysis

### 3.1 Cross-Domain Cohesion Matrix

| Domain A | Domain B | Score | Issue | Recommendation |
|---|---|---|---|---|
| Connection (session) | Protocol | 3/10 | ✅ Clean dependency via function calls | Keep as-is — protocol is a utility |
| Connection (session) | Auth | 7/10 | ⚠️ Auth embedded inside session, tight coupling | Extract auth as own module, inject via interface |
| Connection (session) | Ingestion | 2/10 | ❌ `session.Message` → manual bridge → `ingestion.Message` | Shared `Message` value object in common |
| Ingestion | Fan-out (dispatch) | 3/10 | ⚠️ `dispatch/domain.Message = ingestion/domain.Message` (type alias) | Shared `Message` value object in common |
| Audit | Ingestion | 1/10 | ❌ Both read same DB table but have zero code relationship | Audit should be its own module with explicit Reader port |
| Connection (session) | Audit | 0/10 | ✅ No coupling | Keep separated |

### 3.2 Issues Detected

#### Priority: Critical

**Issue 1: "session" é um God Module — mistura 3+ bounded contexts**

- **Location**: `internal/modules/session/`
- **Problem**: O módulo `session` contém:
  - **Transport**: TCP listener, accept loop (`server.go`)
  - **Authentication**: credential validation (`auth.go`)
  - **Client Lifecycle**: connection tracking, capacity management (`connection_manager.go`)
  - **Topic Management**: topic whitelist (`domain/topic_registry.go`)
  - **Message Handling**: PUBLISH/SUBSCRIBE/PING processing (`handler.go`)
  - **Message DTO**: `session.Message` struct que é um conceito de domínio vazando para fora
- **Cohesion**: 4/10
- **Recommendation**: Decompor em contextos claros. O `Server` é o agregado que orquestra, mas cada responsabilidade deve ter fronteira linguística clara.

**Issue 2: "dispatch" é um nome terrível para fan-out de workers**

- **Location**: `internal/modules/dispatch/`
- **Problem**: "Dispatch" sugere roteamento/despacho para um destino específico. O que o módulo faz é **broadcast/fan-out** — cada mensagem vai para TODOS os workers simultaneamente. O nome correto no vocabulário ubíquo seria algo como `processing`, `fanout`, ou `worker-pool`.
- **Cohesion**: 8/10 (o código em si é coeso, o nome é que engana)
- **Recommendation**: Renomear para `processing` — reflete que é o estágio de processamento extensível via workers/plugins.

**Issue 3: `dispatch/domain.Message` é um type alias para `ingestion/domain.Message`**

- **Location**: `internal/modules/dispatch/domain/worker.go`
- **Problem**: `type Message = ingestiondomain.Message` cria acoplamento direto entre dispatch e ingestion. Se ingestion mudar a struct, dispatch quebra. Viola o princípio de que domínios não dependem de outros domínios.
- **Recommendation**: Extrair `Message` como value object compartilhado em `internal/common/domain/` ou cada módulo define seu próprio tipo e a conversão acontece na fronteira.

#### Priority: High

**Issue 4: `Message` duplicada em 3 lugares com campos idênticos**

- **Location**: `session.Message`, `ingestion/domain.Message`, `dispatch/domain.Message` (alias)
- **Problem**: Três definições da mesma struct com campos idênticos (`ClientID`, `Topic`, `Payload`, `Timezone`, `Timestamp`). O `main.go` faz conversão manual entre `session.Message` → `ingestion.Message`.
- **Recommendation**: Uma única `Message` value object em `internal/common/domain/message.go`, importada por todos os módulos.

**Issue 5: Audit está em `common/` mas é um módulo de negócio**

- **Location**: `internal/common/audit/`
- **Problem**: `common/` deveria conter apenas utilitários compartilhados (logger, interfaces, value objects). Audit é um módulo de negócio completo com handler HTTP, reader, e DTOs próprios.
- **Recommendation**: Mover para `internal/modules/audit/`.

**Issue 6: Logger interface duplicada em 5 lugares**

- **Location**: `common.Logger`, `session.Logger`, `session.NopLogger`, `ingestion/application.Logger`, `ingestion/application.NopLogger`, `dispatch.Logger`, `workers.Logger`, `audit.Logger`
- **Problem**: Cada pacote redefine a mesma interface Logger. Isso é idiomático em Go (accept interfaces, return structs), mas quando TODAS as interfaces são idênticas, é ruído desnecessário.
- **Recommendation**: Manter a interface em `common.Logger` como canônica. Cada módulo pode definir a sua se precisar de um subset diferente (como `workers.Logger` que só tem `Info`), mas os que são idênticos devem importar de `common`.

#### Priority: Medium

**Issue 7: Ingestion tem 3 camadas mas o domínio é quase vazio**

- **Location**: `internal/modules/ingestion/`
- **Problem**: A estrutura `domain/` + `application/` + `adapters/outbound/` sugere complexidade que não existe. O domínio tem apenas `Message` (que é compartilhada) e dois erros. A lógica real está toda em `application/` (Pipeline + Queue). A camada de domínio não justifica existir separada.
- **Recommendation**: Simplificar — o domínio pode ser inline no application, ou melhor: `Message` vai para common e os erros ficam no próprio package.

**Issue 8: Bridge manual no `main.go` entre `session.Message` e `ingestion.Message`**

- **Location**: `cmd/main.go` linhas 80-96
- **Problem**: Uma goroutine inteira dedicada a converter `session.Message` → `ingestion.Message` campo a campo. Isso existe porque cada módulo definiu seu próprio Message type.
- **Recommendation**: Com `Message` unificada em common, essa bridge desaparece completamente.

---

## 4. Bounded Context Map — Proposta

### 4.1 Contextos Propostos

```
┌──────────────────────────────────────────────────────────────────────┐
│                          MQTT Edge Broker                            │
│                                                                      │
│  ┌──────────┐    ┌──────┐    ┌───────────┐    ┌──────────┐          │
│  │ protocol │───▶│ auth │───▶│connection │───▶│ingestion │          │
│  │ (generic)│    │(supp)│    │  (core)   │    │  (core)  │          │
│  └──────────┘    └──────┘    └───────────┘    └────┬─────┘          │
│                                                     │    ┌────────┐ │
│                                                     ├───▶│process-│ │
│                                                     │    │  ing   │ │
│                                                     │    │ (core) │ │
│                                                     │    └────────┘ │
│                                                     ▼               │
│                                               ┌──────────┐         │
│                                               │  audit   │         │
│                                               │(support) │         │
│                                               └──────────┘         │
└──────────────────────────────────────────────────────────────────────┘
```

### 4.2 Integração entre Contextos

```
protocol ──(function calls)──▶ connection
    Padrão: Shared Kernel (protocol é biblioteca pura, sem estado)

auth ──(interface injection)──▶ connection
    Padrão: Customer/Supplier — auth é o supplier, connection é o customer
    connection define a port (Authenticator interface), auth fornece a implementação

connection ──(channel: Message)──▶ ingestion
    Padrão: Published Language (Message value object compartilhado)

ingestion ──(channel: Message)──▶ processing
    Padrão: Published Language (mesmo Message value object)

ingestion ──(shared DB)──▶ audit
    Padrão: Shared Database (audit lê o que ingestion escreveu)
```


---

## 5. Proposed Structure

### 5.1 Root Layout

```
.
├── cmd/
│   └── broker/
│       ├── main.go                          # Entrypoint (slim)
│       ├── bootstrap/                       # Wiring — zero business logic
│       │   ├── database.go                  # SQLite connection + migrations
│       │   ├── modules.go                   # Module initialization & wiring
│       │   ├── server.go                    # TCP + HTTP server startup
│       │   └── shutdown.go                  # Graceful shutdown orchestration
│       └── config/
│           ├── config.go                    # Config struct + Load + validate
│           └── config_test.go
├── internal/
│   ├── common/                              # Shared cross-module contracts
│   │   ├── domain/
│   │   │   └── message.go                   # Message value object (single source of truth)
│   │   └── observability/
│   │       ├── logger.go                    # Logger interface + NopLogger + SlogAdapter
│   │       └── logger_test.go
│   ├── modules/                             # Business domain modules
│   │   ├── protocol/                        # MQTT v3.1.1 wire protocol (Generic)
│   │   ├── auth/                            # Authentication & credential validation (Supporting)
│   │   ├── connection/                      # TCP + Client lifecycle (Core)
│   │   ├── ingestion/                       # Persistence pipeline (Core)
│   │   ├── processing/                      # Fan-out worker engine (Core)
│   │   └── audit/                           # HTTP query API (Supporting)
│   └── platform/                            # Infrastructure adapters
│       └── database/
│           ├── sqlite.go
│           ├── migrator.go
│           └── migrations/
│               └── 001_create_raw_data.sql
├── tests/
│   ├── e2e/
│   │   └── broker_e2e_test.go
│   └── k6/
│       └── ...
├── docker-compose.yml
├── Dockerfile
├── go.mod / go.sum
└── .env / .env.example
```

### 5.2 `internal/common/` — Shared Contracts

```
common/
├── domain/
│   └── message.go                  # THE Message value object
│                                   # Used by: connection, ingestion, processing
└── observability/
    ├── logger.go                   # Logger interface (Info, Error, Warn, Debug)
    │                               # + NopLogger (tests)
    │                               # + NewSlogLogger(l *slog.Logger) Logger
    └── logger_test.go
```

**Mudanças**:
- `common/logger.go` → `common/observability/logger.go` (preparado para métricas/tracing futuro)
- `common/audit/` → movido para `modules/audit/`
- Novo: `common/domain/message.go` — elimina duplicação de Message em 3 módulos

**`common/domain/message.go`**:
```go
package domain

import "time"

// Message represents a data point received from an MQTT client via PUBLISH.
// This is the canonical value object shared across module boundaries.
type Message struct {
    ClientID  string
    Topic     string
    Payload   []byte
    Timezone  string
    Timestamp time.Time
}
```

### 5.3 `internal/modules/protocol/` — MQTT Wire Protocol (Generic)

```
protocol/
├── codec.go                        # Remaining length encode/decode, UTF-8 strings
├── codec_test.go
├── decoder.go                      # DecodeConnect, DecodePublish, DecodeSubscribe
├── decoder_test.go
├── encoder.go                      # EncodeConnack, EncodePuback, EncodeSuback, EncodePingresp
├── encoder_test.go
├── errors.go                       # Protocol-level errors
├── packet.go                       # PacketType, FixedHeader, ConnectPacket, PublishPacket, etc.
├── reader.go                       # ReadPacket (from io.Reader)
└── reader_test.go
```

**Mudanças**: Nenhuma. Este módulo está perfeito. Coesão 10/10.

### 5.4 `internal/modules/auth/` — Authentication (Supporting)

```
auth/
├── domain/
│   ├── authenticator.go            # Authenticator interface (the port)
│   └── errors.go                   # ErrAuthFailed, ErrEmptyCredentials
├── env_authenticator.go            # EnvAuthenticator — validates against env vars
├── env_authenticator_test.go
└── interfaces.go                   # Logger (if needed for future providers)
```

**Rationale**: Auth é extraído como módulo independente porque:
1. **Hoje** é simples (env vars), mas o bounded context linguístico é claro: authenticate, credentials, provider
2. **Amanhã** pode crescer para: external credential stores (Kafka SASL, mTLS), refresh tokens para frontend, múltiplos providers, token-based auth para HTTP API
3. **Padrão de integração**: connection define a port (`Authenticator` interface) no seu `interfaces.go`, auth fornece a implementação. Injeção via constructor no bootstrap.

**`domain/authenticator.go`**:
```go
package domain

// Authenticator validates client credentials.
// Implementations must be safe for concurrent use.
type Authenticator interface {
    Authenticate(username, password string) bool
}
```

**`env_authenticator.go`**:
```go
package auth

import domain "microbroker-mqtt-edge/internal/modules/auth/domain"

// EnvAuthenticator validates credentials against configured values.
// Implements domain.Authenticator.
type EnvAuthenticator struct {
    username string
    password string
}

func NewEnvAuthenticator(username, password string) *EnvAuthenticator {
    return &EnvAuthenticator{username: username, password: password}
}

func (a *EnvAuthenticator) Authenticate(username, password string) bool {
    if username == "" || password == "" {
        return false
    }
    return username == a.username && password == a.password
}

// Compile-time check
var _ domain.Authenticator = (*EnvAuthenticator)(nil)
```

### 5.5 `internal/modules/connection/` — Connection Management (Core)

```
connection/
├── domain/
│   ├── client.go                   # Client entity (ID, Conn, KeepAlive, timestamps)
│   ├── client_test.go
│   ├── topic_registry.go           # TopicRegistry value object (allowed topics)
│   ├── topic_registry_test.go
│   └── errors.go                   # Domain errors (ErrMaxClientsReached, ErrInvalidTopicCount, etc.)
├── client_manager.go               # ClientManager (was ConnectionManager) — tracks active clients
├── client_manager_test.go
├── handler.go                      # handleConnection, readLoop, handlePublish, handleSubscribe
├── handler_test.go
├── interfaces.go                   # Authenticator port (imported from auth/domain) + Logger
├── server.go                       # TCP Server — ListenAndServe, accept loop
└── server_test.go
```

**Mudanças vs `session/`**:

| Antes (session) | Depois (connection) | Razão |
|---|---|---|
| `session/` | `connection/` | "Session" é ambíguo (MQTT session state? HTTP session?). "Connection" é o vocabulário ubíquo: o módulo gerencia conexões TCP de clientes MQTT |
| `session/auth.go` (embedded) | Removido — auth é módulo separado, injetado via `Authenticator` interface | Auth tem bounded context próprio e potencial de crescimento independente |
| `session.Message` | Removido — usa `common/domain.Message` | Elimina duplicação e a bridge no main.go |
| `ConnectionManager` | `ClientManager` | O que ele gerencia são clientes, não conexões genéricas. O nome deve refletir o conceito de domínio |
| `session.Logger` + `session.NopLogger` | Importa `common/observability.Logger` | Elimina duplicação. Se precisar de subset, define local |

**`interfaces.go` — connection depende da port de auth**:
```go
package connection

import authdomain "microbroker-mqtt-edge/internal/modules/auth/domain"

// Authenticator is the port used by the connection module to validate credentials.
// The implementation is provided by the auth module and injected via constructor.
type Authenticator = authdomain.Authenticator
```

**`server.go` — recebe Authenticator injetado**:
```go
func NewServer(
    address string,
    connMgr *ClientManager,
    auth Authenticator,           // injected from auth module
    topics *domain.TopicRegistry,
    msgChan chan<- commondomain.Message,
    timezone string,
    logger Logger,
) *Server { ... }
```

**Bootstrap wiring** (`cmd/broker/bootstrap/modules.go`):
```go
// Auth module
authenticator := auth.NewEnvAuthenticator(cfg.Username, cfg.Password)

// Connection module — auth injected
server := connection.NewServer(cfg.Address(), clientMgr, authenticator, topics, msgChan, cfg.Timezone, logger)
```

### 5.6 `internal/modules/ingestion/` — Persistence Pipeline (Core)

```
ingestion/
├── domain/
│   └── errors.go                   # ErrStoreFailure, ErrQueueFull
├── application/
│   ├── interfaces.go               # Store port, Logger (ou importa de common)
│   ├── pipeline.go                 # Pipeline — routes messages to per-topic queues
│   ├── pipeline_test.go
│   ├── queue.go                    # Queue — FIFO per-topic with sequential consumer
│   └── queue_test.go
└── adapters/
    └── outbound/
        └── database/
            ├── repository.go       # SQLiteRepository implements Store
            └── repository_test.go
```

**Mudanças vs atual**:

| Antes | Depois | Razão |
|---|---|---|
| `ingestion/domain/message.go` | Removido — usa `common/domain.Message` | Message é value object compartilhado, não pertence a um domínio específico |
| `ingestion/domain/errors.go` | `ingestion/domain/errors.go` | Mantém — erros são específicos do domínio de ingestion |
| `ingestion/application/Logger` + `NopLogger` | Importa de `common/observability` | Elimina duplicação |

**`application/interfaces.go` — Store port atualizado**:
```go
package application

import (
    "context"
    domain "microbroker-mqtt-edge/internal/common/domain"
)

type Store interface {
    SaveRawData(ctx context.Context, msg domain.Message) error
    GetByTopic(ctx context.Context, topic string) ([]domain.Message, error)
    Close() error
}
```

### 5.7 `internal/modules/processing/` — Fan-out Worker Engine (Core)

```
processing/
├── domain/
│   └── worker.go                   # Worker interface (Name, Process, Close)
├── fanout.go                       # FanOut engine (was Dispatcher) — broadcasts to all workers
├── fanout_test.go
├── interfaces.go                   # Logger
└── workers/
    ├── logger_worker.go            # Reference implementation — logs messages
    └── logger_worker_test.go
```

**Mudanças vs `dispatch/`**:

| Antes (dispatch) | Depois (processing) | Razão |
|---|---|---|
| `dispatch/` | `processing/` | "Dispatch" sugere roteamento para destino específico. "Processing" reflete o estágio do pipeline: mensagens persistidas são processadas por workers |
| `Dispatcher` | `FanOut` | O nome da struct deve refletir o padrão: fan-out (1 mensagem → N workers simultâneos) |
| `dispatch/domain.Message = ingestion.Message` | Usa `common/domain.Message` diretamente | Elimina type alias e acoplamento entre módulos |
| `dispatch/domain/worker.go` importa `ingestiondomain` | `processing/domain/worker.go` importa `common/domain` | Dependência limpa: domínio → common, nunca domínio → outro domínio |

**`domain/worker.go` — limpo**:
```go
package domain

import (
    "context"
    commondomain "microbroker-mqtt-edge/internal/common/domain"
)

// Worker defines the contract for any message processing plugin.
// Implementations must be safe for concurrent use.
type Worker interface {
    Name() string
    Process(ctx context.Context, msg commondomain.Message) error
    Close() error
}
```

**`fanout.go` — renomeado**:
```go
package processing

// FanOut reads messages from an input channel and broadcasts them
// to all registered workers concurrently. It waits for every worker
// to finish before processing the next message.
type FanOut struct {
    workers []domain.Worker
    input   <-chan commondomain.Message
    logger  Logger
}
```

### 5.8 `internal/modules/audit/` — Query API (Supporting)

```
audit/
├── domain/
│   └── record.go                   # Record struct (query result DTO)
├── handler.go                      # HTTP handler (GET /audit/{topic}, GET /audit-count/{topic})
├── handler_test.go
├── interfaces.go                   # Reader port, Logger
└── adapters/
    └── outbound/
        └── database/
            ├── sqlite_reader.go    # SQLiteReader implements Reader
            └── sqlite_reader_test.go
```

**Mudanças vs `common/audit/`**:

| Antes | Depois | Razão |
|---|---|---|
| `common/audit/` | `modules/audit/` | Audit é um módulo de negócio, não um utilitário compartilhado |
| `audit.Record` inline | `audit/domain/record.go` | Separação clara de domínio |
| `audit.SQLiteReader` no mesmo pacote | `audit/adapters/outbound/database/` | Segue o padrão hexagonal — adapter outbound |
| `audit.Handler` no mesmo pacote | `audit/handler.go` | Handler é o adapter inbound (HTTP) |


---

## 6. `cmd/broker/` — Bootstrap Reorganization

### 6.1 Current Problem

O `main.go` atual tem ~140 linhas e faz tudo: config, logger, context, signals, database, channels, bridge goroutine, workers, pipeline, dispatcher, session server, audit HTTP, e shutdown. É funcional mas não escala.

### 6.2 Proposed Structure

```
cmd/
└── broker/
    ├── main.go                     # Slim entrypoint — calls bootstrap
    ├── bootstrap/
    │   ├── database.go             # SQLite connection + migrations
    │   ├── modules.go              # Wire all modules (connection, ingestion, processing, audit)
    │   ├── server.go               # Start TCP + HTTP servers
    │   └── shutdown.go             # Graceful shutdown sequence
    └── config/
        ├── config.go               # Config struct, Load(), validate()
        └── config_test.go
```

**`main.go` — slim**:
```go
package main

func main() {
    cfg, err := config.Load()
    if err != nil {
        slog.Error("failed to load config", "error", err)
        os.Exit(1)
    }

    logger := observability.NewSlogLogger(slog.Default())

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    app, err := bootstrap.Init(ctx, cfg, logger)
    if err != nil {
        logger.Error("failed to initialize", "error", err)
        os.Exit(1)
    }

    app.Start(ctx)
    app.WaitForShutdown()
}
```

**`bootstrap/modules.go` — wiring sem bridge**:
```go
// Antes: 3 channels + bridge goroutine
sessionChan := make(chan session.Message, cfg.QueueBufferSize)
ingestChan := make(chan ingestiondomain.Message, cfg.QueueBufferSize)
dispatchChan := make(chan dispatchdomain.Message, cfg.QueueBufferSize)
// + goroutine convertendo session.Message → ingestion.Message

// Depois: 2 channels, zero bridge
msgChan := make(chan domain.Message, cfg.QueueBufferSize)      // connection → ingestion
processChan := make(chan domain.Message, cfg.QueueBufferSize)   // ingestion → processing
```

A bridge goroutine desaparece completamente porque todos os módulos usam o mesmo `common/domain.Message`.

---

## 7. `tests/` — Test Structure

```
tests/
├── e2e/
│   └── broker_e2e_test.go          # Full pipeline E2E (TCP → persist → worker)
└── k6/
    ├── Dockerfile.k6
    ├── README.md
    ├── docker-compose.k6.yml
    ├── run.sh
    └── scripts/
        ├── mqtt_publish.js
        └── stress/
            ├── burst_1k.js
            ├── config.js
            ├── payloads.js
            ├── run_burst.sh
            ├── run_stress.sh
            └── smt_line_stress.js
```

**Mudanças**: Nenhuma nos testes. A estrutura de testes está boa. Os E2E testam o pipeline completo e os K6 fazem stress test. Os imports nos E2E precisarão ser atualizados para refletir os novos paths dos módulos.

---

## 8. Dependency Flow — Before vs After

### Before (Current)

```
session/domain ◄── session (server, handler, auth, connection_manager)
                        │
                        ├── protocol (function calls) ✅
                        │
                        └── session.Message ──(manual bridge goroutine)──▶ ingestion/domain.Message
                                                                                │
                                                                                ▼
                                                                          ingestion/application
                                                                                │
                                                                                ▼
                                                                    dispatch/domain.Message = ingestion/domain.Message ❌
                                                                                │
                                                                                ▼
                                                                          dispatch (Dispatcher)
                                                                                │
                                                                                ▼
                                                                          dispatch/workers

common/audit ◄── (reads same DB, zero code relationship with ingestion) ⚠️
```

**Problems**:
- `dispatch/domain` imports `ingestion/domain` (cross-domain dependency)
- `session.Message` → `ingestion.Message` requires manual bridge
- `audit` in `common/` but is a business module

### After (Proposed)

```
common/domain.Message ◄── (imported by all modules as shared value object)
common/observability.Logger ◄── (imported by all modules)

auth/domain.Authenticator ◄── auth (EnvAuthenticator)
    │
    └── (injected into connection via interface) ✅

connection/domain ◄── connection (server, handler, client_manager)
    │                       │
    │                       ├── protocol (function calls) ✅
    │                       ├── auth (via Authenticator interface) ✅
    │                       │
    │                       └── chan domain.Message ──▶ ingestion/application
    │                                                        │
    │                                                        └── chan domain.Message ──▶ processing (FanOut)
    │                                                                                        │
    │                                                                                        ▼
    │                                                                                   processing/workers
    │
    └── (no cross-domain imports, auth decoupled via port) ✅

audit/domain ◄── audit (handler, reader)
    └── (reads same DB, explicit Reader port) ✅
```

**Improvements**:
- Zero cross-domain imports
- No bridge goroutine needed
- Each module depends only on `common/` and its own domain
- Audit is a proper module with hexagonal structure

---

## 9. Migration Checklist

### Phase 1: Foundation (no breaking changes)

- [ ] Create `internal/common/domain/message.go` with the canonical `Message` value object
- [ ] Create `internal/common/observability/logger.go` consolidating the Logger interface
- [ ] Move `internal/common/logger.go` content into `internal/common/observability/logger.go`

### Phase 2: Module Renames & Moves

- [ ] Create `internal/modules/auth/` with domain/authenticator.go, domain/errors.go, env_authenticator.go, env_authenticator_test.go
- [ ] Rename `internal/modules/session/` → `internal/modules/connection/`
- [ ] Remove `auth.go` and `auth_test.go` from connection (now in auth module)
- [ ] Rename `internal/modules/dispatch/` → `internal/modules/processing/`
- [ ] Move `internal/common/audit/` → `internal/modules/audit/`
- [ ] Rename `ConnectionManager` → `ClientManager`
- [ ] Rename `Dispatcher` → `FanOut`

### Phase 3: Unify Message Type

- [ ] Update `connection/server.go` to use `common/domain.Message` instead of local `Message`
- [ ] Update `connection/handler.go` to emit `common/domain.Message`
- [ ] Remove `session.Message` struct
- [ ] Remove `ingestion/domain/message.go` (use common)
- [ ] Remove `dispatch/domain.Message` type alias (use common)
- [ ] Remove bridge goroutine from `main.go`

### Phase 4: Unify Logger

- [ ] Update all modules to import `common/observability.Logger`
- [ ] Remove duplicate Logger interfaces from: `session/interfaces.go`, `ingestion/application/interfaces.go`, `dispatch/interfaces.go`, `workers/logger_worker.go`, `audit/interfaces.go`
- [ ] Keep `NopLogger` only in `common/observability`

### Phase 5: Bootstrap Restructure

- [ ] Move `cmd/main.go` → `cmd/broker/main.go`
- [ ] Create `cmd/broker/bootstrap/` with database.go, modules.go, server.go, shutdown.go
- [ ] Move `internal/config/` → `cmd/broker/config/`
- [ ] Slim down main.go to ~20 lines

### Phase 6: Audit Hexagonal Structure

- [ ] Create `modules/audit/domain/record.go`
- [ ] Create `modules/audit/adapters/outbound/database/sqlite_reader.go`
- [ ] Move handler to `modules/audit/handler.go`
- [ ] Move interfaces to `modules/audit/interfaces.go`

### Phase 7: Verification

- [ ] `go build ./...` — compila sem erros
- [ ] `golangci-lint run ./...` — sem warnings
- [ ] `go test ./...` — todos os testes passam (incluindo E2E)
- [ ] `go vet ./...` — sem issues

---

## 10. Summary of Changes

### What Changes

| Item | Before | After |
|---|---|---|
| Module name | `session` | `connection` |
| Module name | `dispatch` | `processing` |
| Module location | `common/audit` | `modules/audit` |
| Auth location | embedded in `session/auth.go` | `modules/auth/` (own module) |
| Struct name | `Dispatcher` | `FanOut` |
| Struct name | `ConnectionManager` | `ClientManager` |
| Message type | 3 duplicates + bridge goroutine | 1 in `common/domain` |
| Logger interface | 6+ duplicates | 1 in `common/observability` |
| Entrypoint | `cmd/main.go` (140 lines) | `cmd/broker/main.go` (~20 lines) + `bootstrap/` |
| Config location | `internal/config/` | `cmd/broker/config/` |

### What Does NOT Change

- **Zero logic changes** — all business logic, concurrency patterns, and algorithms remain identical
- **Auth logic** — `EnvAuthenticator` code is identical, just moved to its own module and injected via interface
- **Protocol module** — untouched, perfect as-is
- **Platform module** — untouched (SQLite, migrator, migrations)
- **Test logic** — only import paths change
- **K6 stress tests** — untouched
- **Docker/CI** — only entrypoint path changes (`cmd/broker/main.go`)
- **Database schema** — untouched
- **Worker interface** — same contract, just imports from common

### Naming Rationale

| Old Name | New Name | Why |
|---|---|---|
| `session` | `connection` | O módulo gerencia conexões TCP de clientes MQTT. "Session" é ambíguo — MQTT tem conceito de "session state" (clean session flag) que este broker nem implementa. "Connection" é o vocabulário real do código. |
| `dispatch` | `processing` | O módulo não "despacha" para um destino — ele faz broadcast/fan-out para TODOS os workers. "Processing" reflete o estágio do pipeline: dados persistidos são processados por plugins extensíveis. |
| `Dispatcher` | `FanOut` | Nomeia o padrão arquitetural real: fan-out (1:N broadcast). Qualquer dev que leia `FanOut.Start()` entende imediatamente o comportamento. |
| `ConnectionManager` | `ClientManager` | O recurso gerenciado são clientes MQTT (com ID, keep-alive, estado), não conexões TCP genéricas. O nome deve refletir o conceito de domínio. |
