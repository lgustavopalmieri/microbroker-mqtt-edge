# F5: Worker Dispatch — Specification

## Problem Statement

Após cada mensagem ser persistida no SQLite, ela precisa ser distribuída para workers extensíveis que farão o forwarding (Kafka, S3, REST) ou cálculos (OEE, contagem). O sistema de dispatch deve ser fan-out (cada mensagem vai para todos os workers) e extensível via interface.

## Goals

- [ ] Interface Worker clara e simples para implementação de plugins
- [ ] Dispatcher fan-out que distribui cada mensagem para todos os workers registrados
- [ ] Logger Worker como implementação de referência e ferramenta de debug
- [ ] Graceful shutdown de todos os workers

## Out of Scope

| Feature | Reason |
|---------|--------|
| Workers concretos (Kafka, S3, REST) | v2 — apenas interface e logger |
| Retry/circuit breaker por worker | v2 |
| Worker health check | v2 |
| Worker configuration via env | v2 — workers são registrados no código |

---

## User Stories

### P1: Interface Worker ⭐ MVP

**User Story**: Como desenvolvedor de plugins, preciso de uma interface clara para implementar novos workers de forwarding.

**Why P1**: Contrato que todos os workers devem seguir.

**Acceptance Criteria**:

1. WHEN implemento a interface Worker THEN devo ter métodos Name(), Process(ctx, msg) e Close()
2. WHEN Process retorna erro THEN o dispatcher deve logar mas não parar
3. WHEN Close é chamado THEN o worker deve liberar seus recursos

**Independent Test**: Compilação de mock worker.

---

### P1: Fan-Out Dispatcher ⭐ MVP

**User Story**: Como broker, preciso distribuir cada mensagem persistida para todos os workers registrados simultaneamente.

**Why P1**: Core do sistema de extensibilidade.

**Acceptance Criteria**:

1. WHEN uma mensagem chega no canal de input THEN deve ser enviada para todos os workers
2. WHEN um worker falha THEN os outros devem continuar processando
3. WHEN o context é cancelado THEN o dispatcher deve parar
4. WHEN não há workers registrados THEN mensagens devem ser consumidas sem erro
5. WHEN o dispatcher é iniciado THEN deve processar mensagens até o canal fechar ou context cancelar
6. WHEN todos os workers processam uma mensagem THEN o dispatcher deve esperar todos terminarem antes da próxima

**Independent Test**: Testes com mock workers verificando fan-out.

---

### P1: Logger Worker ⭐ MVP

**User Story**: Como desenvolvedor/operador, preciso de um worker que loga cada mensagem recebida para debug e validação do pipeline.

**Why P1**: Implementação de referência e ferramenta essencial de debug.

**Acceptance Criteria**:

1. WHEN o logger worker recebe uma mensagem THEN deve logar topic, clientID e tamanho do payload
2. WHEN Name() é chamado THEN deve retornar "logger"
3. WHEN Close() é chamado THEN deve ser no-op (sem recursos para liberar)

**Independent Test**: Teste verificando que log é chamado.

---

## Edge Cases

- WHEN um worker entra em panic THEN system SHALL recuperar e logar, sem derrubar o dispatcher
- WHEN o canal de input é fechado THEN system SHALL parar o dispatcher graciosamente
- WHEN Process demora muito THEN system SHALL esperar (sem timeout por worker em v1)

---

## Requirement Traceability

| Requirement ID | Story | Phase | Status |
|---------------|-------|-------|--------|
| DISP-01 | P1: Interface Worker | Tasks | Pending |
| DISP-02 | P1: Fan-Out Dispatcher | Tasks | Pending |
| DISP-03 | P1: Logger Worker | Tasks | Pending |

**Coverage:** 3 total, 3 mapped to tasks, 0 unmapped ✅

---

## Success Criteria

- [ ] Mensagem persistida aparece no log do Logger Worker
- [ ] Múltiplos workers recebem a mesma mensagem
- [ ] Falha de um worker não afeta os outros
- [ ] Todos os testes passam com `go test -race ./internal/modules/dispatch/...`
