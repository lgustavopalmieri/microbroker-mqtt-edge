# Feature Scaffolding Reference (Go)

This reference covers the creation of features within a module. Each feature follows the `application/ + adapters/` structure (Clean Architecture / Hexagonal).

> **Implementation note**: All infrastructure tools mentioned in this document (PostgreSQL, Elasticsearch, Kafka, etc.) are examples of implementation choices. The architecture is designed around interfaces and ports — any tool can be swapped or added at any time (e.g. MySQL instead of PostgreSQL, OpenSearch instead of Elasticsearch, RabbitMQ instead of Kafka). Only the `adapters/outbound/` and `adapters/inbound/` implementations change; the domain and application layers remain untouched.

---

## Feature Types

Before scaffolding, determine the feature type:

| Type | Example | Outbound Adapter | Event Listeners | Typical Mocks |
|------|---------|-------------------|-----------------|---------------|
| Write (CRUD) | create, update | `database/` (e.g. PostgreSQL, MySQL) | Yes (side effects) | 4: repository, event_dispatcher, logger, tracer |
| Read/Search | search, list | `<search-engine>/` (e.g. Elasticsearch, OpenSearch) or `database/` | No | 1: repository |
| Event-Driven | authorize_license | `database/` + `external/` | N/A (IS a listener) | 4: repository, event_dispatcher, logger, tracer |

---

## Feature Structure: Write Feature (e.g., create, update)

```
features/<feature-name>/
├── application/
│   ├── usecase.go                         # Use case logic (Execute method)
│   ├── usecase_test.go                    # Unit tests (table-driven)
│   ├── new_usecase.go                     # Constructor (dependency injection)
│   ├── interface.go                       # Port interfaces (Repository, EventDispatcher, Logger, Tracer)
│   ├── dto.go                             # Application-level DTOs (Input/Output)
│   ├── constants.go                       # Span names, event names, error messages
│   └── mocks/                             # gomock-generated mocks
│       ├── event_dispatcher_mock.go
│       ├── logger_mock.go
│       ├── repository_mock.go
│       └── tracer_mock.go
├── event_listeners/                       # Event-driven side effects (optional)
│   ├── <listener-name>/                   # e.g., validate_license, send_credentials_email
│   │   ├── listener/                      # Mirrors application/ pattern
│   │   │   ├── handler.go
│   │   │   ├── handler_test.go
│   │   │   ├── new_handler.go
│   │   │   ├── interface.go
│   │   │   ├── dto.go
│   │   │   ├── constants.go
│   │   │   └── mocks/
│   │   └── adapters/
│   │       ├── inbound/
│   │       │   └── <broker>/              # e.g. kafka/, rabbitmq/, sqs/ — manager.go (consumer management)
│   │       └── outbound/
│   │           ├── database/              # repository.go, new.go, errors.go
│   │           └── external/              # gateway.go, new.go (external API calls)
│   └── <placeholder-listener>/            # Empty folder for future implementation
└── adapters/
    ├── inbound/
    │   ├── http_handler/
    │   │   ├── handler.go                 # Route handler functions
    │   │   ├── handler_test.go
    │   │   ├── dto.go                     # HTTP request/response DTOs
    │   │   ├── di.go                      # Dependency injection / constructor
    │   │   └── mocks/
    │   ├── grpc_service/
    │   │   ├── service.go                 # gRPC service implementation
    │   │   ├── service_test.go
    │   │   ├── dto.go                     # gRPC-specific DTOs / mappers
    │   │   ├── di.go                      # Dependency injection / constructor
    │   │   ├── mocks/
    │   │   ├── pb/                        # Generated protobuf Go code
    │   │   └── proto/                     # .proto source files
    │   └── handler_integration_test.go    # Integration test (optional, at inbound/ root)
    └── outbound/
        └── database/                      # SQL repository (e.g. PostgreSQL, MySQL — swappable)
            ├── repository.go              # SQL repository implementation
            ├── repository_test.go         # Integration tests (testcontainers)
            ├── new.go                     # Constructor
            └── errors.go                  # DB-specific error mapping
```

## Feature Structure: Read/Search Feature

