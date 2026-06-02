# Availability (Disponibilidade) Specification

> First OEE factor for `microbroker-mqtt-edge`. Source of truth for the domain model:
> `docs/industry-4.0-metrics-research.md` (§3.1.1, §4.1.2, §6, §9).
> OEE = **Availability** × Performance × Quality. This feature delivers **only Availability**.

## Problem Statement

The broker ingests machine telemetry, persists raw payloads to `raw_data`, and fans them out to
workers — but it computes **no** OEE metrics. Availability is the first and most data-light factor:
it needs only shift time, scheduled breaks, and machine stop events (`state_change`). Today there is
no place to register shifts, no typed model of machine state over time, and no calculation. We want
the broker to (a) capture machine state transitions as durable intervals, (b) compute Availability
**both live** (in-memory, ~1 s cadence, pushed through a pluggable output sink) **and on-demand**
(REST, arbitrary `[from,to]` window), and (c) register shifts & breaks in SQLite — while keeping the
calculation a **pure, infrastructure-free domain service** that can be injected into a worker, a REST
use case, a websocket publisher, or a DB writer without change.

## Goals

- [ ] **Pure Availability calculation** as an infra-free domain service, reused by every path (live + on-demand). `Availability = RunTime / PlannedProductionTime`, faithful to research §3.1.1.
- [ ] **Shifts & scheduled breaks persisted** via migration `002`, seeded at startup, queryable by machine + time window.
- [ ] **Machine `state_change` events normalized into durable state intervals** per machine (transition pairing), persisted for historical queries.
- [ ] **Live in-memory accumulator** per machine that updates on each event, **ticks every ~1 s**, and emits the current Availability snapshot through a **single `AvailabilitySink` output port** — with ≥1 default sink wired and the port explicitly designed so DB-snapshot, **websocket**, and MQTT-republish sinks plug in later without touching the core.
- [ ] **On-demand REST endpoint** computing Availability for an arbitrary `[from,to]` window for one machine.
- [ ] Reuses the existing ingestion/fan-out pipeline — every payload is still persisted exactly as today; this feature **adds** typed state intervals + the calculation, it does not rebuild ingestion.

## Out of Scope

| Feature | Reason |
| --- | --- |
| Performance & Quality factors, full OEE product | Separate features; Availability ships first (research §9). |
| Production orders, piece-count, cycle-time, reject ingestion | Belong to Performance/Quality/Order features (the `oee` module is laid out to host them next). |
| **DB-snapshot** sink + `availability_snapshots` persistence | P2 — a second sink behind the same port. |
| **MQTT re-publish** sink | P2 — a third sink behind the same port. |
| Shift/break **CRUD UI or admin REST** | P1 seeds config at startup; mutable CRUD is P3. |
| MQTT topic **wildcards** (`machine/+/state`) | Broker uses exact-match topics (max 5). `machine_id` comes from the **payload**; topics are **by data type** (`machine/state`, …), one shared `machine/state` topic for all machines. |
| Multi-shift overlap, DST/shift-rollover rules beyond simple break deduction | Deferred; P1 deducts breaks that overlap the window only. |
| Auth/rate-limiting on new REST endpoints beyond what `audit` already does | Reuse existing HTTP conventions. |

---

## User Stories

### P1: Shift & break configuration store ⭐ MVP

**User Story**: As an operations engineer, I want shifts and their scheduled breaks registered in the
broker so that planned production time can be computed (shift duration − breaks).

**Why P1**: Planned production time is the denominator of Availability — nothing computes without it.

**Acceptance Criteria**:
1. WHEN the broker starts THEN it SHALL apply migration `002` creating `shifts` and `shift_breaks` tables (idempotent, tracked in `schema_migrations`, pure-Go SQLite, `CGO_ENABLED=0`).
2. WHEN the broker starts with shift/break config present THEN it SHALL seed/upsert the configured shifts and breaks into SQLite.
3. WHEN a shift is queried for a machine + time window THEN the store SHALL return the shift(s) and break intervals overlapping that window.
4. WHEN no shift is configured for a machine THEN the system SHALL fall back to treating planned production time as the wall-clock window (no break deduction) and flag the result as `shift_config_missing`.

**Independent Test**: Seed one shift (08:00–16:00, breaks 10:00–10:15 & 12:00–12:30); query `[08:00,16:00]` → returns 480 min span and 45 min of breaks.

