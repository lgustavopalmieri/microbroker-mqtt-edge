---
inclusion: always
description: Quick architecture map — for depth, activate the go-hexagonal-architecture skill.
---

# Architecture Quick Map (Go — Clean/Hexagonal)

For detailed patterns, templates, scaffolding guides, and verification commands, activate the `go-hexagonal-architecture` skill.

## Root Layout

```
cmd/<server>/bootstrap/    → Wiring only, zero business logic
cmd/<server>/config/       → Config struct, load, validate
internal/common/           → Shared interfaces (observability, events), value objects, test helpers
internal/modules/          → Business domain modules (one per bounded context)
internal/platform/         → Infrastructure adapters (database, broker, server, telemetry)
```

## Module Layout

```
internal/modules/<module>/
├── domain/                → Entity, status, errors, validation
│   └── <feature>/         → Feature-specific domain logic (separate Go package)
└── features/<feature>/
    ├── application/       → Use case, interface.go (ports), DTOs, constants, mocks/
    ├── adapters/inbound/  → http_handler/, grpc_service/
    ├── adapters/outbound/ → database/, <search-engine>/, external/
    └── event_listeners/   → Independent mini-modules: listener/ + adapters/
```

## Dependency Flow (always inward)

```
domain → domain/<feature> → application → adapters (inbound + outbound)
```

Never the reverse. Adapters depend on application interfaces. Application depends on domain. Domain depends on nothing external.

## Key Rules

- Domain and application layers: zero infrastructure imports
- Application defines ports in `interface.go`; adapters implement them
- Handlers (HTTP/gRPC) delegate to use cases only — no business logic
- Constructor and logic in separate files (`new_usecase.go` + `usecase.go`)
- Observability via interfaces — never concrete slog/otel in domain or application
- All infrastructure tools (databases, brokers, search engines) are swappable — only adapters change

## When to Activate the Skill

| Task | Skill Reference |
|------|-----------------|
| Create a module | `references/module-scaffolding.md` |
| Create a feature | `references/feature-scaffolding.md` |
| Understand principles | `references/principles.md` |
| Verify compliance | `references/verification.md` |
| Evaluate module split | `references/module-scaffolding.md` (Part 2) |
| Create event listeners | `references/feature-scaffolding.md` (Event Listeners) |
