# Architecture Refactor Tasks

**Design**: `.specs/features/architecture-refactor/design.md`
**Status**: Draft

---

## Execution Plan

### Phase 1: Foundation (Sequential)

Create shared contracts that all modules will depend on.

```
T1 → T2
```

### Phase 2: Module Extraction & Renames (Parallel OK)

With shared contracts in place, extract/rename modules independently.

```
       ┌→ T3 [P] ─┐
T2 ────┼→ T4 [P] ─┼────→ T8
       ├→ T5 [P] ─┤
       ├→ T6 [P] ─┤
       └→ T7 [P] ─┘
```

### Phase 3: Integration (Sequential)

Wire everything together in bootstrap and update entrypoint.

```
T8 → T9 → T10
```

---

## Task Breakdown

### T1: Create `common/domain/message.go` — Shared Message Value Object

**What**: Create the canonical `Message` struct in `internal/common/domain/message.go`
**Where**: `internal/common/domain/message.go`
**Depends on**: None
**Reuses**: Fields from `internal/modules/ingestion/domain/message.go`
**Requirement**: REFAC-01

**Done when**:

- [ ] `internal/common/domain/message.go` exists with `Message` struct (ClientID, Topic, Payload, Timezone, Timestamp)
- [ ] Package is `domain`
- [ ] `go build ./internal/common/domain/` compiles

**Tests**: none (pure value object, no logic)
**Gate**: build

---

### T2: Create `common/observability/logger.go` — Consolidated Logger

**What**: Move Logger interface, NopLogger, and NewSlogLogger to `internal/common/observability/`
**Where**: `internal/common/observability/logger.go`
**Depends on**: None
**Reuses**: `internal/common/logger.go` (move content)
**Requirement**: REFAC-05

**Done when**:

- [ ] `internal/common/observability/logger.go` exists with `Logger` interface, `NopLogger`, `NewSlogLogger`
- [ ] Package is `observability`
- [ ] Old `internal/common/logger.go` is deleted
- [ ] `go build ./internal/common/observability/` compiles

**Tests**: none (interface + trivial adapter, same as current)
**Gate**: build

---

### T3: Create `modules/auth/` — Extract Auth Module [P]

**What**: Create auth module with `domain/authenticator.go`, `domain/errors.go`, `env_authenticator.go`, `env_authenticator_test.go`
**Where**: `internal/modules/auth/`
**Depends on**: T2 (uses observability.Logger if needed)
**Reuses**: `internal/modules/session/auth.go`, `internal/modules/session/auth_test.go`
**Requirement**: REFAC-02

**Done when**:

- [ ] `internal/modules/auth/domain/authenticator.go` exists with `Authenticator` interface
- [ ] `internal/modules/auth/domain/errors.go` exists with `ErrAuthFailed`, `ErrEmptyCredentials`
- [ ] `internal/modules/auth/env_authenticator.go` exists with `EnvAuthenticator` implementing `Authenticator`
- [ ] `internal/modules/auth/env_authenticator_test.go` exists with same test scenarios as current `auth_test.go`
- [ ] Compile-time interface check: `var _ domain.Authenticator = (*EnvAuthenticator)(nil)`
- [ ] Gate check passes: `go test ./internal/modules/auth/...`

**Tests**: unit (table-driven, same scenarios as existing auth_test.go)
**Gate**: quick — `go test ./internal/modules/auth/...`

---

### T4: Rename `session/` → `connection/` + Inject Auth [P]

**What**: Rename session module to connection, rename `ConnectionManager` → `ClientManager`, remove embedded auth (now in auth module), update all internal imports, use `common/domain.Message` and `common/observability.Logger`
**Where**: `internal/modules/connection/`
**Depends on**: T1 (common/domain.Message), T2 (observability.Logger), T3 (auth module)
**Reuses**: All `internal/modules/session/` files
**Requirement**: REFAC-03, REFAC-01, REFAC-05

**Done when**:

- [ ] `internal/modules/connection/` exists with all files from session (except auth.go, auth_test.go)
- [ ] `internal/modules/session/` is deleted
- [ ] `ConnectionManager` renamed to `ClientManager` in `client_manager.go`
- [ ] `server.go` uses `common/domain.Message` for msgChan (no local `Message` struct)
- [ ] `handler.go` emits `common/domain.Message` instead of `session.Message`
- [ ] `server.go` accepts `auth/domain.Authenticator` interface (injected)
- [ ] All files use `common/observability.Logger` (local Logger/NopLogger removed from interfaces.go)
- [ ] `connection/domain/errors.go` removes auth-related errors (ErrAuthFailed moved to auth module)
- [ ] Gate check passes: `go test ./internal/modules/connection/...`
- [ ] Test count: same number of tests as current session package

**Tests**: unit (existing tests migrated, imports updated)
**Gate**: quick — `go test ./internal/modules/connection/...`

---

### T5: Rename `dispatch/` → `processing/` + Rename Dispatcher → FanOut [P]

