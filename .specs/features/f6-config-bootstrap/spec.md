# F6: Configuration & Bootstrap — Specification

## Problem Statement

Todos os módulos precisam ser conectados (wired) no main.go com configuração carregada de variáveis de ambiente. O bootstrap deve iniciar todos os componentes na ordem correta e fazer graceful shutdown quando receber SIGINT/SIGTERM.

## Goals

- [ ] Struct de configuração centralizada carregada de env vars
- [ ] Validação de configuração na inicialização
- [ ] Wiring de todos os módulos no main.go
- [ ] Graceful shutdown completo (listener → queues → workers → store)
- [ ] Logger configurado com slog
- [ ] Dockerfile e .env atualizados

## Out of Scope

| Feature | Reason |
|---------|--------|
| Config file (YAML/TOML) | Env vars são suficientes para edge |
| Hot reload de config | Restart é aceitável no edge |
| Health check endpoint | v2 |

---

## User Stories

### P1: Config Loader ⭐ MVP

**User Story**: Como operador do broker, quero configurar o broker via variáveis de ambiente para facilitar deploy em containers.

**Why P1**: Sem config, nada inicia.

**Acceptance Criteria**:

1. WHEN todas as env vars obrigatórias estão definidas THEN config deve carregar sem erro
2. WHEN BROKER_PORT não está definido THEN deve usar default 1883
3. WHEN BROKER_TOPICS está vazio THEN deve retornar erro
4. WHEN BROKER_MAX_CLIENTS > 5 THEN deve retornar erro
5. WHEN BROKER_USERNAME ou BROKER_PASSWORD estão vazios THEN deve retornar erro
6. WHEN BROKER_DB_PATH não está definido THEN deve usar default "./data/broker.db"

**Independent Test**: Table-driven tests com env vars mockadas.

---

### P1: Bootstrap (main.go) ⭐ MVP

**User Story**: Como operador, quero iniciar o broker com um único comando e que ele faça shutdown gracioso ao receber SIGINT.

**Why P1**: Entry point do sistema.

**Acceptance Criteria**:

1. WHEN executo o binário THEN deve carregar config, criar store, iniciar pipeline, dispatcher e server
2. WHEN envio SIGINT THEN deve parar de aceitar conexões, drenar filas e fechar store
3. WHEN ocorre erro fatal na inicialização THEN deve logar e sair com código 1
4. WHEN o broker inicia com sucesso THEN deve logar endereço e tópicos configurados

**Independent Test**: Build do binário + verificação de que inicia e para.

---

### P1: Common Logger Interface ⭐ MVP

**User Story**: Como desenvolvedor, preciso de uma interface de logging que todos os módulos usem, sem depender de implementação concreta.

**Why P1**: Observabilidade básica.

**Acceptance Criteria**:

1. WHEN uso o logger THEN devo ter métodos Info, Error, Warn, Debug com key-value pairs
2. WHEN o logger é injetado THEN cada módulo deve usá-lo sem importar slog diretamente
3. WHEN o broker inicia THEN o logger default deve ser slog com output em JSON ou text

**Independent Test**: Compilação com logger injetado.

---

### P2: Dockerfile e Docker Compose Atualizados

**User Story**: Como operador, quero fazer deploy do broker via Docker com todas as configurações corretas.

**Why P2**: Importante mas não bloqueia desenvolvimento.

**Acceptance Criteria**:

1. WHEN faço docker build THEN o binário deve compilar com CGO habilitado (para modernc sqlite)
2. WHEN faço docker run THEN o broker deve iniciar com env vars do .env
3. WHEN o container para THEN o broker deve fazer graceful shutdown

**Independent Test**: `docker build` + `docker run` com verificação de logs.

---

## Edge Cases

- WHEN env var tem espaços extras THEN system SHALL fazer trim
- WHEN BROKER_TOPICS tem vírgulas extras (ex: "a,,b") THEN system SHALL ignorar vazios
- WHEN o diretório do DB_PATH não existe THEN system SHALL criar automaticamente
- WHEN shutdown timeout expira THEN system SHALL forçar saída

---

## Requirement Traceability

| Requirement ID | Story | Phase | Status |
|---------------|-------|-------|--------|
| BOOT-01 | P1: Config Loader | Tasks | Pending |
| BOOT-02 | P1: Bootstrap | Tasks | Pending |
| BOOT-03 | P1: Logger Interface | Tasks | Pending |
| BOOT-04 | P2: Docker | Tasks | Pending |

**Coverage:** 4 total, 4 mapped to tasks, 0 unmapped ✅

---

## Success Criteria

- [ ] `go run ./cmd/main.go` inicia o broker e loga configuração
- [ ] SIGINT causa shutdown gracioso com log de confirmação
- [ ] Config inválida causa saída com mensagem de erro clara
- [ ] `docker build` e `docker run` funcionam
