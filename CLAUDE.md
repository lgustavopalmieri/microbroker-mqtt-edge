# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

`microbroker-mqtt-edge` is a Go 1.25, single-module, embeddable MQTT 3.1.1 broker for edge/Industry-4.0 use. It ingests PUBLISH packets over TCP, persists them to local SQLite, and fans them out to workers. Zero external runtime dependencies.

## Commands

See @AGENTS.md for the full command list. Most-used:

- Build everything: `go build ./...` (broker binary: `go build -o microbroker ./cmd/broker`)
- Test: `go test ./...` — single test: `go test -run "^TestName$" ./path/to/pkg` — race: `go test -race ./...`
- Lint: `golangci-lint run ./...` — and `go vet ./...` after moving files or changing imports

Run `golangci-lint run ./...` and `go test ./...` (clean) before committing. Or use `/verify-go`.

## Architecture (Clean / Hexagonal — dependencies always point inward)

```
cmd/broker/bootstrap/   → wiring only, zero business logic
cmd/broker/config/      → config struct, load, validate
internal/common/        → shared interfaces (observability), value objects, test helpers
internal/modules/<m>/   → business domain modules, one per bounded context
internal/platform/      → infrastructure adapters (database, etc.)
```

Module/feature layout: `internal/modules/<module>/[<sub>/]features/<feature>/` with
`domain/` → `application/` (use case + `interface.go` ports + DTOs + `mocks/`) → `adapters/inbound/` (http_handler) + `adapters/outbound/` (database).

Rules — Claude must not violate these:
- **Dependency flow is always inward**: domain → application → adapters. Never the reverse.
- **Domain and application layers import zero infrastructure** (no `database/sql`, no concrete `slog`/otel, no http).
- Application defines ports in `interface.go`; adapters implement them.
- Handlers (HTTP) delegate to use cases only — no business logic in handlers.
- Constructor and logic go in separate files: `new_usecase.go` + `usecase.go`.
- Observability flows through interfaces in `internal/common`, never concrete loggers in domain/application.
- Infrastructure (DB, broker, server) is swappable — to change it, only adapters change.

When scaffolding a new module or feature, use `/hexagonal-scaffold` to match this layout exactly.

## Gotchas

- **SQLite is pure-Go** (`modernc.org/sqlite`, built with `CGO_ENABLED=0`). Never introduce `mattn/go-sqlite3` or any cgo dependency — it breaks the static build (`Dockerfile`).
- **Migrations**: currently a *custom embedded migrator* — add a numbered file (`002_*.sql`, `003_*.sql`, …) to `internal/platform/database/migrations/`; it runs automatically at startup and is tracked in `schema_migrations`. The team plans to move to **goose/atlas** — confirm current state before adding migrations (or use `/add-migration`). Do not hand-write `schema_migrations` rows.
- **Mocks** are generated with `mockgen` (`go.uber.org/mock`) into each package's `mocks/` dir; regenerate after changing an interface. (The `//go:generate`-style header comments have stale paths from the architecture refactor — trust the actual file location, not the comment.)
- **Running locally requires** `BROKER_USERNAME`, `BROKER_PASSWORD`, and `BROKER_TOPICS`. Copy `.env.example` → `.env` first.
- `.kiro/` is **gitignored** (personal steering docs + Kiro skills). The architecture rules above are mirrored here so they stay shared and version-controlled.

## Repo conventions

- Integration branch is `develop`; open PRs against it.
- PR title format: `[<service/package>] <Title>`.
- Commits follow Conventional Commits (`feat:`, `refactor:`, `chore:`, …).
