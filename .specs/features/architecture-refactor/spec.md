# Architecture Refactor Specification

## Problem Statement

O projeto `microbroker-mqtt-edge` tem código de excelente qualidade, mas a organização dos módulos mistura bounded contexts, usa nomes enganosos, e cria acoplamento desnecessário entre domínios. O módulo `session` é um God Module com 3+ responsabilidades, `dispatch` não reflete o padrão fan-out, `Message` está duplicada em 3 lugares, e `audit` está perdido em `common/`. A refatoração visa alinhar a estrutura com DDD e Clean Architecture sem alterar nenhuma lógica de negócio.

## Goals

- [ ] Cada módulo representa um único bounded context com vocabulário ubíquo claro
- [ ] Zero acoplamento cross-domain (módulos dependem apenas de `common/` e seu próprio domínio)
- [ ] Nomes de módulos e structs refletem o que o código realmente faz
- [ ] `Message` value object unificado elimina duplicação e bridge goroutine
- [ ] Auth extraído como módulo independente, preparado para crescimento futuro
- [ ] Todos os testes passam após refatoração (zero mudança de lógica)

## Out of Scope

| Feature | Reason |
|---|---|
| Novas funcionalidades de auth (tokens, providers) | Apenas extração estrutural, lógica idêntica |
| Mudanças no schema do banco | Refatoração puramente estrutural |
| Novos testes | Apenas atualização de imports nos testes existentes |
| Mudanças no protocolo MQTT | Módulo protocol está perfeito |
| Mudanças nos K6 stress tests | Não dependem de imports Go |
| Refatoração de lógica de negócio | Apenas reorganização estrutural |

---

## User Stories

### P1: Unificar Message value object ⭐ MVP

**User Story**: Como desenvolvedor, quero um único `Message` type compartilhado para que não exista duplicação nem bridge goroutine manual entre módulos.

**Why P1**: Elimina a raiz do acoplamento — 3 definições idênticas e uma goroutine de conversão desnecessária.

**Acceptance Criteria**:

1. WHEN o projeto compila THEN `common/domain/message.go` SHALL ser a única definição de `Message`
2. WHEN `connection` emite uma mensagem THEN SHALL usar `common/domain.Message` diretamente
3. WHEN `ingestion` recebe uma mensagem THEN SHALL usar `common/domain.Message` sem conversão
4. WHEN `processing` recebe uma mensagem THEN SHALL usar `common/domain.Message` sem type alias
5. WHEN o main.go é executado THEN SHALL NÃO existir goroutine bridge entre channels

**Independent Test**: `go build ./...` compila + `go test ./...` passa com zero bridge goroutine no main.go

---

### P1: Extrair módulo auth ⭐ MVP

**User Story**: Como desenvolvedor, quero auth como módulo independente para que possa crescer (external providers, tokens, Kafka SASL) sem acoplar ao módulo de conexão.

**Why P1**: Auth tem bounded context próprio e potencial de crescimento significativo. Extrair agora é barato; extrair depois com múltiplos providers seria custoso.

**Acceptance Criteria**:

1. WHEN o módulo `auth` existe THEN SHALL conter `domain/authenticator.go` com a interface `Authenticator`
2. WHEN o módulo `auth` existe THEN SHALL conter `env_authenticator.go` com a implementação atual
3. WHEN `connection` precisa autenticar THEN SHALL receber `Authenticator` via injeção de dependência
4. WHEN o bootstrap wira os módulos THEN SHALL criar `EnvAuthenticator` e injetar em `connection.NewServer`
5. WHEN `auth_test.go` executa THEN SHALL validar os mesmos cenários que o teste atual

**Independent Test**: `go test ./internal/modules/auth/...` passa com os mesmos cenários

---

### P1: Renomear session → connection ⭐ MVP

**User Story**: Como desenvolvedor, quero que o módulo de gerenciamento de conexões TCP se chame `connection` para que o nome reflita o vocabulário ubíquo do domínio.

**Why P1**: "Session" é ambíguo no contexto MQTT (session state vs connection). O módulo gerencia conexões TCP.

**Acceptance Criteria**:

1. WHEN o módulo `connection` existe THEN SHALL conter todos os arquivos de `session` (exceto auth)
2. WHEN `ConnectionManager` é renomeado THEN SHALL se chamar `ClientManager`
3. WHEN imports referenciam `session` THEN SHALL referenciar `connection`
4. WHEN testes executam THEN SHALL passar com os novos paths

**Independent Test**: `go test ./internal/modules/connection/...` passa

---

### P1: Renomear dispatch → processing ⭐ MVP

**User Story**: Como desenvolvedor, quero que o módulo de fan-out se chame `processing` e a struct `FanOut` para que nomes reflitam o padrão arquitetural real.

