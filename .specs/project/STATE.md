# STATE — Persistent Memory

> Cross-session memory for spec-driven work. Decisions, blockers, lessons, deferred ideas.

## Active Work

- **Feature: Availability (Disponibilidade)** — `.specs/features/availability/`
  - Phase: **Specify ✅ · Design ✅ · Tasks ✅** (spec.md, context.md, design.md, tasks.md, TESTING.md).
  - **T1 ✅ done** (`internal/modules/oee/domain/` — MachineState, Shift, Window, Break, PlannedProductionTime, 11 tests).
  - **T2 ✅ done** (`internal/platform/database/migrations/002_create_oee_availability.sql` — shifts, shift_breaks, state_intervals tables + 4 indexes; 9 integration tests).
  - **T3 ✅ done** (`go.mod` + `go.sum` — gorilla/websocket v1.5.3 pinned as direct require).
  - **T4 ✅ done** (`internal/modules/oee/availability/domain/` — StateInterval, AvailabilitySnapshot, Aggregate, Availability, 19 tests; §7 oracle verified).
  - **T5 ✅ done** (`internal/modules/oee/config/application/` — ShiftStore port, Seed use case, LoadShiftsFromJSON loader, mock, 10 tests).
  - **T6 ✅ done** (`internal/modules/oee/availability/features/ingest-state/application/` — IntervalStore+StateObserver ports, DecodeStateChange, Apply use case, 2 mocks, 13 tests).
  - **T7 ✅ done** (`internal/modules/oee/availability/features/live/application/` — AvailabilitySink+ShiftReader+StateIntervalReader ports, Engine with mutex-protected map, Apply+Start+rehydrate+tick, 3 mocks, 8 race-clean tests). StateTransition moved to avdomain.
  - Next actionable: **T8, T9** (T8 needs T4 ✅; T9 no deps; T13/T14 need T7 ✅ now unblocked).
  - Scope: Complex → full pipeline. WS lib locked: **gorilla/websocket**.
  - Note: harness policy = execute inline (no sub-agent spawning unless user asks).

## Key Decisions

- **2026-06-02** — Availability = live **+** on-demand over one **pure infra-free calc core**.
  Flow: `payload → persist → in-memory calc each ~1 s → emitted via single AvailabilitySink port`.
  Why: user wants the per-second in-memory calc with **one update interface ready now**; concrete
  sinks (DB snapshot, **websocket — flagged extremely important**, MQTT republish) plug in later (P2).
- **2026-06-02** — Shift/break config lives in **SQLite via migration 002**, seeded at startup (CRUD = P3).
- **2026-06-02** — On-demand REST uses an **arbitrary `[from,to]` window**.
- **2026-06-02** — `machine_id` from **payload** (topics are exact-match, max 5, no wildcards); the
  availability worker self-filters `state_change` since fan-out broadcasts every message to every worker.
- **2026-06-02** — Topics are **by data type**, not per machine: `machine/state, machine/production,
  machine/cycle, machine/quality` — scalable to full OEE within the 5-topic cap (`.env.example` must free a slot).
- **2026-06-02** — **Websocket sink pulled into P1** ("já vamos fazer o websocket"). Lib choice:
  `github.com/coder/websocket` (pure Go, zero transitive deps → keeps static `CGO_ENABLED=0` build).
  First third-party runtime dep — flag at Execute.
- **2026-06-02** — New bounded context **`internal/modules/oee/`** with shared kernel + `availability/`
  sub-context; Performance/Quality will be siblings. Migration `002` = shifts/shift_breaks/state_intervals
  (no triggers — migrator splits on `;`).

## Blockers

- None. Open design question: a `machine/state` topic must fit within the 5-topic exact-match cap
  (current `.env.example` already uses all 5 slots).

## Deferred Ideas

- Performance, Quality, full OEE product, production orders (separate features).
- Websocket server/transport (port designed in P1; sink impl is P2).
- goose/atlas migrator move (user memory: `migrations-move-to-goose-atlas`) — independent of this feature.

## Tooling

- **`/ship:feature [feature]`** — standalone runner: ships the **next** actionable task (deps ✅) of a
  feature; implement → gate → atomic commit → flip ledger ✅ → stop. Defaults to active feature here.
- **`/ship:task <feature> <Tn>`** — ships a **specific** task by ID (warns on unmet deps); for parallel
  team pickup of a big spec. Both live in `.claude/commands/ship/`, are standalone (don't load the skill),
  and maintain the **Progress Ledger** in each feature's `tasks.md`. Pair with `/loop` for autonomous runs.

## Preferences

- Repo gate before commit: `/verify-go` (build + vet + golangci-lint + `go test -race`). Pure-Go SQLite
  only (`modernc.org/sqlite`, `CGO_ENABLED=0`). Conventional Commits; PRs target `develop`.
- Per-task commits must be **atomic and tell that task's story** (a large task may split into a few
  logical commits, each building green).