```
features/<feature-name>/
├── application/
│   ├── usecase.go
│   ├── usecase_test.go
│   ├── new_usecase.go
│   ├── interface.go                       # Only Repository interface (fewer deps)
│   ├── dto.go
│   ├── constants.go
│   └── mocks/
│       └── repository_mock.go             # Only 1 mock needed
└── adapters/
    ├── inbound/
    │   ├── http_handler/
    │   │   ├── handler.go
    │   │   ├── handler_test.go
    │   │   ├── dto.go
    │   │   ├── di.go
    │   │   └── mocks/
    │   └── grpc_service/
    │       ├── service.go
    │       ├── service_test.go
    │       ├── dto.go
    │       ├── di.go
    │       ├── mocks/
    │       ├── pb/
    │       └── proto/
    └── outbound/
        └── <search-engine>/               # e.g. elasticsearch/, opensearch/, meilisearch/ — swappable
            ├── repository.go
            ├── repository_test.go
            ├── new.go
            ├── errors.go
            ├── builders.go                # Query builders (engine-specific)
            ├── mappers.go                 # Response mappers (engine → domain)
            └── dto.go                     # Engine-specific DTOs
```

---

## Component Templates

### application/interface.go (Write Feature)

```go
package application

import (
    "context"
    "myapp/internal/common/event"
    "myapp/internal/common/observability"
    "myapp/internal/modules/<module>/domain"
)

type Repository interface {
    Save(ctx context.Context, entity *domain.<Entity>) error
    FindByEmail(ctx context.Context, email string) (*domain.<Entity>, error)
}

type EventDispatcher interface {
    Dispatch(ctx context.Context, evt event.Event) error
}

type Logger interface {
    observability.Logger
}

type Tracer interface {
    observability.Tracer
}
```

### application/interface.go (Read/Search Feature)

```go
package application

import (
    "context"
    searchinput "myapp/internal/modules/<module>/domain/search/search_input"
    searchoutput "myapp/internal/modules/<module>/domain/search/search_output"
)

type Repository interface {
    Search(ctx context.Context, input searchinput.Input) (*searchoutput.Output, error)
}
```

### application/dto.go

```go
package application

type <Feature>Input struct {
    Name    string
    Email   string
    License string
}

type <Feature>Output struct {
    ID     string
    Status string
}
```

### application/constants.go

```go
package application

const (
    SpanName<Feature>UseCase = "<Feature>UseCase.Execute"
    Event<Entity><Feature>d  = "<entity>.<feature>d" // e.g., "specialist.created"
    ErrMsg<Feature>Failed    = "failed to <feature> <entity>"
)
```

### application/new_usecase.go

```go
package application

type <Feature>UseCase struct {
    repo       Repository
    dispatcher EventDispatcher
    logger     Logger
    tracer     Tracer
}

func New<Feature>UseCase(
    repo Repository,
    dispatcher EventDispatcher,
    logger Logger,
    tracer Tracer,
) *<Feature>UseCase {
    return &<Feature>UseCase{
        repo:       repo,
        dispatcher: dispatcher,
        logger:     logger,
        tracer:     tracer,
    }
}
```

### application/usecase.go

```go
package application

import (
    "context"
    "fmt"

    "myapp/internal/common/event"
    featureDomain "myapp/internal/modules/<module>/domain/<feature>"
)

func (uc *<Feature>UseCase) Execute(ctx context.Context, input <Feature>Input) (*<Feature>Output, error) {
    ctx, span := uc.tracer.Start(ctx, SpanName<Feature>UseCase)
    defer span.End()

    uc.logger.Info(ctx, "executing <feature>", "input", input)

    entity, err := featureDomain.NewEntity(featureDomain.<Feature>Input{
        Name:    input.Name,
        Email:   input.Email,
        License: input.License,
    })
    if err != nil {
        return nil, fmt.Errorf("%s: %w", ErrMsg<Feature>Failed, err)
    }

    if err := uc.repo.Save(ctx, entity); err != nil {
        return nil, fmt.Errorf("%s: %w", ErrMsg<Feature>Failed, err)
    }

    _ = uc.dispatcher.Dispatch(ctx, event.Event{
        Name:    Event<Entity><Feature>d,
        Payload: entity,
    })

    return &<Feature>Output{
        ID:     entity.ID,
        Status: string(entity.Status),
    }, nil
}
```

### application/usecase_test.go (Table-Driven)