**Why P1**: "Dispatch" sugere roteamento para destino específico; o módulo faz broadcast/fan-out 1:N.

**Acceptance Criteria**:

1. WHEN o módulo `processing` existe THEN SHALL conter todos os arquivos de `dispatch`
2. WHEN `Dispatcher` é renomeado THEN SHALL se chamar `FanOut`
3. WHEN `processing/domain/worker.go` importa Message THEN SHALL importar de `common/domain`
4. WHEN testes executam THEN SHALL passar com os novos paths

**Independent Test**: `go test ./internal/modules/processing/...` passa

---

### P1: Consolidar Logger em common/observability ⭐ MVP

**User Story**: Como desenvolvedor, quero uma única interface `Logger` canônica para que não existam 6+ definições idênticas espalhadas pelo projeto.

**Why P1**: Duplicação de interfaces idênticas é ruído que dificulta manutenção.

**Acceptance Criteria**:

1. WHEN `common/observability/logger.go` existe THEN SHALL conter `Logger`, `NopLogger`, e `NewSlogLogger`
2. WHEN módulos precisam de Logger THEN SHALL importar de `common/observability`
3. WHEN `workers.Logger` (subset com só `Info`) existe THEN SHALL manter definição local (subset legítimo)
4. WHEN `common/logger.go` antigo é removido THEN SHALL não existir mais

**Independent Test**: `go build ./...` compila sem erros

---

### P1: Mover audit para modules ⭐ MVP

**User Story**: Como desenvolvedor, quero audit como módulo de negócio em `modules/audit/` para que `common/` contenha apenas utilitários compartilhados.

**Why P1**: Audit é um módulo de negócio completo (handler HTTP, reader, DTOs), não um utilitário.

**Acceptance Criteria**:

1. WHEN `modules/audit/` existe THEN SHALL conter handler, interfaces, domain/record, adapters/outbound/database
2. WHEN `common/audit/` é removido THEN SHALL não existir mais
3. WHEN o HTTP handler é registrado THEN SHALL funcionar identicamente
4. WHEN testes executam THEN SHALL passar com os novos paths

**Independent Test**: `go test ./internal/modules/audit/...` passa

---

### P1: Reestruturar bootstrap ⭐ MVP

**User Story**: Como desenvolvedor, quero o entrypoint em `cmd/broker/` com bootstrap separado para que o main.go seja slim e a inicialização seja organizada.

**Why P1**: main.go com 140 linhas fazendo tudo não escala. Bootstrap separado permite crescimento limpo.

**Acceptance Criteria**:

1. WHEN `cmd/broker/main.go` existe THEN SHALL ter ~20 linhas delegando para bootstrap
2. WHEN `cmd/broker/bootstrap/` existe THEN SHALL conter database.go, modules.go, server.go, shutdown.go
3. WHEN `cmd/broker/config/` existe THEN SHALL conter a config movida de `internal/config/`
4. WHEN o broker inicia THEN SHALL funcionar identicamente ao atual
5. WHEN `cmd/main.go` antigo é removido THEN SHALL não existir mais

**Independent Test**: `go build ./cmd/broker/` compila + E2E tests passam

---

## Edge Cases

- WHEN um módulo é renomeado e imports não são atualizados THEN `go build ./...` SHALL falhar (detectável)
- WHEN testes referenciam paths antigos THEN `go test ./...` SHALL falhar (detectável)
- WHEN o Dockerfile referencia `cmd/main.go` THEN SHALL ser atualizado para `cmd/broker/main.go`
- WHEN `docker-compose.yml` referencia entrypoint THEN SHALL ser verificado e atualizado se necessário

---

## Requirement Traceability

| Requirement ID | Story | Phase | Status |
|---|---|---|---|
| REFAC-01 | P1: Unificar Message | Tasks | Pending |
| REFAC-02 | P1: Extrair auth | Tasks | Pending |
| REFAC-03 | P1: Renomear session → connection | Tasks | Pending |
| REFAC-04 | P1: Renomear dispatch → processing | Tasks | Pending |
| REFAC-05 | P1: Consolidar Logger | Tasks | Pending |
| REFAC-06 | P1: Mover audit | Tasks | Pending |
| REFAC-07 | P1: Reestruturar bootstrap | Tasks | Pending |

**Coverage**: 7 total, 7 mapped to tasks, 0 unmapped ✅

---

## Success Criteria

- [ ] `go build ./...` compila sem erros
- [ ] `go test ./...` todos os testes passam (incluindo E2E)
- [ ] `golangci-lint run ./...` sem warnings
- [ ] `go vet ./...` sem issues
- [ ] Zero mudança de lógica de negócio — apenas reorganização estrutural
- [ ] Nenhum módulo importa domínio de outro módulo (apenas `common/`)
