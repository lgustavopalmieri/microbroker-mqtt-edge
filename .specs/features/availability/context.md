# Availability Context

**Gathered:** 2026-06-02
**Spec:** `.specs/features/availability/spec.md`
**Status:** Ready for design

---

## Feature Boundary

Compute the **Availability** OEE factor (`RunTime / PlannedProductionTime`) for machines, from
`state_change` events + shift/break config. Two paths over one pure calculation core: a **live
in-memory accumulator** (~1 s tick → pluggable output sink) and an **on-demand REST query** for an
arbitrary `[from,to]` window. Persist shifts/breaks (SQLite, migration 002) and state intervals.
Performance/Quality/OEE-product are explicitly **not** in this feature.

---

## Implementation Decisions

### Where the calculation lives (the central question the user raised)

- **Both live and on-demand**, sharing one **pure, infra-free** calculation core (`domain` service).
- Flow: **payload enters → persisted (existing pipeline) → fed into an in-memory per-machine calc →
  recalculated every ~1 s → result pushed out through a single output port.**
- The MVP leaves **exactly one "update" interface ready**: `AvailabilitySink.Update(ctx, snapshot)`.
  The accumulator/core do **not** know the concrete sink. Future sinks (DB, websocket, MQTT) attach
  with zero changes to the core. Tick interval configurable (default 1 s).
- Quote (user): *"a cada segundo que o cálculo for atualizado em memória teremos a opção no futuro de
  salvar em banco, publicar num websocket ... payload entra > salva > cálculo a cada segundo > sendo
  jogado num websocket, salvando e etc."*

### Output sinks / delivery (user wanted options 1+2+3)

- All three destinations are wanted, as **implementations of the same `AvailabilitySink` port**:
  1. **REST on-demand** (`GET /availability/{machine}?from&to`) — P1.
  2. **Snapshot table** (`availability_snapshots`, persisted ticks) — P2 (`AVAIL-20`).
  3. **Re-publish / websocket** — P2. **Websocket flagged "extremamente importante"** → the port and
     `AvailabilitySnapshot` are designed websocket-first (JSON-serializable, self-contained,
     fan-out sink isolates a slow websocket client). Websocket = `AVAIL-21`, MQTT republish = `AVAIL-22`.
- P1 ships the **port + a default sink** (in-memory latest-snapshot holder + structured log) so the
  end-to-end path is observable before any concrete sink exists.

### Shift & break configuration source

- **SQLite tables via migration `002`** (`shifts`, `shift_breaks`), **seeded at startup** from config.
- Mutable CRUD is deferred to P3. Follows the repo's migration convention (next file = `002_*.sql`,
  pure-Go SQLite, `CGO_ENABLED=0`; do **not** hand-write `schema_migrations`).

### Calculation window (on-demand path)

- **Arbitrary `[from,to]` on request.** Service reads intervals overlapping the window, clips partial
  intervals, deducts breaks overlapping the window. No shift-close detection in MVP.

### Domain modeling decisions (faithful to research doc)

- `IsDowntime()` = `stopped | setup | maintenance`; `IsPlannedStop()` = `setup | maintenance`
  (research §6.1). `off` = outside shift (excluded from planned time); `idle` = not downtime.
- `Availability = (PlannedProductionTime − StopTime) / PlannedProductionTime`, clamped `[0,1]`.
- Validation anchor: research §7 worked example → 88.81 %.

### Integration constraints (discovered in code, locked)

- **`machine_id` comes from the JSON payload, not the topic.** Topics are exact-match (no wildcards),
  max 5; all machines share one `machine/state` topic. Adding it may require freeing a topic slot.
- **Fan-out broadcasts every message to every worker** (`fanout.go:30`) — the availability worker must
  self-filter to `state_change` payloads and ignore everything else without error.
- Existing raw persistence is **reused**, not rebuilt; this feature adds typed interval + snapshot tables.

### Agent's Discretion

- Exact module placement under `internal/modules/` (e.g. a new `oee`/`availability` bounded context vs.
  extending `processing`) — to be decided in **Design** following the hexagonal layout (`/hexagonal-scaffold`).
- Snapshot throttling strategy for the DB sink (P2).
- Whether the live accumulator runs as a `fanout.Worker` that also owns the ticker, or a worker that
  feeds a separate ticking aggregator — Design decision (both honor the port).

---

## Specific References

- `docs/industry-4.0-metrics-research.md` is the domain source of truth — §3.1.1 (formula), §4.1.2
  (`state_change` payload), §6.1 (`MachineState`, `IsDowntime`, `IsPlannedStop`), §6.3 (`Calculate`),
  §7 (worked example = test oracle), §9 (minimum data per metric).
- Mirror the existing `audit/raw` hexagonal feature for the REST read path (domain → application
  `interface.go` ports + DTO → adapters inbound/outbound).
- Mirror `processing/workers/logger` + `fanout.Worker` for the ingestion worker.

---

## Deferred Ideas

- Performance, Quality, full OEE product, production orders, piece-count/cycle-time/reject ingestion
  (separate features — research §3.1.2–3.1.4, §3.2–3.4).
- Websocket **server/transport** stack (P1 only designs the port; sink is P2).
- Shift/break runtime CRUD + `GET /availability/{machine}/live` (P3).
- Migrator move to goose/atlas (tracked in user memory `migrations-move-to-goose-atlas`) — not part of
  this feature; keep using the custom numbered migrator unless that lands first.
