# Architecture Refactor Design

**Spec**: `.specs/features/architecture-refactor/spec.md`
**Architecture Review**: `docs/architecture-review.md`
**Status**: Draft

---

## Architecture Overview

Refatoração puramente estrutural — mover, renomear, e unificar. Zero mudança de lógica.

```
ANTES:                                    DEPOIS:
cmd/main.go (140 lines)                   cmd/broker/main.go (~20 lines)
                                          cmd/broker/bootstrap/ (wiring)
                                          cmd/broker/config/

internal/common/logger.go                 internal/common/observability/logger.go
internal/common/audit/                    internal/modules/audit/ (promoted)
                                          internal/common/domain/message.go (new, shared)

internal/modules/session/                 internal/modules/auth/ (extracted)
  ├── auth.go (embedded)                  internal/modules/connection/ (renamed)
  ├── server.go                             ├── server.go
  ├── handler.go                            ├── handler.go
  ├── connection_manager.go                 ├── client_manager.go (renamed)
  └── domain/                               └── domain/

internal/modules/dispatch/                internal/modules/processing/ (renamed)
  ├── dispatcher.go                         ├── fanout.go (renamed)
  └── domain/worker.go (type alias)         └── domain/worker.go (imports common)

internal/modules/ingestion/               internal/modules/ingestion/ (simplified)
  └── domain/message.go (duplicate)         └── domain/errors.go (message removed)

internal/modules/protocol/                internal/modules/protocol/ (untouched)
```

---

## Code Reuse Analysis

### Existing Components to Leverage

| Component | Location | How to Use |
|---|---|---|
| `EnvAuthenticator` | `session/auth.go` | Move as-is to `auth/env_authenticator.go` |
| `Authenticator` interface | `session/auth.go` | Move to `auth/domain/authenticator.go` |
| `ConnectionManager` | `session/connection_manager.go` | Rename to `ClientManager`, move to `connection/` |
| `Server` | `session/server.go` | Move to `connection/`, update imports |
| `handler.go` | `session/handler.go` | Move to `connection/`, update imports |
| `Dispatcher` | `dispatch/dispatcher.go` | Rename to `FanOut`, move to `processing/fanout.go` |
| `Worker` interface | `dispatch/domain/worker.go` | Move to `processing/domain/`, remove type alias |
| `LoggerWorker` | `dispatch/workers/` | Move to `processing/workers/` |
| `Pipeline` + `Queue` | `ingestion/application/` | Keep, update imports |
| `audit.Handler` + `Reader` | `common/audit/` | Move to `modules/audit/` with hexagonal structure |
| `Logger` + `NopLogger` + `NewSlogLogger` | `common/logger.go` | Move to `common/observability/logger.go` |

### Integration Points

| System | Integration Method |
|---|---|
| Auth → Connection | `Authenticator` interface injected via constructor |
| Connection → Ingestion | `chan common/domain.Message` (direct channel) |
| Ingestion → Processing | `chan common/domain.Message` (direct channel) |
| Ingestion → Audit | Shared SQLite database (audit reads what ingestion writes) |

---

## Components

### `common/domain` — Shared Value Objects

- **Purpose**: Single source of truth for cross-module value objects
- **Location**: `internal/common/domain/`
- **Interfaces**: `Message` struct (ClientID, Topic, Payload, Timezone, Timestamp)
- **Dependencies**: `time` (stdlib only)
- **Reuses**: Fields from existing `session.Message`, `ingestion/domain.Message`

### `common/observability` — Shared Observability Contracts

- **Purpose**: Canonical Logger interface and implementations
- **Location**: `internal/common/observability/`
- **Interfaces**: `Logger` (Info, Error, Warn, Debug), `NopLogger`, `NewSlogLogger`
- **Dependencies**: `log/slog` (stdlib)
- **Reuses**: Existing `common.Logger`, `common.NopLogger`, `common.NewSlogLogger`

### `modules/auth` — Authentication Module

- **Purpose**: Credential validation, extensible for future providers
- **Location**: `internal/modules/auth/`
- **Interfaces**: `domain.Authenticator` (Authenticate(username, password string) bool)
- **Dependencies**: None (pure domain)
- **Reuses**: Existing `session.EnvAuthenticator` code verbatim

### `modules/connection` — Connection Management (renamed from session)

- **Purpose**: TCP server, client lifecycle, MQTT message handling
- **Location**: `internal/modules/connection/`
- **Interfaces**: `Server`, `ClientManager`, `handler` (internal)
- **Dependencies**: `protocol`, `auth/domain.Authenticator`, `common/domain.Message`, `common/observability.Logger`
- **Reuses**: All existing `session/` code, renamed

### `modules/processing` — Fan-out Worker Engine (renamed from dispatch)

- **Purpose**: Broadcast messages to all registered workers concurrently
- **Location**: `internal/modules/processing/`
- **Interfaces**: `FanOut`, `domain.Worker`
- **Dependencies**: `common/domain.Message`, `common/observability.Logger`
- **Reuses**: All existing `dispatch/` code, renamed

### `modules/audit` — Query API (moved from common)

- **Purpose**: HTTP REST API for querying persisted messages
- **Location**: `internal/modules/audit/`
- **Interfaces**: `Reader` port, `Handler` (HTTP)
- **Dependencies**: `common/observability.Logger`, `database/sql`
- **Reuses**: All existing `common/audit/` code, restructured

### `cmd/broker/bootstrap` — Application Wiring

- **Purpose**: Initialize and wire all modules, zero business logic
- **Location**: `cmd/broker/bootstrap/`
- **Interfaces**: `Init(ctx, cfg, logger) (*App, error)`, `App.Start()`, `App.WaitForShutdown()`
- **Dependencies**: All modules, config, platform
- **Reuses**: Existing `main.go` wiring logic, decomposed

---

## Tech Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Auth interface location | `auth/domain/authenticator.go` | Auth owns its contract; connection imports it. Customer/Supplier pattern. |
| Message location | `common/domain/message.go` | Shared value object used by 3+ modules. Not owned by any single domain. |
| Logger subset in workers | Keep local `Logger` with only `Info` | Legitimate subset — workers package only needs Info. ISP principle. |
| Config location | `cmd/broker/config/` | Config is server-specific, not a shared internal package. Follows PROJECT-EXAMPLE.md pattern. |
| Audit hexagonal depth | Shallow — `domain/` + `handler.go` + `adapters/outbound/` | Audit is simple enough that full hexagonal would be overengineering. |