**What**: Rename dispatch module to processing, rename `Dispatcher` → `FanOut`, remove Message type alias, use `common/domain.Message` directly
**Where**: `internal/modules/processing/`
**Depends on**: T1 (common/domain.Message), T2 (observability.Logger)
**Reuses**: All `internal/modules/dispatch/` files
**Requirement**: REFAC-04, REFAC-01, REFAC-05

**Done when**:

- [ ] `internal/modules/processing/` exists with all files from dispatch
- [ ] `internal/modules/dispatch/` is deleted
- [ ] `dispatcher.go` renamed to `fanout.go`, struct `Dispatcher` → `FanOut`, `NewDispatcher` → `NewFanOut`
- [ ] `domain/worker.go` imports `common/domain.Message` directly (no type alias, no ingestion import)
- [ ] `workers/logger_worker.go` uses `common/domain.Message`
- [ ] All files use `common/observability.Logger` where applicable
- [ ] `workers/logger_worker.go` keeps local `Logger` interface (subset with only `Info`) — ISP
- [ ] Gate check passes: `go test ./internal/modules/processing/...`
- [ ] Test count: same number of tests as current dispatch package

**Tests**: unit (existing tests migrated, imports updated)
**Gate**: quick — `go test ./internal/modules/processing/...`

---

### T6: Simplify `ingestion/` — Remove Duplicate Message [P]

**What**: Remove `ingestion/domain/message.go`, update all ingestion code to use `common/domain.Message` and `common/observability.Logger`
**Where**: `internal/modules/ingestion/`
**Depends on**: T1 (common/domain.Message), T2 (observability.Logger)
**Reuses**: Existing ingestion code, only import changes
**Requirement**: REFAC-01, REFAC-05

**Done when**:

- [ ] `internal/modules/ingestion/domain/message.go` is deleted
- [ ] `internal/modules/ingestion/domain/errors.go` remains (domain-specific errors)
- [ ] `application/pipeline.go` imports `common/domain.Message`
- [ ] `application/queue.go` imports `common/domain.Message`
- [ ] `application/interfaces.go` uses `common/domain.Message` in Store interface and removes local Logger/NopLogger
- [ ] `adapters/outbound/database/repository.go` imports `common/domain.Message`
- [ ] Gate check passes: `go test ./internal/modules/ingestion/...`
- [ ] Test count: same number of tests as current ingestion package

**Tests**: unit (existing tests, imports updated)
**Gate**: quick — `go test ./internal/modules/ingestion/...`

---

### T7: Move `common/audit/` → `modules/audit/` [P]

**What**: Promote audit from common utility to proper business module with hexagonal structure
**Where**: `internal/modules/audit/`
**Depends on**: T2 (observability.Logger)
**Reuses**: All `internal/common/audit/` files
**Requirement**: REFAC-06

**Done when**:

- [ ] `internal/modules/audit/handler.go` exists (from common/audit/handler.go)
- [ ] `internal/modules/audit/interfaces.go` exists (Reader port, uses observability.Logger)
- [ ] `internal/modules/audit/domain/record.go` exists (Record struct)
- [ ] `internal/modules/audit/adapters/outbound/database/sqlite_reader.go` exists (from common/audit/sqlite_reader.go)
- [ ] `internal/common/audit/` is deleted
- [ ] Gate check passes: `go test ./internal/modules/audit/...`
- [ ] Test count: same number of tests as current audit package

**Tests**: unit (existing tests migrated)
**Gate**: quick — `go test ./internal/modules/audit/...`

---

### T8: Restructure Bootstrap — `cmd/broker/`

**What**: Move entrypoint to `cmd/broker/main.go`, create bootstrap package, move config, eliminate bridge goroutine
**Where**: `cmd/broker/`
**Depends on**: T3, T4, T5, T6, T7 (all modules in final locations)
**Reuses**: `cmd/main.go` wiring logic, `internal/config/`
**Requirement**: REFAC-07

**Done when**:

- [ ] `cmd/broker/main.go` exists (~20 lines, delegates to bootstrap)
- [ ] `cmd/broker/bootstrap/database.go` exists (SQLite connection + migrations)
- [ ] `cmd/broker/bootstrap/modules.go` exists (wire auth, connection, ingestion, processing, audit)
- [ ] `cmd/broker/bootstrap/server.go` exists (start TCP + HTTP servers)
- [ ] `cmd/broker/bootstrap/shutdown.go` exists (graceful shutdown)
- [ ] `cmd/broker/config/config.go` exists (moved from internal/config/)
- [ ] `cmd/broker/config/config_test.go` exists (moved)
- [ ] `cmd/main.go` is deleted
- [ ] `internal/config/` is deleted
- [ ] NO bridge goroutine — connection writes directly to ingestion channel
- [ ] Only 2 channels: `msgChan` (connection→ingestion) and `processChan` (ingestion→processing)
- [ ] Gate check passes: `go build ./cmd/broker/`

**Tests**: none (wiring only, verified by E2E)
**Gate**: build — `go build ./cmd/broker/`

---

### T9: Update Dockerfile & docker-compose

**What**: Update entrypoint path in Dockerfile from `cmd/main.go` to `cmd/broker/main.go`
**Where**: `Dockerfile`, `docker-compose.yml`
**Depends on**: T8
**Reuses**: Existing Dockerfile
**Requirement**: REFAC-07

