# TESTING — Test Strategy & Gate Commands

> Derived from CLAUDE.md / AGENTS.md and the existing test suite (`audit/raw`, `ingestion`,
> `fanout`, `config`). Read by Tasks/Execute/Validate every session to co-locate tests and run gates.

## Test Coverage Matrix

| Code Layer | Required Test Type | How (tooling) |
| --- | --- | --- |
| `domain/` (pure value objects, calculators) | **unit** | testify `assert`/`require`, table-driven. No mocks (pure). |
| `application/` (use cases + ports) | **unit** | gomock (`go.uber.org/mock`) for outbound ports; table-driven. |
| `adapters/outbound/database/` | **integration** | Real `modernc.org/sqlite` against an **isolated temp/`:memory:` DB per test** (see `ingestion/repository_test.go`). |
| `adapters/inbound/http_handler/` | **unit** | `net/http/httptest` + mock use case. |
| `adapters/inbound/worker/` | **unit** | fake/mocked ports; assert filtering + delegation. |
| `adapters/outbound/sink/` | **unit** | in-process fakes; for websocket use `httptest.Server` + gorilla dialer. |
| migration `*.sql` | **integration** | apply via `database.Migrator` to a temp DB; assert tables/idempotency. |
| `cmd/broker/config/` | **unit** | env-var driven, table-driven (`config_test.go`). |
| `cmd/broker/bootstrap/` (wiring) | **e2e** | full-stack test in `tests/` — boot, publish, assert persistence + REST + ws. |

**Rules:** tests are co-located in the same task that creates the layer (never a separate "write tests"
task). `Tests: none` is only valid for pure docs/config-file changes.

## Gate Check Commands

| Gate | Command | When |
| --- | --- | --- |
| **build** | `go build ./...` | docs/config-only tasks |
| **quick** | `go test ./<changed-package>/...` | per unit/integration task |
| **full** | `go build ./... && go vet ./... && golangci-lint run ./... && go test -race ./...` | integration/e2e tasks, and before any commit (== `/verify-go`) |

> Mocks are generated with `mockgen` into each package's `mocks/` dir — regenerate after changing a port.

## Parallelism Assessment

| Test Type | Parallel-Safe | Why |
| --- | --- | --- |
| unit | **Yes** | Isolated per package; no shared state. |
| integration | **Yes** | Each test owns a temp/`:memory:` SQLite DB — no shared file. |
| e2e | **No** | Binds TCP/HTTP ports + a real DB file; run sequentially. |

## Conventions

- Pure-Go SQLite only (`modernc.org/sqlite`, `CGO_ENABLED=0`). Never `mattn/go-sqlite3`.
- Add/extend tests for changed code even if unasked (AGENTS.md). Prefer table-driven.
- The `test-expert` skill drives test authoring (two-phase: list cases → implement).
