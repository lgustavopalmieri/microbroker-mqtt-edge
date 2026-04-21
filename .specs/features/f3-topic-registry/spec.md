# F3: Topic Registry — Specification

**Status**: COMPLETE ✅ (implementada dentro de F2: Session Management)

## Problem Statement

O broker precisa aceitar dados apenas em tópicos pré-definidos por variável de ambiente, com limite máximo de 5 tópicos. Qualquer PUBLISH em tópico não permitido deve ser ignorado.

## Goals

- [x] Registro de tópicos via variável de ambiente (lista separada por vírgula)
- [x] Validação de tópicos no PUBLISH (apenas tópicos permitidos são aceitos)
- [x] Máximo de 5 tópicos por broker
- [x] Validação na inicialização (rejeita config inválida)

## Out of Scope

| Feature | Reason |
|---------|--------|
| Wildcards (+, #) | Complexidade desnecessária para edge |
| Tópicos dinâmicos (add/remove em runtime) | Restart é aceitável no edge |
| Hierarquia de tópicos | Matching exato é suficiente |

---

## User Stories

### P1: Registro e Validação de Tópicos ⭐ MVP

**User Story**: Como operador do broker, quero definir os tópicos permitidos via variável de ambiente para controlar quais dados o broker aceita.

**Why P1**: Sem isso, o broker aceitaria qualquer tópico — sem controle.

**Acceptance Criteria**:

1. WHEN crio um TopicRegistry com 1-5 tópicos válidos THEN deve criar sem erro ✅
2. WHEN crio com 0 tópicos THEN deve retornar erro ✅
3. WHEN crio com >5 tópicos THEN deve retornar erro ✅
4. WHEN crio com tópico vazio THEN deve retornar erro ✅
5. WHEN verifico um tópico permitido THEN IsAllowed retorna true ✅
6. WHEN verifico um tópico não permitido THEN IsAllowed retorna false ✅
7. WHEN tópicos têm espaços extras THEN devem ser trimados ✅
8. WHEN listo os tópicos THEN todos devem ser retornados ✅

**Independent Test**: 8 testes unitários passando em `internal/session/domain/topic_registry_test.go`

---

### P1: Validação no PUBLISH ⭐ MVP

**User Story**: Como broker, quero rejeitar silenciosamente PUBLISH em tópicos não permitidos.

**Acceptance Criteria**:

1. WHEN client publica em tópico permitido THEN mensagem é processada ✅
2. WHEN client publica em tópico não permitido THEN mensagem é ignorada ✅
3. WHEN client publica QoS 1 em tópico não permitido THEN PUBACK é enviado (evita retries) ✅

**Independent Test**: Testado em `internal/session/handler_test.go` (TestHandler_PublishDisallowedTopic)

---

## Implementation Note

F3 foi implementada como parte de F2 (Session Management) por ser um value object do domínio de sessão:
- `internal/session/domain/topic_registry.go` — TopicRegistry struct com NewTopicRegistry, IsAllowed, Topics
- `internal/session/domain/topic_registry_test.go` — 8 testes unitários
- `internal/session/handler.go` — Validação no handlePublish e handleSubscribe

Não justificava um módulo separado — é um value object coeso com o domínio de sessão.

---

## Requirement Traceability

| Requirement ID | Story | Phase | Status |
|---------------|-------|-------|--------|
| TOPIC-01 | P1: Registro e Validação | Done | ✅ Verified |
| TOPIC-02 | P1: Validação no PUBLISH | Done | ✅ Verified |

**Coverage:** 2 total, 2 implemented, 2 verified ✅

---

## Success Criteria

- [x] TopicRegistry criado com validação completa (1-5 tópicos, sem vazios, trim)
- [x] PUBLISH em tópico não permitido é ignorado silenciosamente
- [x] SUBSCRIBE em tópico não permitido retorna failure code (0x80)
- [x] 8 testes unitários passando com `go test -race ./internal/session/domain/...`
