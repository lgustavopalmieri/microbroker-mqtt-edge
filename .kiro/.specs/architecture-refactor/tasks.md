# Architecture Refactor Tasks

**Design**: `.specs/features/architecture-refactor/design.md`
**Status**: Done

---

## Execution Plan

### Phase 1: Foundation (Sequential) ✅ DONE

```
T1 ✅ → T2 ✅
```

### Phase 2: Module Extraction & Renames ✅ DONE

```
       ┌→ T3 ✅ ─┐
T2 ────┼→ T4 ✅ ─┼────→ T8
       ├→ T5 ✅ ─┤
       ├→ T6 ✅ ─┤
       └→ T7 ✅ ─┘
```

### Phase 3: Integration ✅ DONE

```
T8 ✅ → T9 ✅ → T10 ✅
```

### Phase 4: Cleanup (added post-execution)

```
T11 (pending) — Generate gomock mocks for interfaces, replace manual test doubles
```

---

## Task Breakdown

### T1: Create `common/domain/message.go` ✅ COMPLETE

**What**: Create the canonical `Message` struct
**Where**: `internal/common/domain/message.go`
**Requirement**: REFAC-01

**Done when**:

- [x] `internal/common/domain/message.go` exists with `Message` struct
- [x] Package is `domain`
- [x] `go build ./internal/common/domain/` compiles

---

### T2: Create `common/observability/logger.go` ✅ COMPLETE

**What**: Consolidate Logger interface, NopLogger, NewSlogLogger
**Where**: `internal/common/observability/logger.go`
**Requirement**: REFAC-05

**Done when**:

- [x] `internal/common/observability/logger.go` exists with `Logger`, `NopLogger`, `NewSlogLogger`
- [x] Old `internal/common/logger.go` deleted
- [x] All modules use `type Logger = observability.Logger` alias (except workers ISP subset)
- [x] Only 2 Logger definitions remain: `observability.Logger` (canonical) + `workers.Logger` (ISP subset)

---

### T3: Create `modules/auth/` ✅ COMPLETE

**What**: Extract auth as independent module
**Where**: `internal/modules/auth/`
**Requirement**: REFAC-02

**Done when**:

- [x] `auth/domain/authenticator.go` — `Authenticator` interface
- [x] `auth/domain/errors.go` — `ErrAuthFailed`, `ErrEmptyCredentials`
- [x] `auth/env_authenticator.go` — `EnvAuthenticator` with compile-time check
- [x] `auth/env_authenticator_test.go` — 7 table-driven test cases (same as original)
- [x] `go test ./internal/modules/auth/...` passes

---

### T4: Rename `session/` → `connection/` ✅ COMPLETE

**What**: Rename module, rename `ConnectionManager` → `ClientManager`, inject auth, use shared Message/Logger
**Where**: `internal/modules/connection/`
**Requirement**: REFAC-03, REFAC-01, REFAC-05

**Done when**:

- [x] `internal/modules/connection/` exists with all session files (except auth)
- [x] `internal/modules/session/` deleted
- [x] `ConnectionManager` → `ClientManager`
- [x] Uses `common/domain.Message` (no local Message struct)
- [x] Uses `auth/domain.Authenticator` via injection
- [x] Uses `observability.Logger` via type alias
- [x] `go test ./internal/modules/connection/...` passes

---

### T5: Rename `dispatch/` → `processing/` ✅ COMPLETE

**What**: Rename module, `Dispatcher` → `FanOut`, remove Message type alias
**Where**: `internal/modules/processing/`
**Requirement**: REFAC-04, REFAC-01, REFAC-05

**Done when**:

- [x] `internal/modules/processing/` exists
- [x] `internal/modules/dispatch/` deleted
- [x] `Dispatcher` → `FanOut`, `NewDispatcher` → `NewFanOut`
- [x] `domain/worker.go` imports `common/domain.Message` directly (no type alias)
- [x] `workers/logger_worker.go` keeps local `Logger` (ISP subset with only `Info`)
- [x] `go test ./internal/modules/processing/...` passes

---

### T6: Simplify `ingestion/` ✅ COMPLETE

**What**: Remove duplicate Message, update imports to common/domain and observability
**Where**: `internal/modules/ingestion/`
**Requirement**: REFAC-01, REFAC-05

**Done when**:

- [x] `ingestion/domain/message.go` deleted
- [x] `ingestion/domain/errors.go` remains
- [x] All application + adapter files use `common/domain.Message`
- [x] `application/interfaces.go` uses `type Logger = observability.Logger`
- [x] `go test ./internal/modules/ingestion/...` passes

---

### T7: Move `common/audit/` → `modules/audit/` ✅ COMPLETE

