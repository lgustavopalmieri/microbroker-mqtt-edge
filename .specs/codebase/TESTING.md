# Testing Strategy

## Test Types

| Type | Tool | Command | When |
|------|------|---------|------|
| Unit | `go test` + `testify` + `gomock` | `go test ./...` | Lógica pura, parsing, validações |
| Integration | `go test` + SQLite `:memory:` | `go test -tags=integration ./...` | Store, pipeline com DB real |
| E2E | `go test` + TCP real + SQLite `:memory:` | `go test -race ./tests/e2e/...` | Fluxo completo: client TCP → protocol → session → ingestion → SQLite → dispatch → worker |
| Race Detection | `go test -race` | `go test -race ./...` | Concorrência, channels, mutexes |

## Gate Check Commands

| Gate | Command | When to use |
|------|---------|-------------|
| quick | `go build ./... && go vet ./... && go test ./internal/modules/protocol/...` | Após mudanças em um pacote |
| full | `go build ./... && go vet ./... && go test -race ./... && golangci-lint run ./...` | Antes de commit |
| build | `go build ./...` | Verificação rápida de compilação |

## Test Coverage Matrix

| Code Layer | Location | Required Test Type | Parallel-Safe |
|-----------|----------|-------------------|---------------|
| Protocol types/constants | `internal/modules/protocol/` | unit | Yes |
| Protocol decoder | `internal/modules/protocol/` | unit | Yes |
| Protocol encoder | `internal/modules/protocol/` | unit | Yes |
| Session domain (Client, TopicRegistry) | `internal/modules/session/domain/` | unit | Yes |
| Auth | `internal/modules/session/` | unit | Yes |
| Connection Manager | `internal/modules/session/` | unit | Yes |
| TCP Server + Handler | `internal/modules/session/` | integration | No |
| Ingestion domain (Message) | `internal/modules/ingestion/domain/` | unit | Yes |
| Queue (FIFO) | `internal/modules/ingestion/application/` | unit | Yes |
| Pipeline | `internal/modules/ingestion/application/` | unit | Yes |
| SQLite Store | `internal/modules/ingestion/adapters/outbound/database/` | integration | No |
| Worker interface | `internal/modules/dispatch/domain/` | none | Yes |
| Dispatcher | `internal/modules/dispatch/` | unit | Yes |
| Logger Worker | `internal/modules/dispatch/workers/` | unit | Yes |
| Config | `internal/config/` | unit | Yes |
| Bootstrap (main.go) | `cmd/` | build | No |
| E2E (full pipeline) | `tests/e2e/` | e2e | No |

## Test Patterns

- Table-driven tests com `testify/assert` e `require`
- Mocks via `go.uber.org/mock/gomock`
- Factory functions para inputs: `xxxFactory(overrides ...func(*Type)) *Type`
- SQLite `:memory:` para testes de integração do store
- `net.Pipe()` para testes de integração TCP
- `context.WithCancel` para testar graceful shutdown

## Parallelism Assessment

- Testes unitários: parallel-safe (cada teste é independente)
- Testes de integração SQLite: NOT parallel-safe (SQLite single-writer)
- Testes de integração TCP: NOT parallel-safe (port binding)
- Testes e2e: NOT parallel-safe (TCP + SQLite combinados, cada teste sobe seu próprio broker em porta 0)
