# Testing Strategy

## Test Types

| Type | Tool | Command | When |
|------|------|---------|------|
| Unit | `go test` + `testify` + `gomock` | `go test ./...` | Lógica pura, parsing, validações |
| Integration | `go test` + SQLite `:memory:` | `go test -tags=integration ./...` | Store, pipeline com DB real |
| Race Detection | `go test -race` | `go test -race ./...` | Concorrência, channels, mutexes |

## Gate Check Commands

| Gate | Command | When to use |
|------|---------|-------------|
| quick | `go build ./... && go vet ./... && go test ./internal/protocol/...` | Após mudanças em um pacote |
| full | `go build ./... && go vet ./... && go test -race ./... && golangci-lint run ./...` | Antes de commit |
| build | `go build ./...` | Verificação rápida de compilação |

## Test Coverage Matrix

| Code Layer | Location | Required Test Type | Parallel-Safe |
|-----------|----------|-------------------|---------------|
| Protocol types/constants | `internal/protocol/` | unit | Yes |
| Protocol decoder | `internal/protocol/` | unit | Yes |
| Protocol encoder | `internal/protocol/` | unit | Yes |
| Session domain (Client, TopicRegistry) | `internal/session/domain/` | unit | Yes |
| Auth | `internal/session/` | unit | Yes |
| Connection Manager | `internal/session/` | unit | Yes |
| TCP Server + Handler | `internal/session/` | integration | No |
| Ingestion domain (Message) | `internal/ingestion/domain/` | unit | Yes |
| Queue (FIFO) | `internal/ingestion/` | unit | Yes |
| Pipeline | `internal/ingestion/` | unit | Yes |
| SQLite Store | `internal/ingestion/store/` | integration | No |
| Worker interface | `internal/dispatch/domain/` | none | Yes |
| Dispatcher | `internal/dispatch/` | unit | Yes |
| Logger Worker | `internal/dispatch/workers/` | unit | Yes |
| Config | `internal/config/` | unit | Yes |
| Bootstrap (main.go) | `cmd/` | build | No |

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
