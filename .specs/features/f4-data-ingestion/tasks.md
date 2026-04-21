# F4: Data Ingestion Pipeline — Tasks

**Spec**: `.specs/features/f4-data-ingestion/spec.md`
**Status**: Done

---

## Execution Plan

### Phase 1: Domain (Sequential)

```
T1
```

### Phase 2: Store + Queue (Parallel OK)

```
     ┌→ T2 [P] ─┐
T1 ──┤           ├──→ T4
     └→ T3 [P] ─┘
```

### Phase 3: Pipeline Integration (Sequential)

```
T4
```

---

## Task Breakdown

### T1: Entidade Message e Interfaces

**What**: Criar a entidade Message, erros de domínio e interfaces (ports) do módulo ingestion.
**Where**: `internal/ingestion/domain/message.go`, `internal/ingestion/domain/errors.go`, `internal/ingestion/interfaces.go`
**Depends on**: F1 completa, F2 completa
**Reuses**: Nenhum
**Requirement**: INGEST-01

**Done when**:

- [ ] `Message` struct com ClientID, Topic, Payload ([]byte), Timezone, Timestamp (time.Time)
- [ ] Erros: `ErrStoreFailure`, `ErrQueueFull`
- [ ] Interface `Store` com `SaveRawData(ctx, msg) error`, `Close() error`
- [ ] Interface `Store` com `Migrate(ctx) error` para criação de tabela
- [ ] Gate check passes: `go build ./internal/ingestion/...`

**Tests**: unit
**Gate**: build

---

### T2: SQLite Store [P]

**What**: Implementar o adapter SQLite que persiste mensagens na tabela raw_data com transação atômica e mutex.
**Where**: `internal/ingestion/store/sqlite.go`, `internal/ingestion/store/sqlite_test.go`
**Depends on**: T1
**Reuses**: Interface `Store` de T1
**Requirement**: INGEST-03

**Done when**:

- [ ] `SQLiteStore` struct com `*sql.DB` e `sync.Mutex`
- [ ] `NewSQLiteStore(dbPath string) (*SQLiteStore, error)` — abre DB com WAL mode, busy timeout, MaxOpenConns=1
- [ ] `Migrate(ctx) error` — cria tabela `raw_data` e índices (client, topic, timezone, timestamp, payload, created_at)
- [ ] `SaveRawData(ctx, msg) error` — lock mutex, begin tx, insert, commit
- [ ] `Close() error` — fecha conexão
- [ ] Testes de integração com `:memory:`:
  - Migrate cria tabela
  - SaveRawData insere corretamente
  - Múltiplos inserts mantêm ordem
  - Concorrência: 5 goroutines inserindo simultaneamente (mutex garante serialização)
  - Campos são recuperáveis via SELECT
- [ ] Gate check passes: `go test -race ./internal/ingestion/store/...`
- [ ] Test count: ≥6 tests pass

**Tests**: integration
**Gate**: full

---

### T3: Fila FIFO (Topic Queue) [P]

**What**: Implementar a fila FIFO por tópico usando Go channels com consumer sequencial.
**Where**: `internal/ingestion/queue.go`, `internal/ingestion/queue_test.go`
**Depends on**: T1
**Reuses**: `domain.Message` de T1
**Requirement**: INGEST-02

**Done when**:

- [ ] `Queue` struct com topic (string), messages (chan Message), done (chan struct{})
- [ ] `NewQueue(topic string, bufferSize int) *Queue`
- [ ] `Enqueue(msg Message)` — envia para channel (bloqueia se cheio)
- [ ] `StartConsumer(ctx, store Store, dispatchChan chan<- Message, logger)` — goroutine que:
  1. Lê da fila
  2. Chama `store.SaveRawData`
  3. Se sucesso, envia para `dispatchChan`
  4. Se erro, loga e continua (não perde próximas mensagens)
  5. Para quando ctx é cancelado
- [ ] Testes:
  - Ordem FIFO: enqueue A,B,C → consumer processa A,B,C
  - Consumer para quando context cancelado
  - Mensagem vai para dispatchChan após save
  - Erro no store: mensagem não vai para dispatchChan, consumer continua
  - Backpressure: buffer cheio bloqueia Enqueue
- [ ] Gate check passes: `go test -race ./internal/ingestion/...`
- [ ] Test count: ≥5 tests pass

**Tests**: unit
**Gate**: quick

---

### T4: Pipeline Orchestrator

**What**: Implementar o orquestrador que cria filas por tópico, roteia mensagens e inicia consumers.
**Where**: `internal/ingestion/pipeline.go`, `internal/ingestion/pipeline_test.go`
**Depends on**: T2, T3
**Reuses**: `Queue` de T3, `Store` interface de T1
**Requirement**: INGEST-04

**Done when**:

- [ ] `Pipeline` struct com mapa de queues, store, dispatchChan, logger
- [ ] `NewPipeline(topics []string, store Store, dispatchChan chan<- Message, bufSize int, logger) *Pipeline`
- [ ] `Start(ctx, inputChan <-chan Message)` — inicia consumers + loop de roteamento
- [ ] Roteia mensagem para fila correta por tópico
- [ ] Descarta mensagem se tópico não existe (loga warning)
- [ ] Para quando context cancelado
- [ ] Testes:
  - Mensagem roteada para fila correta
  - Mensagem com tópico desconhecido é descartada
  - Múltiplos tópicos funcionam simultaneamente
  - Graceful shutdown: context cancel para tudo
  - Mensagem persistida aparece no dispatchChan
- [ ] Gate check passes: `go test -race ./internal/ingestion/...`
- [ ] Test count: ≥5 tests pass (novos)

**Tests**: unit
**Gate**: quick

**Commit**: `feat(ingestion): complete data ingestion pipeline with FIFO queues and SQLite persistence`

---

## Parallel Execution Map

```
Phase 1 (Sequential):
  T1

Phase 2 (Parallel):
  T1 complete, then:
    ├── T2 [P]  SQLite Store
    └── T3 [P]  FIFO Queue

Phase 3 (Sequential):
  T2, T3 complete, then:
    T4  Pipeline Orchestrator
```

---

## Task Granularity Check

| Task | Scope | Status |
|------|-------|--------|
| T1: Message entity + interfaces | 3 files (entity, errors, interfaces) | ✅ Granular |
| T2: SQLite Store | 1 adapter + tests | ✅ Granular |
| T3: FIFO Queue | 1 struct + consumer + tests | ✅ Granular |
| T4: Pipeline | 1 orchestrator + tests | ✅ Granular |

---

## Diagram-Definition Cross-Check

| Task | Depends On (body) | Diagram Shows | Status |
|------|-------------------|---------------|--------|
| T1 | F1, F2 | Start | ✅ Match |
| T2 | T1 | T1 → T2 | ✅ Match |
| T3 | T1 | T1 → T3 | ✅ Match |
| T4 | T2, T3 | T2,T3 → T4 | ✅ Match |

---

## Test Co-location Validation

| Task | Code Layer | Matrix Requires | Task Says | Status |
|------|-----------|----------------|-----------|--------|
| T1 | Ingestion domain | unit | unit (build gate) | ✅ OK |
| T2 | SQLite Store | integration | integration | ✅ OK |
| T3 | Queue (FIFO) | unit | unit | ✅ OK |
| T4 | Pipeline | unit | unit | ✅ OK |