---

### P1: Machine state ingestion → durable intervals ⭐ MVP

**User Story**: As the system, I want `state_change` events turned into closed time intervals per
machine so that downtime over any window can be measured.

**Why P1**: Stop events are the only telemetry input to Availability; intervals are what both paths read.

**Acceptance Criteria**:
1. WHEN a message arrives on the configured state topic with a JSON `state_change` payload THEN the worker SHALL decode it (`machine_id`, `state`, `previous_state`, `timestamp`) per research §6.2.
2. WHEN the worker receives a message that is **not** a `state_change` (any other topic/type) THEN it SHALL ignore it without error (fan-out broadcasts every message to every worker).
3. WHEN a valid transition arrives for a machine that already has an open interval THEN the worker SHALL **close** the previous interval at the new event's timestamp and **open** a new interval for the new state.
4. WHEN a transition repeats the current state (no-op) or arrives with a non-increasing timestamp THEN the worker SHALL handle it idempotently (ignore the no-op; reject/clip the out-of-order event) without corrupting interval continuity.
5. WHEN a `state_change` payload is missing `machine_id`/`state`/`timestamp` or has an unknown state THEN the worker SHALL log and skip it, never panic or block the fan-out.
6. WHEN an interval is closed THEN it SHALL be persisted (machine_id, state, started_at, ended_at, is_downtime, is_planned_stop) for historical queries.

**Independent Test**: Publish `running`@08:00 then `stopped`@09:00 then `running`@09:10 → two closed intervals (`running` 60 min, `stopped` 10 min) persisted with `is_downtime` set correctly.

---

### P1: Pure Availability calculation core ⭐ MVP

**User Story**: As a developer, I want one infrastructure-free function that computes Availability so
that the live worker, the REST use case, and any future sink all share identical math.

**Why P1**: Single source of truth for the formula; satisfies CLAUDE.md (domain imports zero infra).

**Acceptance Criteria**:
1. WHEN given planned production time and stop time THEN the service SHALL return `Availability = (PlannedProductionTime − StopTime) / PlannedProductionTime`, clamped to `[0,1]` (research §3.1.1, §6.3).
2. WHEN classifying a state THEN the domain SHALL expose `IsDowntime()` (`stopped|setup|maintenance`) and `IsPlannedStop()` (`setup|maintenance`) exactly per research §6.1; `off` = outside shift, `idle` = not downtime.
3. WHEN `PlannedProductionTime == 0` THEN the service SHALL return `0` (or an explicit "undefined/no data") and never divide by zero.
4. WHEN the calculation runs THEN it SHALL import **no** `database/sql`, no concrete logger, no http — pure `time`/arithmetic only.

**Independent Test**: Reproduce research §7 exactly: planned 420 min, stop 47 min → Availability = 88.81 %.

---

### P1: Live in-memory accumulator + `AvailabilitySink` output port ⭐ MVP

**User Story**: As an operator, I want Availability recalculated continuously in memory as events stream
in, with the result pushed out every second through a pluggable sink, so that dashboards (websocket),
storage, and re-publishing can be added later without changing the calculation.

**Why P1**: This is the centerpiece — `payload → persist → live calc each ~1s → emitted via sink`.
The MVP must leave **one update interface ready** so future sinks attach with zero core changes.

**Acceptance Criteria**:
1. WHEN a state interval opens/closes for a machine THEN the in-memory accumulator for that machine SHALL update its running planned/run/downtime totals.
2. WHEN ~1 s elapses (ticker) THEN the accumulator SHALL produce a current `AvailabilitySnapshot{machine_id, window, availability, planned_time, run_time, downtime, planned_stop, unplanned_stop, current_state, computed_at, flags}` and pass it to the `AvailabilitySink`.
3. WHEN the snapshot is emitted THEN it SHALL go through the single output port `AvailabilitySink interface { Update(ctx, AvailabilitySnapshot) error }` (the "one ready interface"); the core SHALL NOT know which concrete sink(s) receive it.
4. WHEN multiple sinks are registered THEN a composite/fan-out sink SHALL deliver to all of them, isolating a failing/slow sink from the others (non-blocking, error-logged) — explicitly so a websocket sink can be one of them.
5. WHEN no concrete sink is configured THEN a default sink (in-memory "latest snapshot" holder + structured log) SHALL be wired so the path is observable end-to-end.
6. WHEN the accumulator is built THEN `AvailabilitySnapshot` SHALL be JSON-serializable and self-contained (websocket-ready), and the tick interval SHALL be configurable (default 1 s).