```go
package application_test

import (
    "context"
    "testing"

    "github.com/golang/mock/gomock"
    "myapp/internal/modules/<module>/features/<feature>/application"
    "myapp/internal/modules/<module>/features/<feature>/application/mocks"
)

func TestExecute(t *testing.T) {
    tests := []struct {
        name      string
        input     application.<Feature>Input
        setupMock func(repo *mocks.MockRepository, dispatcher *mocks.MockEventDispatcher, logger *mocks.MockLogger, tracer *mocks.MockTracer)
        wantErr   bool
        errMsg    string
    }{
        {
            name: "success",
            input: application.<Feature>Input{
                Name:    "John Doe",
                Email:   "john@example.com",
                License: "LIC-001",
            },
            setupMock: func(repo *mocks.MockRepository, dispatcher *mocks.MockEventDispatcher, logger *mocks.MockLogger, tracer *mocks.MockTracer) {
                tracer.EXPECT().Start(gomock.Any(), gomock.Any()).Return(context.Background(), &noopSpan{})
                logger.EXPECT().Info(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
                repo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)
                dispatcher.EXPECT().Dispatch(gomock.Any(), gomock.Any()).Return(nil)
            },
            wantErr: false,
        },
        // ... more test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            ctrl := gomock.NewController(t)
            defer ctrl.Finish()

            repo := mocks.NewMockRepository(ctrl)
            dispatcher := mocks.NewMockEventDispatcher(ctrl)
            logger := mocks.NewMockLogger(ctrl)
            tracer := mocks.NewMockTracer(ctrl)

            tt.setupMock(repo, dispatcher, logger, tracer)

            uc := application.New<Feature>UseCase(repo, dispatcher, logger, tracer)
            _, err := uc.Execute(context.Background(), tt.input)

            if (err != nil) != tt.wantErr {
                t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

---

## Adapter Templates

### adapters/inbound/http_handler/di.go

```go
package httphandler

type Handler struct {
    useCase UseCasePort
}

func NewHandler(useCase UseCasePort) *Handler {
    return &Handler{useCase: useCase}
}
```

### adapters/inbound/http_handler/handler.go

```go
package httphandler

import (
    "encoding/json"
    "net/http"

    "myapp/internal/modules/<module>/features/<feature>/application"
)

type UseCasePort interface {
    Execute(ctx context.Context, input application.<Feature>Input) (*application.<Feature>Output, error)
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
    var req <Feature>Request
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid request body", http.StatusBadRequest)
        return
    }

    output, err := h.useCase.Execute(r.Context(), application.<Feature>Input{
        Name:    req.Name,
        Email:   req.Email,
        License: req.License,
    })
    if err != nil {
        // Map domain errors to HTTP status codes
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(<Feature>Response{
        ID:     output.ID,
        Status: output.Status,
    })
}
```

### adapters/inbound/http_handler/dto.go

```go
package httphandler

type <Feature>Request struct {
    Name    string `json:"name"`
    Email   string `json:"email"`
    License string `json:"license"`
}

type <Feature>Response struct {
    ID     string `json:"id"`
    Status string `json:"status"`
}
```

### adapters/inbound/grpc_service/di.go

```go
package grpcservice

import "myapp/internal/modules/<module>/features/<feature>/application"

type Service struct {
    pb.Unimplemented<Entity>ServiceServer
    useCase UseCasePort
}

func NewService(useCase UseCasePort) *Service {
    return &Service{useCase: useCase}
}
```

### adapters/inbound/grpc_service/service.go

```go
package grpcservice

import (
    "context"

    "myapp/internal/modules/<module>/features/<feature>/application"
    pb "myapp/internal/modules/<module>/features/<feature>/adapters/inbound/grpc_service/pb"
)

type UseCasePort interface {
    Execute(ctx context.Context, input application.<Feature>Input) (*application.<Feature>Output, error)
}

func (s *Service) <Feature>(ctx context.Context, req *pb.<Feature>Request) (*pb.<Feature>Response, error) {
    output, err := s.useCase.Execute(ctx, application.<Feature>Input{
        Name:    req.Name,
        Email:   req.Email,
        License: req.License,
    })
    if err != nil {
        return nil, mapToGRPCError(err)
    }

    return &pb.<Feature>Response{
        Id:     output.ID,
        Status: output.Status,
    }, nil
}
```

### adapters/outbound/database/new.go

> Note: This template uses PostgreSQL as an example. Replace with your project's database (MySQL, SQLite, CockroachDB, etc.). The application layer's `Repository` interface remains the same — only this adapter implementation changes.

```go
package database

import "database/sql"

type PostgresRepository struct {
    db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
    return &PostgresRepository{db: db}
}
```

### adapters/outbound/database/repository.go

```go
package database

import (
    "context"
    "fmt"

    "myapp/internal/modules/<module>/domain"
)

