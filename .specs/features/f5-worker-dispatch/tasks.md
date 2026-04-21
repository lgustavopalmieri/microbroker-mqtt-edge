# F5: Worker Dispatch — Tasks

**Spec**: `.specs/features/f5-worker-dispatch/spec.md`
**Status**: Draft

---

## Execution Plan

### Phase 1: Interface (Sequential)

```
T1
```

### Phase 2: Dispatcher + Worker (Parallel OK)

```
     ┌→ T2 [P] ─┐
T1 ──┤           ├──→ (done)
     └→ T3 [P] ─┘
```

---

## Task Breakdown

### T1: Interface Worker e Tipos

**What**: Definir a interface Worker e tipos compartilhados do módulo dispatch.
**Where**: `internal/dispatch/domain/worker.go`, `internal/dispatch/interfaces.go`
**Depends on**: F4 T1 (usa `ingestion/domain.Message`)
**Reuses**: `ingestion/domain.Message`
**Requirement**: DISP-01

**Done when**:

- [ ] Interface `Worker` com `Name() string`, `Process(ctx context.Context, msg Message) error`, `Close() error`
- [ ] Type alias ou re-export de `Message` para evitar import circular (ou usar tipo do ingestion diretamente)
- [ ] Gate check passes: `go build ./internal/dispatch/...`

**Tests**: none (interface pura)
**Gate**: build

---

### T2: Fan-Out Dispatcher [P]

**What**: Implementar o dispatcher que lê do canal de input e distribui para todos os workers registrados.
**Where**: `internal/dispatch/dispatcher.go`, `internal/dispatch/dispatcher_test.go`
**Depends on**: T1
**Reuses**: Interface `Worker` de T1
**Requirement**: DISP-02

**Done when**:

- [ ] `Dispatcher` struct com workers []Worker, input <-chan Message, logger
- [ ] `NewDispatcher(input <-chan Message, workers []Worker, logger) *Dispatcher`
- [ ] `Start(ctx context.Context)` — loop que lê do input e faz fan-out
- [ ] `fanOut(ctx, msg)` — lança goroutine por worker, espera todos com WaitGroup
- [ ] `Close() error` — chama Close() em todos os workers
- [ ] Recover de panic em worker individual (não derruba dispatcher)
- [ ] Testes com mock workers (gomock):
  - Fan-out: 3 workers recebem a mesma mensagem
  - Worker com erro: outros continuam
  - Context cancelado: dispatcher para
  - Sem workers: mensagens consumidas sem erro
  - Canal fechado: dispatcher para
  - Ordem: mensagens processadas na ordem de chegada
- [ ] Gate check passes: `go test -race ./internal/dispatch/...`
- [ ] Test count: ≥6 tests pass

**Tests**: unit
**Gate**: quick

---

### T3: Logger Worker [P]

**What**: Implementar o worker de log que serve como referência e ferramenta de debug.
**Where**: `internal/dispatch/workers/logger_worker.go`, `internal/dispatch/workers/logger_worker_test.go`
**Depends on**: T1
**Reuses**: Interface `Worker` de T1
**Requirement**: DISP-03

**Done when**:

- [ ] `LoggerWorker` struct com logger
- [ ] `NewLoggerWorker(logger) *LoggerWorker`
- [ ] `Name()` retorna "logger"
- [ ] `Process(ctx, msg)` loga: topic, clientID, payload size, timestamp
- [ ] `Close()` é no-op, retorna nil
- [ ] Testes:
  - Name retorna "logger"
  - Process loga campos corretos (verificar via buffer de log)
  - Close retorna nil
- [ ] Gate check passes: `go test -race ./internal/dispatch/workers/...`
- [ ] Test count: ≥3 tests pass

**Tests**: unit
**Gate**: quick

**Commit**: `feat(dispatch): worker dispatch system with fan-out and logger worker`

---

## Parallel Execution Map

```
Phase 1 (Sequential):
  T1

Phase 2 (Parallel):
  T1 complete, then:
    ├── T2 [P]  Dispatcher
    └── T3 [P]  Logger Worker
```

---

## Task Granularity Check

| Task | Scope | Status |
|------|-------|--------|
| T1: Worker interface | 2 files (interface + types) | ✅ Granular |
| T2: Dispatcher | 1 struct + fan-out + tests | ✅ Granular |
| T3: Logger Worker | 1 struct + tests | ✅ Granular |

---

## Diagram-Definition Cross-Check

| Task | Depends On (body) | Diagram Shows | Status |
|------|-------------------|---------------|--------|
| T1 | F4 T1 | Start | ✅ Match |
| T2 | T1 | T1 → T2 | ✅ Match |
| T3 | T1 | T1 → T3 | ✅ Match |

---

## Test Co-location Validation

| Task | Code Layer | Matrix Requires | Task Says | Status |
|------|-----------|----------------|-----------|--------|
| T1 | Worker interface | none | none | ✅ OK |
| T2 | Dispatcher | unit | unit | ✅ OK |
| T3 | Logger Worker | unit | unit | ✅ OK |
