# F2: Session Management — Tasks

**Spec**: `.specs/features/f2-session-management/spec.md`
**Status**: Done

---

## Execution Plan

### Phase 1: Domain (Sequential)

Entidades e value objects do domínio de sessão.

```
T1 → T2
```

### Phase 2: Core Components (Parallel OK)

Auth e Connection Manager são independentes.

```
     ┌→ T3 [P] ─┐
T2 ──┤           ├──→ T5
     └→ T4 [P] ─┘
```

### Phase 3: Integration (Sequential)

Server TCP e handler que integram tudo.

```
T5 → T6
```

---

## Task Breakdown

### T1: Entidade Client e Erros de Domínio

**What**: Criar a entidade Client com metadados de conexão e erros de domínio do módulo session.
**Where**: `internal/session/domain/client.go`, `internal/session/domain/errors.go`, `internal/session/domain/client_test.go`
**Depends on**: F1 completa (usa `protocol` para tipos)
**Reuses**: Nenhum
**Requirement**: SESS-01

**Done when**:

- [ ] `Client` struct com ID, Conn (net.Conn), KeepAlive, CreatedAt, LastSeen
- [ ] `NewClient(id string, conn net.Conn, keepAlive uint16) *Client`
- [ ] `ResetDeadline()` atualiza deadline da conn para `now + 1.5 * keepAlive`
- [ ] `Write(data []byte) error` escreve na conn
- [ ] `Close() error` fecha a conn
- [ ] Erros: `ErrMaxClientsReached`, `ErrClientAlreadyExists`, `ErrAuthFailed`, `ErrConnectionTimeout`
- [ ] Testes com `net.Pipe()`: ResetDeadline, Write, Close
- [ ] Gate check passes: `go test -race ./internal/session/...`
- [ ] Test count: ≥4 tests pass

**Tests**: unit
**Gate**: quick

---

### T2: Topic Registry (Value Object)

**What**: Criar o TopicRegistry que valida e armazena os tópicos permitidos pelo broker.
**Where**: `internal/session/domain/topic_registry.go`, `internal/session/domain/topic_registry_test.go`
**Depends on**: T1
**Reuses**: Nenhum
**Requirement**: SESS-01 (parte do domínio)

**Done when**:

- [ ] `TopicRegistry` struct com mapa de tópicos permitidos
- [ ] `NewTopicRegistry(topics []string) (*TopicRegistry, error)` com validação (1-5 tópicos, sem vazios)
- [ ] `IsAllowed(topic string) bool`
- [ ] `Topics() []string`
- [ ] Erro se 0 tópicos, erro se >5 tópicos, erro se tópico vazio
- [ ] Testes: criação válida, 0 tópicos, 6 tópicos, tópico vazio, IsAllowed true/false, Topics retorna todos
- [ ] Gate check passes: `go test -race ./internal/session/...`
- [ ] Test count: ≥6 tests pass (novos)

**Tests**: unit
**Gate**: quick

---

### T3: Authenticator [P]

**What**: Implementar autenticação por username/password com credenciais de env vars.
**Where**: `internal/session/auth.go`, `internal/session/auth_test.go`
**Depends on**: T2
**Reuses**: Nenhum
**Requirement**: SESS-02

**Done when**:

- [ ] Interface `Authenticator` com método `Authenticate(username, password string) bool`
- [ ] `EnvAuthenticator` struct que implementa `Authenticator` com credenciais de config
- [ ] `NewEnvAuthenticator(username, password string) *EnvAuthenticator`
- [ ] Retorna true apenas se username E password coincidem
- [ ] Retorna false se username vazio ou password vazio
- [ ] Testes: credenciais corretas, username errado, password errado, ambos errados, vazios
- [ ] Gate check passes: `go test -race ./internal/session/...`
- [ ] Test count: ≥5 tests pass (novos)

**Tests**: unit
**Gate**: quick

---

### T4: Connection Manager [P]

**What**: Implementar o gerenciador de conexões com limite de clients e thread-safety.
**Where**: `internal/session/connection_manager.go`, `internal/session/connection_manager_test.go`
**Depends on**: T2
**Reuses**: `domain.Client` de T1
**Requirement**: SESS-03

**Done when**:

- [ ] `ConnectionManager` struct com mutex, mapa de clients e maxClients
- [ ] `NewConnectionManager(maxClients int) *ConnectionManager`
- [ ] `CanAccept() bool` — thread-safe
- [ ] `Add(client *domain.Client) error` — retorna `ErrMaxClientsReached` se cheio
- [ ] `Remove(clientID string)` — libera slot
- [ ] `Get(clientID string) (*domain.Client, bool)`
- [ ] `Count() int`
- [ ] `CloseAll()` — fecha todas as conexões (para shutdown)
- [ ] Testes: add até limite, add além do limite, remove libera slot, CanAccept, concorrência (10 goroutines tentando Add)
- [ ] Gate check passes: `go test -race ./internal/session/...`
- [ ] Test count: ≥7 tests pass (novos)

**Tests**: unit
**Gate**: quick

---

### T5: TCP Server e Accept Loop