func (r *PostgresRepository) Save(ctx context.Context, entity *domain.<Entity>) error {
    _, err := r.db.ExecContext(ctx,
        `INSERT INTO <table> (id, name, email, license, status, created_at, updated_at)
         VALUES ($1, $2, $3, $4, $5, $6, $7)`,
        entity.ID, entity.Name, entity.Email, entity.License,
        entity.Status, entity.CreatedAt, entity.UpdatedAt,
    )
    if err != nil {
        return fmt.Errorf("%w: %v", ErrSaveFailed, err)
    }
    return nil
}

func (r *PostgresRepository) FindByEmail(ctx context.Context, email string) (*domain.<Entity>, error) {
    row := r.db.QueryRowContext(ctx,
        `SELECT id, name, email, license, status, created_at, updated_at
         FROM <table> WHERE email = $1`, email,
    )
    var entity domain.<Entity>
    if err := row.Scan(
        &entity.ID, &entity.Name, &entity.Email, &entity.License,
        &entity.Status, &entity.CreatedAt, &entity.UpdatedAt,
    ); err != nil {
        return nil, fmt.Errorf("%w: %v", ErrFindFailed, err)
    }
    return &entity, nil
}
```

### adapters/outbound/database/errors.go

```go
package database

import "errors"

var (
    ErrSaveFailed = errors.New("failed to save entity")
    ErrFindFailed = errors.New("failed to find entity")
)
```

### adapters/outbound/elasticsearch/ (Search Feature)

> Note: This template uses Elasticsearch as an example. Replace with your project's search engine (OpenSearch, Meilisearch, Typesense, or even PostgreSQL full-text search). The application layer's `Repository` interface remains the same — only this adapter implementation changes.

```go
// repository.go
package elasticsearch

import (
    "context"
    searchinput "myapp/internal/modules/<module>/domain/search/search_input"
    searchoutput "myapp/internal/modules/<module>/domain/search/search_output"
)

type ElasticsearchRepository struct {
    client *elasticsearch.Client
    index  string
}

func (r *ElasticsearchRepository) Search(ctx context.Context, input searchinput.Input) (*searchoutput.Output, error) {
    query := r.buildQuery(input)       // builders.go
    result, err := r.execute(ctx, query)
    if err != nil {
        return nil, fmt.Errorf("%w: %v", ErrSearchFailed, err)
    }
    return r.mapResponse(result), nil  // mappers.go
}
```

```go
// builders.go — ES-specific query construction
package elasticsearch

func (r *ElasticsearchRepository) buildQuery(input searchinput.Input) map[string]interface{} {
    // Build Elasticsearch query DSL from domain input
}
```

```go
// mappers.go — ES response → domain output
package elasticsearch

func (r *ElasticsearchRepository) mapResponse(result *esResult) *searchoutput.Output {
    // Map Elasticsearch response to domain output struct
}
```

---

## Event Listeners

Event listeners are independent mini-modules within a feature. They follow the same `listener/ + adapters/` separation.

### When to Create Event Listeners

- The feature emits domain events (e.g., "specialist.created")
- Side effects should be decoupled from the main use case (e.g., validate license, send email)
- Processing should be async (Kafka consumer, message queue)

### Event Listener Structure

The `inbound/` adapter folder is named after the message broker being used (e.g. `kafka/`, `rabbitmq/`, `sqs/`). The broker is a project-level choice — the listener/handler logic remains the same regardless of which broker is used.

```
event_listeners/<listener-name>/
├── listener/                              # Mirrors application/ pattern
│   ├── handler.go                         # Handler logic (Execute/Handle method)
│   ├── handler_test.go                    # Unit tests
│   ├── new_handler.go                     # Constructor
│   ├── interface.go                       # Port interfaces
│   ├── dto.go                             # Listener-specific DTOs
│   ├── constants.go                       # Span names, event names
│   └── mocks/                             # gomock-generated mocks
└── adapters/
    ├── inbound/
    │   └── <broker>/                      # e.g. kafka/, rabbitmq/, sqs/
    │       └── manager.go                 # Consumer/subscription management
    └── outbound/
        ├── database/
        │   ├── repository.go
        │   ├── new.go
        │   └── errors.go
        └── external/                      # External API gateway (optional)
            ├── gateway.go
            └── new.go
```

### Listener Handler Template

```go
// listener/handler.go
package listener

import (
    "context"
    "fmt"
)

