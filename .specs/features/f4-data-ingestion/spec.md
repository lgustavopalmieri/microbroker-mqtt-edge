# F4: Data Ingestion Pipeline — Specification

## Problem Statement

Após o broker receber dados via PUBLISH, precisamos garantir que cada mensagem seja persistida em SQLite de forma ordenada e confiável antes de ser encaminhada para workers. A ordem de chegada por tópico deve ser preservada, e nenhum dado pode ser perdido.

## Goals

- [ ] Fila FIFO nativa por tópico com processamento sequencial garantido
- [ ] Persistência em SQLite com transação atômica por mensagem
- [ ] Lock transacional para consistência entre as 5 filas concorrentes
- [ ] Backpressure natural via channels quando SQLite não acompanha
- [ ] Forwarding de cada mensagem persistida para o canal de workers

## Out of Scope

| Feature | Reason |
|---------|--------|
| Dead-letter queue | Complexidade para v1; erros são logados |
| Retry automático de persistência | SQLite local raramente falha; log é suficiente |
| Compactação/rotação do SQLite | v2 |
| Batch insert | Simplicidade; 1 insert por mensagem garante ordem |

---

## User Stories

### P1: Entidade Message ⭐ MVP

**User Story**: Como desenvolvedor do pipeline, preciso de uma entidade Message que represente um dado recebido do broker com todos os metadados necessários para persistência.

**Why P1**: Tipo compartilhado entre session → ingestion → dispatch.

**Acceptance Criteria**:

1. WHEN crio uma Message THEN devo ter ClientID, Topic, Payload, Timezone, Timestamp
2. WHEN serializo o Payload THEN deve ser tratado como JSON ([]byte)
3. WHEN o Timestamp é criado THEN deve usar `time.Now()` com timezone configurado

**Independent Test**: Criação e validação de campos.

---

### P1: Fila FIFO por Tópico ⭐ MVP

**User Story**: Como broker, preciso de uma fila FIFO para cada tópico que garanta processamento sequencial das mensagens na ordem de chegada.

**Why P1**: Garantia de ordem é requisito fundamental para dados industriais.

**Acceptance Criteria**:

1. WHEN enfileiro mensagens A, B, C THEN devem ser processadas na ordem A, B, C
2. WHEN a fila está vazia THEN o consumer deve bloquear esperando
3. WHEN o context é cancelado THEN o consumer deve parar graciosamente
4. WHEN o buffer da fila está cheio THEN o Enqueue deve bloquear (backpressure)
5. WHEN o consumer processa uma mensagem THEN deve chamar o Store e depois enviar ao dispatchChan

**Independent Test**: Testes com channels verificando ordem e blocking.

---

### P1: SQLite Store (raw_data) ⭐ MVP

**User Story**: Como broker, preciso persistir cada mensagem na tabela `raw_data` do SQLite com transação atômica para garantir zero data loss.

**Why P1**: Persistência é o core do pipeline — sem ela, dados se perdem.

**Acceptance Criteria**:

1. WHEN salvo uma mensagem THEN deve ser inserida na tabela raw_data com todos os campos
2. WHEN 2 filas tentam salvar simultaneamente THEN o mutex deve serializar as escritas
3. WHEN a transação falha THEN deve retornar erro sem corromper dados
4. WHEN o store é criado THEN deve criar a tabela e índices automaticamente (migrate)
5. WHEN consulto por tópico THEN devo encontrar as mensagens na ordem de inserção
6. WHEN o store é fechado THEN a conexão SQLite deve ser liberada

**Independent Test**: Testes de integração com SQLite `:memory:`.

---

### P1: Pipeline Orchestrator ⭐ MVP

**User Story**: Como broker, preciso de um orquestrador que roteia mensagens do canal de entrada para a fila correta por tópico e inicia os consumers.

**Why P1**: Cola tudo junto — recebe do session, distribui para filas, persiste, encaminha.

**Acceptance Criteria**:

1. WHEN uma mensagem chega no canal de entrada THEN deve ser roteada para a fila do tópico correto
2. WHEN uma mensagem é para tópico desconhecido THEN deve ser descartada (logada)
3. WHEN o pipeline inicia THEN deve criar 1 consumer por tópico
4. WHEN o context é cancelado THEN todos os consumers devem parar
5. WHEN uma mensagem é persistida com sucesso THEN deve ser enviada ao dispatchChan

**Independent Test**: Teste com channels mockando store.

---

## Edge Cases

- WHEN SQLite retorna erro de disco cheio THEN system SHALL logar erro e continuar tentando próxima mensagem
- WHEN o canal de dispatch está cheio THEN system SHALL bloquear até ter espaço (backpressure cascata)
- WHEN o payload não é JSON válido THEN system SHALL salvar mesmo assim (payload é opaco)
- WHEN o tópico da mensagem não existe no pipeline THEN system SHALL descartar e logar warning
- WHEN o store é criado com path inválido THEN system SHALL retornar erro na inicialização

---

## Requirement Traceability

| Requirement ID | Story | Phase | Status |
|---------------|-------|-------|--------|
| INGEST-01 | P1: Entidade Message | Tasks | Pending |
| INGEST-02 | P1: Fila FIFO | Tasks | Pending |
| INGEST-03 | P1: SQLite Store | Tasks | Pending |
| INGEST-04 | P1: Pipeline Orchestrator | Tasks | Pending |

**Coverage:** 4 total, 4 mapped to tasks, 0 unmapped ✅

---

## Success Criteria

- [ ] Mensagens publicadas em tópicos diferentes são persistidas na ordem correta por tópico
- [ ] SQLite contém todos os dados após 1000 PUBLISH rápidos
- [ ] Backpressure funciona: fila cheia bloqueia o handler sem perder dados
- [ ] Todos os testes passam com `go test -race ./internal/ingestion/...`