**Done when**:

- [ ] Dockerfile references `cmd/broker/` as entrypoint
- [ ] docker-compose.yml is verified (update if needed)
- [ ] `docker build .` succeeds (if Docker available)

**Tests**: none
**Gate**: build

---

### T10: Final Verification — Full Test Suite

**What**: Run complete verification: build, lint, vet, all tests including E2E
**Where**: Project root
**Depends on**: T8, T9
**Reuses**: Existing test suite
**Requirement**: All (REFAC-01 through REFAC-07)

**Done when**:

- [ ] `go build ./...` passes
- [ ] `go vet ./...` passes
- [ ] `golangci-lint run ./...` passes (if available)
- [ ] `go test ./...` passes (ALL tests including E2E)
- [ ] No module imports another module's domain (only `common/`)
- [ ] No duplicate `Message` definitions
- [ ] No duplicate `Logger` interfaces (except legitimate subsets)
- [ ] No bridge goroutine in bootstrap

**Tests**: full suite
**Gate**: full — `go build ./... && go vet ./... && go test ./...`

---

## Parallel Execution Map

```
Phase 1 (Sequential — Foundation):
  T1 ──→ T2

Phase 2 (Parallel — Module Extraction & Renames):
  T2 complete, then:
    ├── T3 [P]  (auth extraction)
    ├── T4 [P]  (session → connection)  *depends on T3 for auth import
    ├── T5 [P]  (dispatch → processing)
    ├── T6 [P]  (ingestion simplification)
    └── T7 [P]  (audit promotion)

Phase 3 (Sequential — Integration):
  All Phase 2 complete, then:
    T8 ──→ T9 ──→ T10
```

**Note on T4**: T4 depends on T3 (needs auth module to exist for import). In practice, T4 can start in parallel but must wait for T3's `auth/domain/authenticator.go` to exist before compiling. If running sequentially, do T3 before T4.

---

## Task Granularity Check

| Task | Scope | Status |
|---|---|---|
| T1: Create common/domain/message.go | 1 file | ✅ Granular |
| T2: Create common/observability/logger.go | 1 file + delete 1 | ✅ Granular |
| T3: Create modules/auth/ | 4 files (domain + impl + test) | ✅ Granular (cohesive module) |
| T4: Rename session → connection | Move + rename + update imports | ⚠️ Multiple files but single cohesive operation |
| T5: Rename dispatch → processing | Move + rename + update imports | ⚠️ Multiple files but single cohesive operation |
| T6: Simplify ingestion | Delete 1 file + update imports | ✅ Granular |
| T7: Move audit to modules | Move + restructure | ⚠️ Multiple files but single cohesive operation |
| T8: Restructure bootstrap | Create 5 files + delete 2 | ⚠️ Cohesive wiring operation |
| T9: Update Dockerfile | 1-2 files | ✅ Granular |
| T10: Final verification | 0 files (verification only) | ✅ Granular |

**Note**: T4, T5, T7, T8 touch multiple files but each is a single cohesive operation (move a module). Splitting further would create artificial dependencies and make the refactor harder to reason about.

---

## Diagram-Definition Cross-Check

| Task | Depends On (body) | Diagram Shows | Status |
|---|---|---|---|
| T1 | None | No incoming arrows | ✅ Match |
| T2 | None | T1 → T2 | ⚠️ Diagram shows T1→T2 but body says None. T2 doesn't actually need T1. Sequential for simplicity. |
| T3 | T2 | T2 → T3 | ✅ Match |
| T4 | T1, T2, T3 | T2 → T4, T3 → T4 (implicit via Phase 2 start) | ✅ Match |
| T5 | T1, T2 | T2 → T5 | ✅ Match |
| T6 | T1, T2 | T2 → T6 | ✅ Match |
| T7 | T2 | T2 → T7 | ✅ Match |
| T8 | T3, T4, T5, T6, T7 | All Phase 2 → T8 | ✅ Match |
| T9 | T8 | T8 → T9 | ✅ Match |
| T10 | T8, T9 | T9 → T10 | ✅ Match |

---

## Test Co-location Validation

| Task | Code Layer | Requires | Task Says | Status |
|---|---|---|---|---|
| T1 | Value object (no logic) | none | none | ✅ OK |
| T2 | Interface + adapter (trivial) | none | none | ✅ OK |
| T3 | Domain + implementation | unit | unit | ✅ OK |
| T4 | Module rename (existing tests) | unit (migrate) | unit | ✅ OK |
| T5 | Module rename (existing tests) | unit (migrate) | unit | ✅ OK |
| T6 | Import updates (existing tests) | unit (migrate) | unit | ✅ OK |
| T7 | Module move (existing tests) | unit (migrate) | unit | ✅ OK |
| T8 | Bootstrap wiring | E2E (verified by T10) | none | ✅ OK (E2E in T10) |
| T9 | Dockerfile | none | none | ✅ OK |
| T10 | Verification only | full suite | full suite | ✅ OK |