**Independent Test**: Feed `running`@T then `stopped`@T+30s into the accumulator with a 1 s tick and a fake sink; assert the sink receives monotonically-updating snapshots and availability drops as downtime accrues.

---

### P1: Websocket live sink ⭐ MVP

**User Story**: As a dashboard, I want to subscribe over websocket to a machine's Availability and
receive each ~1 s snapshot as JSON, so an operator sees Availability update in real time.

**Why P1**: User pulled this into the MVP ("já vamos fazer o websocket"). It is the first concrete
proof that the `AvailabilitySink` port works for a streaming consumer.

**Acceptance Criteria**:
1. WHEN a client connects to `GET /availability/{machine}/ws` THEN the server SHALL upgrade to websocket and subscribe the client to that machine's snapshots.
2. WHEN the live engine emits a snapshot for a machine THEN the websocket sink SHALL push it as JSON to every client subscribed to that machine.
3. WHEN a websocket client is slow or disconnects THEN it SHALL be dropped without blocking the engine, other clients, or other sinks (per AVAIL-13 fan-out isolation).
4. WHEN the websocket sink is registered THEN it SHALL be one entry in the composite sink alongside the default log sink — no change to the engine/core.

**Independent Test**: Connect a test ws client to `/availability/CNC-01/ws`, feed a transition into the engine, assert the client receives a JSON snapshot frame; drop the client and assert the engine keeps ticking.

---

### P1: On-demand Availability REST query (`[from,to]`) ⭐ MVP

**User Story**: As an analyst, I want to GET the Availability of a machine for an arbitrary time window
so I can audit any shift/period after the fact.

**Why P1**: The chosen on-demand semantics (arbitrary `[from,to]`) and a concrete consumer of the core.

**Acceptance Criteria**:
1. WHEN `GET /availability/{machine}?from=<rfc3339>&to=<rfc3339>` is called THEN the use case SHALL read state intervals overlapping `[from,to]` + the shift/breaks for that window, call the pure core, and return JSON.
2. WHEN the response is built THEN it SHALL include `availability`, `planned_production_time`, `run_time`, `downtime` (split planned/unplanned), `interval_count`, and data-coverage flags (`shift_config_missing`, `open_interval_clipped`, `no_data`).
3. WHEN `from`/`to` are missing, unparseable, or `from >= to` THEN it SHALL return `400` with a clear error (reuse audit HTTP error conventions).
4. WHEN the window has no intervals THEN it SHALL return `200` with `availability: null` (or `0`) and `no_data: true` rather than erroring.
5. WHEN an interval is still open at `to` (machine never left the state) THEN it SHALL be **clipped** to `to` and flagged `open_interval_clipped`.

**Independent Test**: With the §7 dataset persisted, `GET /availability/CNC-01?from=08:00&to=16:00` returns `availability ≈ 0.8881`.

---

### P2: Additional sinks — DB snapshot & MQTT re-publish

**User Story**: As a platform owner, I want the live snapshots persisted and/or re-published to MQTT,
so the data reaches storage and external systems — added as more sinks behind the same port.

**Why P2**: The P1 port + websocket prove the model; these are additive sinks (zero core change).

**Acceptance Criteria**:
1. WHEN the DB-snapshot sink is enabled THEN each emitted snapshot SHALL be persisted (throttled) to an `availability_snapshots` table (migration `003`), queryable via REST.
2. WHEN the MQTT re-publish sink is enabled THEN snapshots SHALL be published to a configured output topic.
3. WHEN any sink is slow/failing THEN the others and the core SHALL be unaffected (per AVAIL-13 fan-out isolation).

**Independent Test**: Enable the DB sink; after 5 s of live ticks, `availability_snapshots` has rows.

---

### P3: Shift/break CRUD & live snapshot endpoint

**User Story**: As an operations engineer, I want to manage shifts/breaks at runtime and read the
current live snapshot via REST.

**Why P3**: Convenience; MVP seeds config at startup and exposes only the on-demand query.