**What**: Implementar o TCP listener com accept loop, verificação de limite e spawn de goroutines por conexão.
**Where**: `internal/session/server.go`, `internal/session/interfaces.go`
**Depends on**: T3, T4
**Reuses**: `ConnectionManager`, `Authenticator`, `TopicRegistry`
**Requirement**: SESS-04

**Done when**:

- [ ] `Server` struct com config, listener, connMgr, auth, topics, msgChan, logger
- [ ] `NewServer(...)` constructor com todas as dependências injetadas
- [ ] `ListenAndServe(ctx context.Context) error` — accept loop com context cancellation
- [ ] `Close() error` — fecha o listener
- [ ] `interfaces.go` com `MessageSink` interface (canal de saída para ingestion)
- [ ] Verifica `CanAccept()` antes de criar goroutine
- [ ] Fecha conexão imediatamente se limite atingido
- [ ] Loga erros de accept sem parar o loop
- [ ] Gate check passes: `go build ./internal/session/...`

**Tests**: integration (testado em T6 junto com handler)
**Gate**: build

---

### T6: Packet Handler (Orquestração Completa)

**What**: Implementar o handler que processa o fluxo completo: CONNECT → auth → read loop (PUBLISH, PINGREQ, SUBSCRIBE, DISCONNECT) com keep-alive.
**Where**: `internal/session/handler.go`, `internal/session/handler_test.go`
**Depends on**: T5
**Reuses**: `protocol.ReadPacket`, `protocol.DecodeConnect`, `protocol.DecodePublish`, `protocol.DecodeSubscribe`, todos os encoders
**Requirement**: SESS-05

**Done when**:

- [ ] `handleConnection(ctx, conn)` — fluxo completo de uma conexão
- [ ] Espera CONNECT como primeiro pacote (com timeout de 5s)
- [ ] Rejeita se primeiro pacote não é CONNECT
- [ ] Autentica via `Authenticator`
- [ ] Registra client via `ConnectionManager`
- [ ] Envia CONNACK com return code apropriado
- [ ] `readLoop(ctx, client, decoder)` — loop de leitura de pacotes
- [ ] Trata PUBLISH: valida tópico, envia para msgChan, responde PUBACK se QoS 1
- [ ] Trata PINGREQ: responde PINGRESP
- [ ] Trata SUBSCRIBE: responde SUBACK
- [ ] Trata DISCONNECT: remove client e retorna
- [ ] Keep-alive: `ResetDeadline()` a cada pacote recebido
- [ ] Remove client do ConnectionManager em qualquer saída (defer)
- [ ] Testes de integração com `net.Pipe()`:
  - Connect válido → CONNACK accepted
  - Connect com auth errada → CONNACK bad auth + disconnect
  - Publish QoS 0 → mensagem no canal
  - Publish QoS 1 → PUBACK + mensagem no canal
  - Publish em tópico inválido → ignorado
  - PINGREQ → PINGRESP
  - DISCONNECT → client removido
  - Primeiro pacote não é CONNECT → disconnect
- [ ] Gate check passes: `go test -race ./internal/session/...`
- [ ] Test count: ≥8 tests pass (novos)

**Tests**: integration
**Gate**: full

**Commit**: `feat(session): complete session management with auth and connection control`

---

## Parallel Execution Map

```
Phase 1 (Sequential):
  T1 ──→ T2

Phase 2 (Parallel):
  T2 complete, then:
    ├── T3 [P]  Authenticator
    └── T4 [P]  Connection Manager

Phase 3 (Sequential):
  T3, T4 complete, then:
    T5 ──→ T6
```

---

## Task Granularity Check

| Task | Scope | Status |
|------|-------|--------|
| T1: Client entity | 1 struct + methods + errors | ✅ Granular |
| T2: Topic Registry | 1 value object + tests | ✅ Granular |
| T3: Authenticator | 1 interface + 1 impl + tests | ✅ Granular |
| T4: Connection Manager | 1 struct + methods + tests | ✅ Granular |
| T5: TCP Server | 1 struct + accept loop | ✅ Granular |
| T6: Packet Handler | 1 handler + read loop + tests | ⚠️ OK — coeso, tudo no mesmo fluxo |

---

## Diagram-Definition Cross-Check

| Task | Depends On (body) | Diagram Shows | Status |
|------|-------------------|---------------|--------|
| T1 | F1 completa | Start | ✅ Match |
| T2 | T1 | T1 → T2 | ✅ Match |
| T3 | T2 | T2 → T3 | ✅ Match |
| T4 | T2 | T2 → T4 | ✅ Match |
| T5 | T3, T4 | T3,T4 → T5 | ✅ Match |
| T6 | T5 | T5 → T6 | ✅ Match |

---

## Test Co-location Validation

| Task | Code Layer | Matrix Requires | Task Says | Status |
|------|-----------|----------------|-----------|--------|
| T1 | Session domain (Client) | unit | unit | ✅ OK |
| T2 | Session domain (TopicRegistry) | unit | unit | ✅ OK |
| T3 | Auth | unit | unit | ✅ OK |
| T4 | Connection Manager | unit | unit | ✅ OK |
| T5 | TCP Server | integration | integration (in T6) | ✅ OK (merged forward) |
| T6 | TCP Server + Handler | integration | integration | ✅ OK |
