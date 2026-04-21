# Project State

## Decisions

| Date | Decision | Context |
|------|----------|---------|
| 2026-04-21 | MQTT 3.1.1 (não 5.0) | Maior compatibilidade com devices industriais |
| 2026-04-21 | QoS 0 e 1 apenas | QoS 2 complexo demais para benefício no edge |
| 2026-04-21 | `modernc.org/sqlite` | Pure Go, sem CGO, cross-compilation |
| 2026-04-21 | Arquitetura modular com channels | Meio-termo entre hexagonal e pragmatismo |
| 2026-04-21 | Max 5 clients, max 5 tópicos | Design constraint para edge computing |

## Blockers

_Nenhum no momento._

## Lessons

_Nenhuma ainda._

## Todos

- [ ] Criar TESTING.md após definir estratégia de testes
- [ ] Avaliar necessidade de migration tool para SQLite

## Deferred Ideas

- TLS/SSL (v2)
- Workers concretos: Kafka, S3, REST (v2)
- Cálculos OEE, contagem de peças (v3)
- WebSocket output (v3)

## Preferences

_Nenhuma registrada._