**Acceptance Criteria**:
1. WHEN `GET /availability/{machine}/live` is called THEN it SHALL return the latest in-memory snapshot from the default sink.
2. WHEN shift/break CRUD endpoints are called THEN they SHALL create/update/delete shift config in SQLite with validation.

---

## Edge Cases

- WHEN the broker has all 5 topic slots used and a `machine/state` topic must be added THEN configuration SHALL require freeing a slot (max 5 exact-match topics) — documented, surfaced as a config error if absent.
- WHEN two machines publish to the same `machine/state` topic THEN intervals SHALL be keyed by payload `machine_id`, not by topic/client.
- WHEN the window `[from,to]` falls entirely inside a scheduled break (planned time = 0) THEN return `no_data`/`availability: null`, no divide-by-zero.
- WHEN downtime intervals partially overlap the window THEN only the overlapping portion SHALL count toward stop time; same clipping for planned time vs. breaks.
- WHEN `state_change` timestamps go backwards or duplicate THEN the worker SHALL preserve interval monotonicity (reject/clip), never produce negative durations.
- WHEN a machine is `off` (outside shift) within the window THEN that time SHALL be excluded from planned production time, not counted as downtime.
- WHEN a worker panics on a malformed payload THEN the existing fan-out recover SHALL keep the pipeline alive (defense in depth on top of AVAIL graceful skip).
- WHEN the process restarts THEN open in-memory intervals SHALL be rebuilt from the last persisted interval per machine (no double counting).

---

## Requirement Traceability

| Requirement ID | Story | Phase | Status |
| --- | --- | --- | --- |
| AVAIL-01 | P1: Shift config | Design | Pending |
| AVAIL-02 | P1: Shift config | Design | Pending |
| AVAIL-03 | P1: Shift config | Design | Pending |
| AVAIL-04 | P1: State ingestion | Design | Pending |
| AVAIL-05 | P1: State ingestion | Design | Pending |
| AVAIL-06 | P1: State ingestion | Design | Pending |
| AVAIL-07 | P1: State ingestion | Design | Pending |
| AVAIL-08 | P1: Calculation core | Design | Pending |
| AVAIL-09 | P1: Calculation core | Design | Pending |
| AVAIL-10 | P1: Calculation core | Design | Pending |
| AVAIL-11 | P1: Live accumulator + sink | Design | Pending |
| AVAIL-12 | P1: Live accumulator + sink | Design | Pending |
| AVAIL-13 | P1: Live accumulator + sink (port) | Design | Pending |
| AVAIL-14 | P1: Live accumulator + sink (default sink) | Design | Pending |
| AVAIL-15 | P1: Live accumulator + sink (ws-ready) | Design | Pending |
| AVAIL-16 | P1: On-demand REST | Design | Pending |
| AVAIL-17 | P1: On-demand REST | Design | Pending |
| AVAIL-18 | P1: On-demand REST | Design | Pending |
| AVAIL-19 | P1: On-demand REST | Design | Pending |
| AVAIL-21 | P1: Websocket live sink | Design | Pending |
| AVAIL-25 | P1: Websocket endpoint + connection hub | Design | Pending |
| AVAIL-20 | P2: DB snapshot sink | - | Pending |
| AVAIL-22 | P2: MQTT re-publish sink | - | Pending |
| AVAIL-23 | P3: Shift/break CRUD | - | Pending |
| AVAIL-24 | P3: Live snapshot endpoint (`/live`) | - | Pending |

**ID format:** `AVAIL-NN`
**Status values:** Pending → In Design → In Tasks → Implementing → Verified
**Coverage:** 25 total, 0 mapped to tasks (Tasks phase pending) — **P1 = AVAIL-01..19, 21, 25**.

---

## Success Criteria

- [ ] Research §7 example reproduces exactly through the pure core (Availability = 88.81 %) — unit test.
- [ ] `state_change` stream produces correct persisted intervals (table-driven worker test, race-clean).
- [ ] Live accumulator emits ~1 Hz snapshots through `AvailabilitySink`; swapping the sink requires **zero** changes to the core/accumulator (proven by a fake sink in tests).
- [ ] `GET /availability/{machine}?from&to` returns correct Availability + coverage flags for seeded data.
- [ ] `go build ./...`, `go vet ./...`, `golangci-lint run ./...`, `go test -race ./...` all green (the `/verify-go` gate).
- [ ] Zero new cgo deps; SQLite stays `modernc.org/sqlite`; static build still works.
