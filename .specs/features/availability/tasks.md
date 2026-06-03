# Availability Tasks

**Design**: `.specs/features/availability/design.md`
**Spec**: `.specs/features/availability/spec.md` (P1 = AVAIL-01..19, 21, 25)
**Testing**: `.specs/codebase/TESTING.md`
**Status**: Approved — ready to ship via `/ship:feature availability` (or `/ship:task availability T<n>`)

Scaffolding: use `/hexagonal-scaffold` per feature (domain→application(`interface.go`)→adapters), `/add-migration` for T2, `test-expert` for tests, `go-concurrency-patterns` for the engine (T7), `/verify-go` for the full gate. No MCPs configured (MCP: NONE everywhere).

---

## Progress Ledger

> Maintained by `/ship:*`. **⬜ Pending · 🔄 In progress · ✅ Done.** A task is actionable when **all** its deps are ✅.
> (No T15 — the websocket sink + endpoint were merged into T14.)

| Task | Status | Depends on | Actionable now? |
| --- | --- | --- | --- |
| T1 — shared kernel | ✅ | — | ✅ yes |
| T2 — migration 002 | ✅ | — | ✅ yes |
| T3 — gorilla dep | ✅ | — | ✅ yes |
| T4 — calculator | ✅ | T1 | ✅ yes |
| T5 — config app | ⬜ | T1 | ✅ yes |
| T6 — ingest app | ⬜ | T4 | — |
| T7 — live engine | ⬜ | T4 | — |
| T8 — query app | ⬜ | T4 | — |
| T9 — env vars | ⬜ | — | ✅ yes |
| T10 — shift repo | ⬜ | T2, T5 | — |
| T11 — interval repo | ⬜ | T2, T6 | — |
| T12 — worker | ⬜ | T6 | — |
| T13 — log/composite sinks | ⬜ | T7 | — |
| T14 — websocket sink+endpoint | ⬜ | T3, T7 | — |
| T16 — query repo | ⬜ | T2, T8 | — |
| T17 — query handler | ⬜ | T8 | — |
| T18 — wiring + e2e | ⬜ | T9,T10,T11,T12,T13,T14,T16,T17 | — |
| T19 — docs | ⬜ | T18 | — |

**Next up:** T2, T3, T4, T5, or T9 (no unmet deps).

---

## Execution Plan

### Phase 1 — Foundation
```
T1 [P] (kernel) ──┐
T2 [P] (migration)│
T3 [P] (go.mod ws)│
                  └─→ T4 (availability domain + calculator)   [needs T1]
```

### Phase 2 — Application layers (parallel, all need domain)
```
        ┌→ T5 [P] config app        [needs T1]
T4 ─────┼→ T6 [P] ingest-state app  [needs T4]
        ├→ T7 [P] live app/engine   [needs T4]
        └→ T8 [P] query app         [needs T4]
T9 [P] config env vars              [no deps]
```

### Phase 3 — Adapters (parallel)
```
T2,T5 → T10 [P] config repo
T2,T6 → T11 [P] interval repo
T6   → T12 [P] ingest-state worker
T7   → T13 [P] composite+log sinks
T3,T7→ T14 [P] websocket sink+hub+endpoint
T2,T8 → T16 [P] query repo
T8   → T17 [P] query http handler
```

### Phase 4 — Integration (sequential)
```
T10,T11,T12,T13,T14,T16,T17,T9 → T18 (bootstrap wiring + e2e) → T19 (docs)
```

---

## Task Breakdown

### T1: Shared OEE domain kernel [P]
**What**: `MachineState` (+`IsDowntime`/`IsPlannedStop`/`Valid`), `Window`, `Break`, `Shift` with `PlannedProductionTime(Window)` resolving recurring shift occurrences minus breaks.
**Where**: `internal/modules/oee/domain/`
**Depends on**: None
**Reuses**: research §6.1 definitions
**Requirement**: AVAIL-09, AVAIL-10
**Tools**: MCP NONE · Skill `hexagonal-scaffold`, `test-expert`
**Done when**:
- [ ] `IsDowntime`=`stopped|setup|maintenance`, `IsPlannedStop`=`setup|maintenance` per §6.1
- [ ] `PlannedProductionTime` correct for single-day, multi-day, weekday-masked, break-overlapping windows
- [ ] Gate passes: `go test ./internal/modules/oee/domain/...`
- [ ] Test count: ≥6 table-driven cases pass
**Tests**: unit · **Gate**: quick
**Commit**: `feat(oee): add shared OEE domain kernel`

