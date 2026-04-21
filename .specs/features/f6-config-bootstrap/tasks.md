# F6: Configuration & Bootstrap — Tasks

**Spec**: `.specs/features/f6-config-bootstrap/spec.md`
**Status**: Draft

---

## Execution Plan

### Phase 1: Foundation (Parallel OK)

```
┌→ T1 [P] ─┐
┤           ├──→ T3
└→ T2 [P] ─┘
```

### Phase 2: Integration (Sequential)

```
T3 → T4 → T5
```

---

## Task Breakdown

### T1: Common Logger Interface [P]

**What**: Criar a interface de logging compartilhada que todos os módulos usam.
**Where**: `internal/common/logger.go`
**Depends on**: None
**Reuses**: Nenhum
**Requirement**: BOOT-03

**Done when**:

- [ ] Interface `Logger` com métodos: `Info(msg string, args ...any)`, `Error(msg string, args ...any)`, `Warn(msg string, args ...any)`, `Debug(msg string, args ...any)`
- [ ] `NewSlogLogger() Logger` — adapter que wrapa `*slog.Logger`
- [ ] `NewNopLogger() Logger` — logger que descarta tudo (para testes)
- [ ] Gate check passes: `go build ./internal/common/...`

**Tests**: none (interface + adapters triviais)
**Gate**: build

---

### T2: Config Loader [P]

**What**: Implementar o carregamento e validação de configuração via env vars.
**Where**: `internal/config/config.go`, `internal/config/config_test.go`
**Depends on**: None
**Reuses**: Nenhum
**Requirement**: BOOT-01

**Done when**:

- [ ] `Config` struct com: Host, Port, Username, Password, Topics ([]string), MaxClients, QueueBufferSize, DBPath, Timezone
- [ ] `Load() (*Config, error)` — lê de env vars com defaults
- [ ] `Address() string` — retorna "host:port"
- [ ] Defaults: Port=1883, MaxClients=5, QueueBufferSize=10000, DBPath="./data/broker.db", Timezone="UTC"
- [ ] Validação: Topics não vazio, MaxClients 1-5, Username não vazio, Password não vazio
- [ ] Trim de espaços em todos os campos string
- [ ] Topics: split por vírgula, ignora vazios
- [ ] Testes:
  - Config válida com todas as env vars
  - Defaults aplicados quando env vars ausentes
  - Erro quando Topics vazio
  - Erro quando MaxClients > 5
  - Erro quando Username vazio
  - Erro quando Password vazio
  - Trim de espaços
  - Topics com vírgulas extras
- [ ] Gate check passes: `go test -race ./internal/config/...`
- [ ] Test count: ≥8 tests pass

**Tests**: unit
**Gate**: quick

---

### T3: Bootstrap (main.go)

**What**: Implementar o wiring de todos os módulos no main.go com graceful shutdown.
**Where**: `cmd/main.go`
**Depends on**: T1, T2, F1-F5 completas
**Reuses**: Todos os módulos
**Requirement**: BOOT-02

**Done when**:

- [ ] Carrega config via `config.Load()`
- [ ] Cria logger via `common.NewSlogLogger()`
- [ ] Cria context com cancel
- [ ] Configura signal handling (SIGINT, SIGTERM)
- [ ] Cria SQLite store e executa migrate
- [ ] Cria channels: ingestChan e dispatchChan
- [ ] Cria workers (LoggerWorker)
- [ ] Cria e inicia Pipeline
- [ ] Cria e inicia Dispatcher
- [ ] Cria e inicia Server
- [ ] Aguarda signal → cancel context
- [ ] Shutdown: server.Close() → aguarda goroutines → store.Close()
- [ ] Loga configuração no startup (endereço, tópicos, max clients)
- [ ] Loga confirmação de shutdown
- [ ] Cria diretório do DB_PATH se não existe
- [ ] Gate check passes: `go build ./cmd/...`

**Tests**: build (integração manual)
**Gate**: build

---

### T4: Dockerfile e Docker Compose

**What**: Atualizar Dockerfile para compilar corretamente com modernc/sqlite e atualizar .env e docker-compose.
**Where**: `Dockerfile`, `.env`, `docker-compose.yml`
**Depends on**: T3
**Reuses**: Nenhum
**Requirement**: BOOT-04

**Done when**:

- [ ] Dockerfile: multi-stage build com Go 1.22, CGO_ENABLED=0 (modernc é pure Go)
- [ ] Dockerfile: copia binário para scratch/alpine, expõe porta 1883
- [ ] Dockerfile: cria diretório /data para SQLite
- [ ] `.env` atualizado com todas as variáveis documentadas
- [ ] `docker-compose.yml` atualizado com porta 1883, volume para /data, env_file
- [ ] Gate check passes: `go build ./cmd/...`

**Tests**: none (infra)
**Gate**: build

**Commit**: `feat(bootstrap): complete configuration, wiring and Docker setup`

---

### T5: Testes End-to-End

