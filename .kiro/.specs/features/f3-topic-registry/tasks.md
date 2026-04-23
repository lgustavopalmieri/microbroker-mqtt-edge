# F3: Topic Registry — Tasks

**Spec**: `.specs/features/f3-topic-registry/spec.md`
**Status**: Done (implementada dentro de F2)

---

## Implementation Note

F3 foi absorvida dentro de F2 (Session Management) durante o planejamento porque o TopicRegistry é um value object do domínio de sessão — não justifica módulo ou feature separada.

As tasks correspondentes em F2 são:
- **F2-T2**: Topic Registry (Value Object) → `internal/modules/session/domain/topic_registry.go` + testes
- **F2-T6**: Packet Handler → validação de tópicos no handlePublish e handleSubscribe

---

## Mapping F3 → F2

| F3 Requirement | F2 Task | File | Status |
|---------------|---------|------|--------|
| TOPIC-01: Registro e Validação | F2-T2 | `internal/modules/session/domain/topic_registry.go` | ✅ Done |
| TOPIC-02: Validação no PUBLISH | F2-T6 | `internal/modules/session/handler.go` | ✅ Done |

---

## Test Coverage

| Test File | Tests | Status |
|-----------|-------|--------|
| `internal/modules/session/domain/topic_registry_test.go` | 8 testes | ✅ All passing |
| `internal/modules/session/handler_test.go` (TestHandler_PublishDisallowedTopic) | 1 teste | ✅ Passing |
| `internal/modules/session/handler_test.go` (TestHandler_Subscribe) | 1 teste (failure code) | ✅ Passing |