### T2: Migration 002 — OEE availability schema [P]
**What**: `002_create_oee_availability.sql` creating `shifts`, `shift_breaks`, `state_intervals` (+ indexes). CREATE TABLE/INDEX only (migrator splits on `;` — no triggers).
**Where**: `internal/platform/database/migrations/002_create_oee_availability.sql` (+ migration-apply test)
**Depends on**: None
**Reuses**: `001_create_raw_data.sql`, `database.Migrator`
**Requirement**: AVAIL-01, AVAIL-07 (schema)
**Tools**: MCP NONE · Skill `add-migration`, `test-expert`
**Done when**:
- [ ] Three tables + indexes per design data-model section
- [ ] Migrator applies idempotently on a temp DB; re-run is a no-op (tracked in `schema_migrations`)
- [ ] Gate passes: `go test ./internal/platform/database/...`
- [ ] Test count: existing migrator tests + ≥1 new pass
**Tests**: integration · **Gate**: quick
**Commit**: `feat(oee): add migration 002 for availability schema`

### T3: Add gorilla/websocket dependency [P]
**What**: `go get github.com/gorilla/websocket` + `go mod tidy`; isolate to module graph only.
**Where**: `go.mod`, `go.sum`
**Depends on**: None
**Reuses**: —
**Requirement**: AVAIL-21 (enabler)
**Tools**: MCP NONE · Skill NONE
**Done when**:
- [ ] `github.com/gorilla/websocket` is a direct require; `go mod tidy` clean
- [ ] Static build still works: `CGO_ENABLED=0 go build ./...`
- [ ] Gate passes: `go build ./...`
**Tests**: none · **Gate**: build
**Commit**: `chore(oee): add gorilla/websocket dependency`

### T4: Availability domain + pure calculator
**What**: `StateInterval`, `AvailabilitySnapshot`, and pure `Aggregate(Window, []StateInterval, planned) Input` + `Availability(Input) Result` (clamped `[0,1]`, guards planned==0).
**Where**: `internal/modules/oee/availability/domain/`
**Depends on**: T1
**Reuses**: research §6.3 (Availability portion); kernel from T1
**Requirement**: AVAIL-08, AVAIL-09
**Tools**: MCP NONE · Skill `hexagonal-scaffold`, `test-expert`
**Done when**:
- [ ] Reproduces research §7 oracle exactly: planned 420m, stop 47m → Availability `0.8881±0.0001`
- [ ] Interval clipping, planned/unplanned split, open-interval-to-window-end, planned==0→`HasData=false`
- [ ] Imports zero infra (no `database/sql`, no http, no concrete logger)
- [ ] Gate passes: `go test ./internal/modules/oee/availability/domain/...`
- [ ] Test count: ≥8 cases pass
**Tests**: unit · **Gate**: quick
**Commit**: `feat(oee): add pure availability calculator`

### T5: Config application — shift seed use case + ports [P]
**What**: `ShiftStore` port (`interface.go`: `Upsert`, `ForMachineWindow`), `Seed` use case, JSON shift/break loader, dto, generated mocks.
**Where**: `internal/modules/oee/config/application/` (+ `mocks/`)
**Depends on**: T1
**Reuses**: audit application pattern; `Shift` kernel
**Requirement**: AVAIL-02, AVAIL-03
**Tools**: MCP NONE · Skill `hexagonal-scaffold`, `test-expert`
**Done when**:
- [ ] Loader parses a JSON shifts file into `[]Shift`; invalid JSON → error
- [ ] `Seed` upserts each shift via mocked `ShiftStore`
- [ ] Gate passes: `go test ./internal/modules/oee/config/...`
- [ ] Test count: ≥4 cases pass
**Tests**: unit · **Gate**: quick
**Commit**: `feat(oee): add shift config seed use case`