**What**: Implementar suite de testes e2e que validam o fluxo completo do broker com TCP real, SQLite real (`:memory:`), e todos os módulos wired. Cada teste sobe o broker programaticamente (Server + Pipeline + Dispatcher + SQLite + migrations) e conecta clients TCP reais.
**Where**: `tests/e2e/broker_e2e_test.go`
**Depends on**: T3 (bootstrap completo, todos os módulos wired)
**Reuses**: Todos os módulos, helpers de construção de pacotes MQTT do `session/handler_test.go`
**Requirement**: BOOT-05

**Done when**:

- [ ] Helper `setupBroker(t)` que:
  - Cria SQLite `:memory:` + roda migrations
  - Cria channels (ingestChan, dispatchChan)
  - Cria Pipeline com topics configurados
  - Cria Dispatcher com mock worker que coleta mensagens
  - Cria Server com TCP real em porta 0 (auto-assign)
  - Retorna struct com server addr, store, mock worker, cancel func
- [ ] Helper `connectClient(t, addr, clientID, user, pass)` que:
  - Abre conexão TCP real
  - Envia CONNECT packet
  - Lê e valida CONNACK
  - Retorna `net.Conn`
- [ ] Helper `publishMessage(t, conn, topic, payload, qos, packetID)` que:
  - Envia PUBLISH packet via TCP
  - Se QoS 1, lê e valida PUBACK
- [ ] Teste: **Happy path completo**
  - 1 client conecta, publica 5 mensagens em 2 tópicos diferentes
  - Verifica via `store.GetByTopic()` que todas as 5 mensagens estão no SQLite
  - Verifica que o mock worker recebeu todas as 5 mensagens
  - Verifica campos: clientID, topic, payload corretos
- [ ] Teste: **Múltiplos clients simultâneos**
  - 3 clients conectam em paralelo
  - Cada um publica 10 mensagens (total 30)
  - Verifica que todas as 30 estão no banco
  - Verifica que o mock worker recebeu todas as 30
  - Verifica que a ordem por tópico é mantida (mensagens do mesmo tópico em ordem de envio)
- [ ] Teste: **6º client rejeitado**
  - 5 clients conectam com sucesso
  - 6º client tenta conectar → conexão é fechada pelo server (read retorna EOF/error)
  - Os 5 originais publicam 1 mensagem cada → todas as 5 aparecem no banco
- [ ] Teste: **Auth falha não polui pipeline**
  - 1 client conecta com credenciais erradas → recebe CONNACK com código de erro
  - Verifica que o banco está vazio
  - Verifica que o mock worker não recebeu nada
  - 1 client conecta com credenciais corretas e publica → mensagem aparece no banco
- [ ] Teste: **Graceful shutdown sob carga**
  - 1 client conecta e publica 5 mensagens
  - Espera mensagens chegarem no dispatch channel
  - Cancela context
  - Verifica que as mensagens já processadas estão no banco
- [ ] Teste: **Tópico não permitido não persiste**
  - 1 client conecta e publica em tópico permitido e em tópico não permitido
  - Verifica que apenas a mensagem do tópico permitido está no banco
  - Verifica que o client continua conectado (publica outra mensagem no tópico permitido com sucesso)
- [ ] Gate check passes: `go test -race ./tests/e2e/...`
- [ ] Test count: ≥6 tests pass

**Tests**: e2e (TCP real + SQLite real `:memory:`)
**Gate**: full

**Commit**: `test(e2e): end-to-end broker tests with real TCP and SQLite`

---

## Parallel Execution Map

```
Phase 1 (Parallel):
  ├── T1 [P]  Logger Interface
  └── T2 [P]  Config Loader

Phase 2 (Sequential):
  T1, T2 complete, then:
    T3 ──→ T4 ──→ T5
```

---

## Task Granularity Check

| Task | Scope | Status |
|------|-------|--------|
| T1: Logger Interface | 1 interface + 2 adapters | ✅ Granular |
| T2: Config Loader | 1 struct + loader + tests | ✅ Granular |
| T3: Bootstrap | 1 file (main.go) | ⚠️ OK — wiring only, zero business logic |
| T4: Docker | 3 files (infra) | ✅ Granular |
| T5: Testes E2E | 1 test file + helpers | ✅ Granular |

---

## Diagram-Definition Cross-Check

| Task | Depends On (body) | Diagram Shows | Status |
|------|-------------------|---------------|--------|
| T1 | None | Start (parallel) | ✅ Match |
| T2 | None | Start (parallel) | ✅ Match |
| T3 | T1, T2, F1-F5 | T1,T2 → T3 | ✅ Match |
| T4 | T3 | T3 → T4 | ✅ Match |
| T5 | T3 | T4 → T5 | ✅ Match |

---

## Test Co-location Validation

| Task | Code Layer | Matrix Requires | Task Says | Status |
|------|-----------|----------------|-----------|--------|
| T1 | Common (logger) | none | none | ✅ OK |
| T2 | Config | unit | unit | ✅ OK |
| T3 | Bootstrap (main.go) | build | build | ✅ OK |
| T4 | Docker (infra) | none | none | ✅ OK |
| T5 | E2E (cross-module) | e2e | e2e | ✅ OK |