**What**: Promote audit to proper business module following PROJECT-EXAMPLE.md pattern — domain + features/query with application (usecase, interface, dto, mocks), adapters inbound (http_handler with DI) and outbound (database repository with constructor)
**Where**: `internal/modules/audit/`
**Requirement**: REFAC-06

**Done when**:

- [x] `audit/domain/record.go` — Record entity
- [x] `audit/domain/errors.go` — ErrQueryFailed
- [x] `audit/features/query/application/usecase.go` + `new_usecase.go` — UseCase (GetByTopic, CountByTopic)
- [x] `audit/features/query/application/interface.go` — Repository port + Logger alias
- [x] `audit/features/query/application/dto.go` — GetByTopicOutput, CountByTopicOutput
- [x] `audit/features/query/application/mocks/` — gomock generated MockRepository
- [x] `audit/features/query/adapters/inbound/http_handler/handler.go` — HTTP endpoints
- [x] `audit/features/query/adapters/inbound/http_handler/di.go` — constructor with UseCase injection
- [x] `audit/features/query/adapters/inbound/http_handler/handler_test.go` — gomock table-driven tests
- [x] `audit/features/query/adapters/outbound/database/repository.go` + `new.go` — SQLite implementation
- [x] `internal/common/audit/` deleted
- [x] Bootstrap wires: repo → usecase → handler
- [x] `go test ./internal/modules/audit/...` passes

---

### T8: Restructure Bootstrap ✅ COMPLETE

**What**: New entrypoint at `cmd/broker/main.go` (~50 lines), bootstrap package with database, modules, server, shutdown
**Where**: `cmd/broker/`
**Requirement**: REFAC-07

**Done when**:

- [x] `cmd/broker/main.go` — slim entrypoint delegating to bootstrap
- [x] `cmd/broker/bootstrap/database.go` — InitDatabase (dir, connection, migrations)
- [x] `cmd/broker/bootstrap/modules.go` — InitModules (wires auth, connection, ingestion, processing)
- [x] `cmd/broker/bootstrap/server.go` — StartServers (TCP + HTTP, launches goroutines)
- [x] `cmd/broker/bootstrap/shutdown.go` — GracefulShutdown (ordered cleanup)
- [x] `cmd/broker/config/config.go` + `config_test.go` (moved from internal/config/)
- [x] `cmd/main.go` deleted, `internal/config/` deleted
- [x] NO bridge goroutine — 2 channels: `msgChan` + `processChan`
- [x] `go build ./cmd/broker/` compiles

---

### T9: Update Dockerfile ✅ COMPLETE

**What**: Update build path from `cmd/main.go` to `cmd/broker`
**Where**: `Dockerfile`
**Requirement**: REFAC-07

**Done when**:

- [x] Dockerfile references `./cmd/broker` as build target

---

### T10: Final Verification ✅ COMPLETE

**What**: Full build + vet + test suite
**Requirement**: All (REFAC-01 through REFAC-07)

**Results**:

- [x] `go build ./...` — passes
- [x] `go vet ./...` — passes
- [x] `go test ./...` — ALL 14 test packages pass (including E2E)
- [x] Zero references to old paths (`session`, `dispatch`, `common/audit`, `internal/config`)
- [x] Only 2 Logger definitions (observability canonical + workers ISP subset)
- [x] No bridge goroutine in main.go

---

### T11: Generate gomock mocks for interfaces — ✅ COMPLETE

**What**: Install `go.uber.org/mock/gomock`, generate mocks for all interfaces using `mockgen`, refactor audit test to use gomock
**Where**: `mocks/` subdirectories per package
**Requirement**: test-expert skill compliance

**Done when**:

- [x] `go.uber.org/mock` added to go.mod
- [x] `mockgen` generated mocks for: `Store`, `Reader`, `Worker`, `Authenticator`
- [x] `audit/handler_test.go` refactored to use `gomock` + `mocks.MockReader`
- [x] `go test ./...` passes

**Generated mocks**:

- `ingestion/application/mocks/mock_store.go`
- `audit/mocks/mock_reader.go`
- `processing/domain/mocks/mock_worker.go`
- `auth/domain/mocks/mock_authenticator.go`

**Note**: Queue/Pipeline/FanOut tests keep manual spy/callback test doubles — these test async goroutine flows where spy patterns (`.getSaved()`, `.processFn` callback) are more expressive than gomock for verifying concurrent behavior. E2E `collectWorker` also kept as-is (integration test double, not a unit mock).

---

## Deviations from Original Plan

| Planned | Actual | Reason |
|---|---|---|
| Mocks with gomock from start | Manual test doubles migrated, gomock added in T11 | Discovered test-expert skill requirement after initial implementation |