### T6: Ingest-state application — transition→interval use case + ports [P]
**What**: `IntervalStore` + `StateObserver` ports (`interface.go`), `StateChange` decode+validate (research §6.2), `Apply` use case (close open interval, open new, persist, notify observer), dto, mocks.
**Where**: `internal/modules/oee/availability/features/ingest-state/application/` (+ `mocks/`)
**Depends on**: T4
**Reuses**: `StateInterval`, `MachineState` (T1/T4)
**Requirement**: AVAIL-04, AVAIL-06
**Tools**: MCP NONE · Skill `hexagonal-scaffold`, `test-expert`
**Done when**:
- [ ] Valid transition closes prior + opens new interval and calls `StateObserver.Apply`
- [ ] No-op transition ignored; non-increasing timestamp rejected/clipped; missing fields → error (no panic)
- [ ] Gate passes: `go test ./internal/modules/oee/availability/features/ingest-state/...`
- [ ] Test count: ≥6 cases pass
**Tests**: unit · **Gate**: quick
**Commit**: `feat(oee): add state-change ingestion use case`

### T7: Live application — engine, ticker, AvailabilitySink port [P]
**What**: `AvailabilitySink` port (`Update(ctx, AvailabilitySnapshot) error`), `AvailabilitySnapshot` dto (JSON-ready), `ShiftReader` port, `Engine` (`Apply` + `Start(ctx)` ticker with injectable clock + interval), mocks.
**Where**: `internal/modules/oee/availability/features/live/application/` (+ `mocks/`)
**Depends on**: T4
**Reuses**: pure calculator (T4), kernel (T1)
**Requirement**: AVAIL-11, AVAIL-12, AVAIL-13 (port), AVAIL-15
**Tools**: MCP NONE · Skill `hexagonal-scaffold`, `go-concurrency-patterns`, `test-expert`
**Done when**:
- [ ] `Apply` updates per-machine state; concurrent `Apply`+tick is `-race` clean
- [ ] Each tick builds a snapshot via the calculator and calls `AvailabilitySink.Update` (fake sink asserts monotonic updates as downtime accrues)
- [ ] Tick interval configurable (default 1s); rehydrates open interval via `IntervalStore.LastOpen` on start
- [ ] Gate passes: `go test -race ./internal/modules/oee/availability/features/live/...`
- [ ] Test count: ≥6 cases pass
**Tests**: unit · **Gate**: quick (race-enabled)
**Commit**: `feat(oee): add live availability engine and sink port`

### T8: Query application — on-demand use case + ports [P]
**What**: `IntervalReader` + `ShiftReader` ports (`interface.go`), `Execute(ctx, machine, Window)` use case, output dto with coverage flags, mocks.
**Where**: `internal/modules/oee/availability/features/query/application/` (+ `mocks/`)
**Depends on**: T4
**Reuses**: pure calculator (T4)
**Requirement**: AVAIL-16, AVAIL-17, AVAIL-18
**Tools**: MCP NONE · Skill `hexagonal-scaffold`, `test-expert`
**Done when**:
- [ ] Reads intervals + planned time (mocked), calls calculator, returns Output with `availability`, times, downtime split, `interval_count`, flags
- [ ] `no_data`, `shift_config_missing`, `open_interval_clipped` flags set correctly
- [ ] §7 oracle reproduced end-to-end through the use case
- [ ] Gate passes: `go test ./internal/modules/oee/availability/features/query/...`
- [ ] Test count: ≥6 cases pass
**Tests**: unit · **Gate**: quick
**Commit**: `feat(oee): add availability query use case`

### T9: OEE config env vars [P]
**What**: Add `BROKER_OEE_ENABLED`, `BROKER_OEE_STATE_TOPIC`, `BROKER_OEE_TICK_INTERVAL`, `BROKER_OEE_WS_ENABLED`, `BROKER_OEE_SHIFTS_PATH` to `Config` with defaults + validation.
**Where**: `cmd/broker/config/config.go` (+ `config_test.go`)
**Depends on**: None
**Reuses**: existing `envOrDefault`/`envIntOrDefault` helpers
**Requirement**: AVAIL-03, AVAIL-12 (config surface)
**Tools**: MCP NONE · Skill `test-expert`
**Done when**:
- [ ] Defaults applied (state topic `machine/state`, tick `1s`, ws on); tick parses Go duration
- [ ] Validation: if OEE enabled, `BROKER_OEE_STATE_TOPIC` must be within `BROKER_TOPICS`
- [ ] Gate passes: `go test ./cmd/broker/config/...`
- [ ] Test count: existing + ≥3 new cases pass
**Tests**: unit · **Gate**: quick
**Commit**: `feat(oee): add OEE config env vars`

