# STATE — Persistent Memory

> Cross-session memory for spec-driven work. Decisions, blockers, lessons, deferred ideas.

## Active Work

- **Feature: Availability (Disponibilidade)** — `.specs/features/availability/`
  - Phase: **Specify ✅ · Design ✅ · Tasks ✅** (spec.md, context.md, design.md, tasks.md, TESTING.md).
  - Next: **Execute** — 19 atomic tasks (T1..T19), 4 phases. Start at Phase 1 (T1 kernel, T2 migration, T3 go.mod).
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

## Preferences

- Repo gate before commit: `/verify-go` (build + vet + golangci-lint + `go test -race`). Pure-Go SQLite
  only (`modernc.org/sqlite`, `CGO_ENABLED=0`). Conventional Commits; PRs target `develop`.
