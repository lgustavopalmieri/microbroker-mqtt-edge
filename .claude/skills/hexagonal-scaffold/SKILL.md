---
name: hexagonal-scaffold
description: Scaffold a new module or feature in this repo's strict Clean/Hexagonal layout (domain → application → adapters). Use when creating a new bounded-context module under internal/modules, or a new feature within an existing module.
---

# Hexagonal scaffolder

Generate code that matches this repo's existing layout exactly. Read a sibling feature first (e.g. `internal/modules/audit/raw/features/count-by-topic/`) and mirror its package names, file names, and import style before writing anything.

Arguments (`$ARGUMENTS`): `<module>/<feature>` for a feature (e.g. `audit/raw/get-by-id`), or just `<module>` for a new module. If ambiguous, ask which.

## Canonical feature layout

```
internal/modules/<module>/[<sub>/]features/<feature>/
├── application/
│   ├── interface.go      # ports the use case needs (e.g. Repository) — the ONLY place infra is abstracted
│   ├── dto.go            # input/output DTOs
│   ├── new_usecase.go    # constructor only
│   ├── usecase.go        # business logic only
│   └── mocks/            # mockgen output for each port in interface.go
├── adapters/
│   ├── inbound/http_handler/
│   │   ├── di.go         # wiring (construct usecase + adapters)
│   │   ├── handler.go
│   │   └── handler_test.go
│   └── outbound/database/
│       ├── new.go        # constructor only
│       └── repository.go # implements a port from application/interface.go
```

Module-level domain (shared across the module's features) lives at
`internal/modules/<module>/[<sub>/]domain/` — entity, errors, status, validation. Domain depends on nothing external.

## Non-negotiable rules (see CLAUDE.md)

- Dependency flow is always inward: domain → application → adapters. Never the reverse.
- `domain/` and `application/` import zero infrastructure (`database/sql`, http, concrete `slog`/otel are all forbidden there).
- Application declares ports in `interface.go`; outbound adapters implement them.
- Handlers delegate to the use case only — no business logic in `handler.go`.
- Keep constructor and logic in separate files (`new_usecase.go` + `usecase.go`).
- Observability goes through interfaces in `internal/common`, never concrete loggers in domain/application.

## After scaffolding

1. Generate mocks for each new port: `mockgen -destination=<feature>/application/mocks/mock_<port>.go -package=mocks microbroker-mqtt-edge/<feature-application-import-path> <PortName>`
2. Run `go build ./...`, then `golangci-lint run ./...` and `go test ./...`. Fix until green.
3. Add table-driven tests for the use case happy path and edge cases.