### T10: Config outbound repo — shifts/shift_breaks [P]
**What**: SQLite repo implementing `ShiftStore` (`Upsert`, `ForMachineWindow`) over `shifts`/`shift_breaks`.
**Where**: `internal/modules/oee/config/adapters/outbound/database/`
**Depends on**: T2, T5
**Reuses**: `ingestion/repository.go` tx+mutex pattern
**Requirement**: AVAIL-02, AVAIL-03
**Tools**: MCP NONE · Skill `test-expert`
**Done when**:
- [ ] `Upsert` persists shift + breaks; `ForMachineWindow` returns planned time for a window (and per-machine vs `*` fallback)
- [ ] Integration test on a temp DB (migration 002 applied)
- [ ] Gate passes: `go test ./internal/modules/oee/config/...`
- [ ] Test count: ≥4 cases pass
**Tests**: integration · **Gate**: quick
**Commit**: `feat(oee): add shift config repository`

### T11: Ingest-state outbound repo — state_intervals [P]
**What**: SQLite repo implementing `IntervalStore` (`OpenInterval`, `CloseOpen`, `LastOpen`).
**Where**: `internal/modules/oee/availability/features/ingest-state/adapters/outbound/database/`
**Depends on**: T2, T6
**Reuses**: `ingestion/repository.go` tx+mutex pattern
**Requirement**: AVAIL-06, AVAIL-07
**Tools**: MCP NONE · Skill `test-expert`
**Done when**:
- [ ] `OpenInterval` inserts `ended_at NULL`; `CloseOpen` updates the open row; `LastOpen` returns it
- [ ] Restart scenario: open interval survives and is read back
- [ ] Integration test on a temp DB (migration 002 applied)
- [ ] Gate passes: `go test ./internal/modules/oee/availability/features/ingest-state/...`
- [ ] Test count: ≥4 cases pass
**Tests**: integration · **Gate**: quick
**Commit**: `feat(oee): add state-interval repository`

### T12: Ingest-state inbound worker [P]
**What**: `fanout.Worker` that self-filters to the state topic, decodes the payload, and delegates to the `Apply` use case; non-state messages return nil.
**Where**: `internal/modules/oee/availability/features/ingest-state/adapters/inbound/worker/`
**Depends on**: T6
**Reuses**: `fanout.Worker`, `LoggerWorker` pattern
**Requirement**: AVAIL-05
**Tools**: MCP NONE · Skill `test-expert`
**Done when**:
- [ ] `Process` ignores non-state topics (returns nil); decodes + delegates state messages
- [ ] Malformed payload logged + skipped, never panics; `Name()`/`Close()` implemented
- [ ] Gate passes: `go test ./internal/modules/oee/availability/features/ingest-state/...`
- [ ] Test count: ≥5 cases pass
**Tests**: unit · **Gate**: quick
**Commit**: `feat(oee): add state-change worker`

### T13: Composite + Log sinks [P]
**What**: `CompositeSink` (fan-out with per-sink failure/slowness isolation) and `LogSink` (default; logs + holds latest snapshot per machine).
**Where**: `internal/modules/oee/availability/features/live/adapters/outbound/sink/{composite,logsink}/`
**Depends on**: T7
**Reuses**: `observability.Logger`; `AvailabilitySink` port (T7)
**Requirement**: AVAIL-13, AVAIL-14
**Tools**: MCP NONE · Skill `go-concurrency-patterns`, `test-expert`
**Done when**:
- [ ] Composite delivers to all sinks; a failing/slow sink does not block others (asserted with a blocking fake)
- [ ] LogSink stores + returns latest snapshot per machine
- [ ] Gate passes: `go test -race ./internal/modules/oee/availability/features/live/...`
- [ ] Test count: ≥5 cases pass
**Tests**: unit · **Gate**: quick (race-enabled)
**Commit**: `feat(oee): add composite and log availability sinks`

### T14: Websocket sink + hub + endpoint [P]
**What**: `WebsocketSink` implementing `AvailabilitySink` over a connection **hub** keyed by `machine_id` (gorilla/websocket, non-blocking, drop-on-slow), plus the `GET /availability/{machine}/ws` upgrade handler that registers conns with the hub.
**Where**: `internal/modules/oee/availability/features/live/adapters/{outbound/sink/websocket,inbound/http_handler}/`
**Depends on**: T3, T7
**Reuses**: `AvailabilitySink` port (T7); `RegisterRoutes(mux)` pattern
**Requirement**: AVAIL-21, AVAIL-25
**Tools**: MCP NONE · Skill `go-concurrency-patterns`, `test-expert`
**Done when**:
- [ ] `Update(snapshot)` broadcasts JSON to clients subscribed to that machine
- [ ] Endpoint upgrades + subscribes; slow/closed client dropped without blocking the hub (asserted via `httptest.Server` + gorilla dialer)
- [ ] Gate passes: `go test -race ./internal/modules/oee/availability/features/live/...`
- [ ] Test count: ≥4 cases pass
**Tests**: unit · **Gate**: quick (race-enabled)
**Commit**: `feat(oee): add websocket availability sink and endpoint`