func (h *Handler) Handle(ctx context.Context, input HandleInput) error {
    ctx, span := h.tracer.Start(ctx, SpanNameHandleEvent)
    defer span.End()

    h.logger.Info(ctx, "handling event", "entity_id", input.EntityID)

    // 1. Call external service (gateway)
    result, err := h.gateway.Validate(ctx, input.License)
    if err != nil {
        return fmt.Errorf("%s: %w", ErrMsgValidationFailed, err)
    }

    // 2. Update entity status based on result
    if err := h.repo.UpdateStatus(ctx, input.EntityID, result.Status); err != nil {
        return fmt.Errorf("%s: %w", ErrMsgUpdateFailed, err)
    }

    // 3. Dispatch follow-up event
    _ = h.dispatcher.Dispatch(ctx, event.Event{
        Name:    EventLicenseValidated,
        Payload: result,
    })

    return nil
}
```

### Kafka Consumer Manager Template

> Note: This template uses Kafka as an example. Replace with your project's message broker (RabbitMQ, SQS, NATS, etc.). The `listener/` handler logic remains identical regardless of broker — only the `adapters/inbound/<broker>/` implementation changes.

```go
// adapters/inbound/<broker>/manager.go (example: kafka)
package kafka

import (
    "context"
    "encoding/json"

    "myapp/internal/modules/<module>/features/<feature>/event_listeners/<listener>/listener"
)

type ConsumerManager struct {
    handler *listener.Handler
    logger  observability.Logger
}

func (m *ConsumerManager) HandleMessage(ctx context.Context, msg []byte) error {
    var input listener.HandleInput
    if err := json.Unmarshal(msg, &input); err != nil {
        m.logger.Error(ctx, "failed to unmarshal message", "error", err)
        return err
    }
    return m.handler.Handle(ctx, input)
}
```

### Placeholder Listeners

Not every listener needs to be implemented immediately. Create empty folders as placeholders:

```
event_listeners/
├── validate_license/    # Fully implemented
└── send_credentials_email/  # Empty — placeholder for future implementation
```

---

## File Naming Conventions Summary

| Layer | Logic file | Constructor file | DI file | Notes |
|---|---|---|---|---|
| application/ | `usecase.go` | `new_usecase.go` | — | Logic and constructor always separated |
| listener/ | `handler.go` | `new_handler.go` | — | Same pattern as application |
| adapters/outbound/database/ | `repository.go` | `new.go` | — | e.g. PostgreSQL, MySQL — swappable |
| adapters/outbound/<search-engine>/ | `repository.go` | `new.go` | — | e.g. Elasticsearch, OpenSearch — + `builders.go`, `mappers.go`, `dto.go` |
| adapters/outbound/external/ | `gateway.go` | `new.go` | — | External API calls |
| adapters/inbound/grpc_service/ | `service.go` | — | `di.go` | DI separated from service |
| adapters/inbound/http_handler/ | `handler.go` | — | `di.go` | DI separated from handler |

Files present across multiple layers:
- `dto.go` — application, listener, all inbound adapters, search-engine outbound adapter
- `constants.go` — application and listener (span names, event names, error messages)
- `errors.go` — domain, outbound adapters (database, search-engine)
- `interface.go` — application and listener (dependency port contracts)
- `mocks/` — application, listener, all inbound adapters (gomock-generated)

---

## Feature Composition Matrix

When creating a new feature, use this template to determine which components apply:

```yaml
<feature-name>:
  event_listeners: yes/no (list listener names and status: implemented/placeholder)
  adapters/outbound/database: yes/no (e.g. PostgreSQL, MySQL — swappable)
  adapters/outbound/elasticsearch: yes/no (e.g. Elasticsearch, OpenSearch — with builders, mappers, dto if yes)
  adapters/inbound/http_handler: yes/no
  adapters/inbound/grpc_service: yes/no
  handler_integration_test: yes/no (at adapters/inbound/ root)
  application/mocks: count and list (event_dispatcher, logger, repository, tracer — only what's needed)
  domain sub-packages: list feature-specific domain packages and their contents
```

Not every feature needs all components. Use only what the feature requires — avoid creating empty boilerplate.

## Post-Generation Checklist

- [ ] Application layer depends ONLY on interfaces defined in its own `interface.go`
- [ ] Handlers (HTTP/gRPC) only call use case methods — no business logic
- [ ] Repositories implement interfaces defined in application/ — not the reverse
- [ ] Constructor and logic are in separate files
- [ ] Mocks live in `mocks/` sub-folders
- [ ] DTOs exist per layer (no shared DTOs across layers)
- [ ] Event listeners are independent mini-modules with their own `listener/ + adapters/`
- [ ] `go build ./...` passes
- [ ] `golangci-lint run ./...` passes
- [ ] `go test ./...` passes
