---
name: go-hexagonal-architecture
description: >
  Go hexagonal/clean architecture expert. ALWAYS read this skill BEFORE proposing any fix or plan
  that involves: module structure, domain layer design, feature creation, adapter patterns,
  event listener wiring, repository ownership, cross-layer dependencies, dependency injection,
  or any structural change to a module or feature. The patterns here are project-specific and
  differ from generic Go conventions — reasoning from first principles without reading this skill
  will produce incorrect plans (e.g. putting business logic in adapters, mixing inbound/outbound
  concerns, or violating the dependency flow). Triggers: creating modules, creating features,
  "scaffold a module", "scaffold a feature", assessing architecture compliance, evaluating module
  boundaries, understanding hexagonal principles, fixing cross-layer imports, reviewing module
  structure, PR review of architecture comments, or any mention of "architecture assessment",
  "module boundaries", "hexagonal", "clean architecture", "compliance check", "maturity assessment",
  "create a module", "create a feature", "cross-boundary", "adapters", "inbound", "outbound",
  "use case", "domain layer", "event listener", "repository ownership", "dependency flow".
---

# Go Hexagonal Architecture Expert

You are an expert in this project's Go hexagonal/clean architecture. This skill provides everything needed for architecture design, module creation, feature scaffolding, evaluation, and compliance assessment.

## Core Philosophy

- `cmd/` = Bootstrap (orchestration only, zero business logic)
- `internal/modules/` = Business domain modules (one per bounded context)
- `internal/common/` = Shared cross-module utilities (interfaces, value objects, test helpers)
- `internal/platform/` = Infrastructure adapters (database, broker, server, telemetry)

## Architecture Layers (per module)

```
domain/          → Entity, status, errors, validation, feature-specific sub-packages
features/        → One folder per feature, each with application/ + adapters/
  application/   → Use case orchestration, interfaces (ports), DTOs
  adapters/
    inbound/     → Driving adapters (HTTP handlers, gRPC services)
    outbound/    → Driven adapters (database repos, search engine repos, external gateways)
  event_listeners/ → Event-driven side effects (optional, independent mini-modules)
```

## Dependency Flow (CRITICAL — always inward, never reverse)

```
domain (entity, status, errors, validate)
    ↑
domain/<feature>/ (factory, validation, feature-specific errors)
    ↑
features/<feature>/application/ (use case, interfaces, DTOs)
    ↑
features/<feature>/adapters/inbound/ (http_handler, grpc_service)
features/<feature>/adapters/outbound/ (database, search-engine)
```

## The 8 Principles

| # | Principle | Criticality | Key Rule |
|---|-----------|-------------|----------|
| 1 | Well-defined boundaries | High | Each module owns its domain; no cross-module imports at domain level |
| 2 | Composability | Medium | Modules are independent and composable via interfaces |
| 3 | Independence | High | Modules can evolve, test and deploy independently |
| 4 | Explicit communication | High | Inter-module communication via events or well-defined contracts |
| 5 | Replaceability | Medium | Adapters are swappable without touching domain or application logic |
| 6 | State isolation | 🔴 CRITICAL | No shared mutable state between modules; each module owns its data |
| 7 | Observability | High | Logging, tracing and metrics via interfaces — never concrete in domain/application |
| 8 | Fail independence | High | One module's failure must not cascade to others |

## Top 8 Critical Violations

1. 🔴 Business logic in adapters — handlers/repositories must NOT contain domain rules
2. 🔴 Domain importing infrastructure — domain/ must NEVER import from adapters/ or platform/
3. 🔴 Application layer importing concrete adapters — application/ depends on interfaces only
4. 🔴 Cross-module domain imports — module A's domain must NOT import module B's domain
5. 🟠 Fat handlers — HTTP/gRPC handlers with business logic instead of delegating to use cases
6. 🟠 Missing interface.go — application/ and listener/ must define port interfaces, not depend on concrete types
7. 🟠 Concrete observability in application — use case must inject logger/tracer interfaces, not slog/otel directly
8. 🟠 Mocks alongside production code — mocks belong in `mocks/` sub-folders, not mixed with source files

## Decision Tree: Which Reference to Load

```
TASK TYPE                                    → LOAD REFERENCE
──────────────────────────────────────────────────────────────
Creating a new module from scratch           → references/module-scaffolding.md (Part 1)
Evaluating whether to split a module         → references/module-scaffolding.md (Part 2)
Creating a new feature within a module       → references/feature-scaffolding.md
Creating event listeners for a feature       → references/feature-scaffolding.md (Event Listeners section)
Understanding a specific principle           → references/principles.md
Assessing architecture compliance            → references/verification.md
Running detection commands                   → references/verification.md
Maturity scoring                             → references/verification.md
```

## Use Case Instructions

### Creating a New Module

Load `references/module-scaffolding.md` — Part 1: Module Creation.

1. Gather requirements (module name, entities, features, external integrations)
2. Generate domain layer (entity, status, errors, validation)
3. Generate feature-specific domain sub-packages
4. For each feature, follow `references/feature-scaffolding.md`
5. Run verification commands from `references/verification.md`

### Creating a New Feature

Load `references/feature-scaffolding.md`.

1. Determine feature type (CRUD write, read/search, event-driven)
2. Create domain sub-package if needed
3. Create application/ (use case, interfaces, DTOs, constants, mocks)
4. Create adapters/inbound/ (http_handler and/or grpc_service)
5. Create adapters/outbound/ (database and/or search-engine)
6. Create event_listeners/ if the feature emits domain events
7. Run verification

### Evaluating Whether to Split a Module

Load `references/module-scaffolding.md` — Part 2: Module Evaluation.

Apply the 6-criteria test and cohesion/coupling scoring.

### Assessing Architecture Compliance

Load `references/verification.md`.

Run all detection commands, score each principle, produce prioritized report.

### Understanding a Specific Principle

Load `references/principles.md`.

Each principle includes: definition, rules for AI agents, Go code examples, and common violations.

## Quick Anti-Pattern Check

Before generating any code, verify:

- [ ] Domain layer has ZERO imports from `adapters/`, `platform/`, or external infrastructure packages
- [ ] Application layer depends ONLY on interfaces defined in its own `interface.go`
- [ ] Handlers (HTTP/gRPC) only call use case methods — no business logic
- [ ] Repositories implement interfaces defined in application/ — not the reverse
- [ ] Each Go package has a single clear responsibility
- [ ] Constructor and logic are in separate files (`new_usecase.go` + `usecase.go`)
- [ ] Mocks live in `mocks/` sub-folders, generated via gomock
- [ ] `dto.go` exists in application, listener, and all adapter packages (no shared DTOs across layers)
- [ ] `constants.go` holds span names, event names, error messages (application and listener only)
- [ ] `errors.go` holds domain errors (domain), storage errors (outbound adapters)
- [ ] Event listeners are independent mini-modules with their own `listener/ + adapters/`

> **Note**: All infrastructure tools in this skill (PostgreSQL, Elasticsearch, Kafka, slog, OpenTelemetry, etc.) are examples. The architecture is interface-driven — any tool can be swapped or added at any time. Only the adapter implementations change; domain and application layers remain untouched.