### T16: Query outbound repo — interval reader [P]
**What**: SQLite repo implementing `IntervalReader.ByMachineRange(ctx, machine, from, to)` returning intervals overlapping the window (open interval included).
**Where**: `internal/modules/oee/availability/features/query/adapters/outbound/database/`
**Depends on**: T2, T8
**Reuses**: `ingestion/repository.go` read pattern
**Requirement**: AVAIL-17
**Tools**: MCP NONE · Skill `test-expert`
**Done when**:
- [ ] Returns closed + open intervals overlapping `[from,to]`; excludes non-overlapping
- [ ] Integration test on a temp DB (migration 002 applied)
- [ ] Gate passes: `go test ./internal/modules/oee/availability/features/query/...`
- [ ] Test count: ≥4 cases pass
**Tests**: integration · **Gate**: quick
**Commit**: `feat(oee): add interval reader repository`

### T17: Query inbound HTTP handler [P]
**What**: `GET /availability/{machine}?from=&to=` handler: validate params, call use case, render JSON, map errors (400/200/500) per audit conventions.
**Where**: `internal/modules/oee/availability/features/query/adapters/inbound/http_handler/`
**Depends on**: T8
**Reuses**: audit `http_handler` pattern + test style
**Requirement**: AVAIL-18, AVAIL-19
**Tools**: MCP NONE · Skill `test-expert`
**Done when**:
- [ ] Missing/unparseable/`from>=to` → 400 JSON; valid → 200 with snapshot JSON; use-case error → 500
- [ ] No-data window → 200 with `no_data:true`
- [ ] Gate passes: `go test ./internal/modules/oee/availability/features/query/...`
- [ ] Test count: ≥5 cases pass (httptest + mock use case)
**Tests**: unit · **Gate**: quick
**Commit**: `feat(oee): add availability query HTTP handler`

### T18: Bootstrap wiring + e2e
**What**: Wire everything: `InitModules` builds repos/engine/sinks (composite=log+ws)/worker (appended to `allWorkers`) and runs the shift seed; `StartServers` registers query + ws routes and starts the engine ticker goroutine; `GracefulShutdown` closes engine/sinks. Add an e2e test under `tests/`.
**Where**: `cmd/broker/bootstrap/{modules.go,server.go,shutdown.go}`, `tests/`
**Depends on**: T9, T10, T11, T12, T13, T14, T16, T17
**Reuses**: existing bootstrap wiring + `tests/` harness
**Requirement**: AVAIL-01..19, 21, 25 (integration)
**Tools**: MCP NONE · Skill `test-expert`, `verify-go`
**Done when**:
- [ ] e2e: publish `running`→`stopped`→`running`; assert intervals persisted, `GET /availability/{m}?from&to` returns expected Availability, a ws client receives a snapshot
- [ ] Seed runs at startup from `BROKER_OEE_SHIFTS_PATH`; OEE disabled flag bypasses wiring cleanly
- [ ] Full gate passes: `go build ./... && go vet ./... && golangci-lint run ./... && go test -race ./...`
- [ ] Test count: full suite green; new e2e passes
**Tests**: e2e · **Gate**: full
**Commit**: `feat(oee): wire availability module and add e2e`

### T19: Docs — env, README, example
**What**: Document the new env vars, `GET /availability/{m}?from&to`, `…/ws`, and the topic-by-type scheme; free a topic slot for `machine/state` in `.env.example`.
**Where**: `README.md`, `.env.example`, `docs/`
**Depends on**: T18
**Reuses**: existing README tables
**Requirement**: AVAIL-01 (operability)
**Tools**: MCP NONE · Skill NONE
**Done when**:
- [ ] README documents endpoints + env vars; `.env.example` includes `machine/state` within the 5-topic cap
- [ ] Gate passes: `go build ./...`
**Tests**: none · **Gate**: build
**Commit**: `docs(oee): document availability endpoints and config`

---

## Validation Tables

### Check 1 — Granularity
| Task | Scope | Status |
| --- | --- | --- |
| T1 kernel | 1 domain pkg (cohesive value objects) | ✅ |
| T2 migration | 1 SQL file + test | ✅ |
| T3 go.mod | 1 dep | ✅ |
| T4 calculator | 1 domain pkg | ✅ |
| T5–T8 app layers | 1 use case + ports each | ✅ |
| T9 env vars | 1 file | ✅ |
| T10,T11,T16 repos | 1 repo each | ✅ |
| T12 worker | 1 adapter | ✅ |
| T13 sinks | 2 cohesive sinks, same port | ✅ (OK if cohesive) |
| T14 ws sink+endpoint | 1 cohesive ws unit (shared hub) | ✅ (OK if cohesive) |
| T17 handler | 1 endpoint | ✅ |
| T18 wiring+e2e | integration task (by design) | ✅ |
| T19 docs | docs only | ✅ |

### Check 2 — Diagram ↔ Definition Cross-Check
| Task | Depends On (body) | Diagram | Status |
| --- | --- | --- | --- |
| T1 | none | none | ✅ |
| T2 | none | none | ✅ |
| T3 | none | none | ✅ |
| T4 | T1 | T1→T4 | ✅ |
| T5 | T1 | T4-phase needs T1 (T1→T5) | ✅ |
| T6 | T4 | T4→T6 | ✅ |
| T7 | T4 | T4→T7 | ✅ |
| T8 | T4 | T4→T8 | ✅ |
| T9 | none | standalone [P] | ✅ |
| T10 | T2,T5 | T2,T5→T10 | ✅ |
| T11 | T2,T6 | T2,T6→T11 | ✅ |
| T12 | T6 | T6→T12 | ✅ |
| T13 | T7 | T7→T13 | ✅ |
| T14 | T3,T7 | T3,T7→T14 | ✅ |
| T16 | T2,T8 | T2,T8→T16 | ✅ |
| T17 | T8 | T8→T17 | ✅ |
| T18 | T9,T10,T11,T12,T13,T14,T16,T17 | all→T18 | ✅ |
| T19 | T18 | T18→T19 | ✅ |

### Check 3 — Test Co-location
| Task | Layer | Matrix Requires | Task Says | Status |
| --- | --- | --- | --- | --- |
| T1 | domain | unit | unit | ✅ |
| T2 | migration | integration | integration | ✅ |
| T3 | go.mod | none | none | ✅ |
| T4 | domain | unit | unit | ✅ |
| T5 | application | unit | unit | ✅ |
| T6 | application | unit | unit | ✅ |
| T7 | application | unit | unit (race) | ✅ |
| T8 | application | unit | unit | ✅ |
| T9 | config | unit | unit | ✅ |
| T10 | outbound db | integration | integration | ✅ |
| T11 | outbound db | integration | integration | ✅ |
| T12 | inbound worker | unit | unit | ✅ |
| T13 | outbound sink | unit | unit (race) | ✅ |
| T14 | sink + inbound http | unit | unit (race) | ✅ |
| T16 | outbound db | integration | integration | ✅ |
| T17 | inbound http | unit | unit | ✅ |
| T18 | bootstrap wiring | e2e | e2e | ✅ |
| T19 | docs | none | none | ✅ |

All three checks pass — tasks ready for approval.

---

## Parallel Execution Map
```
Phase 1:  T1[P]  T2[P]  T3[P]   then  T4
Phase 2:  T5[P]  T6[P]  T7[P]  T8[P]  T9[P]
Phase 3:  T10[P] T11[P] T12[P] T13[P] T14[P] T16[P] T17[P]
Phase 4:  T18  →  T19
```
e2e (T18) is **not** parallel-safe (ports + DB file) → sequential. All Phase-2/3 unit & integration tests are parallel-safe (isolated packages / temp DBs).

## Requirement Coverage
25 P1 IDs → all mapped. AVAIL-01→T2; 02→T5,T10; 03→T5,T9,T10; 04→T6; 05→T12; 06→T6,T11; 07→T2,T11; 08→T4; 09→T1,T4; 10→T1; 11→T7; 12→T7,T9; 13→T7,T13; 14→T13; 15→T7; 16→T8; 17→T8,T16; 18→T8,T17; 19→T17; 21→T14; 25→T14. (Integration proof: T18.)
