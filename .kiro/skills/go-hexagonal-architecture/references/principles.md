# Hexagonal Architecture Principles Reference (Go)

All 8 principles with rules and Go code examples.

---

## Principle 1: Well-Defined Boundaries

Each module has clear responsibilities and doesn't expose internal details to other modules.

**Rules:**
- ✅ Keep all domain logic within module boundaries
- ✅ Each module is a separate Go package tree under `internal/modules/<module>/`
- ✅ Inter-module communication happens via events, gRPC, or HTTP — never direct imports
- ❌ Never import another module's domain, application, or adapter packages
- ❌ Never share database entities between modules

```go
// ✅ GOOD: Module A communicates with Module B via an external gateway
// features/create/event_listeners/validate_license/adapters/outbound/external/gateway.go
type LicenseGateway struct {
    client *http.Client
    baseURL string
}

func (g *LicenseGateway) ValidateLicense(ctx context.Context, licenseID string) (bool, error) {
    // HTTP call to another module's API — explicit boundary
}

// ❌ BAD: Direct import of another module's domain
import "myapp/internal/modules/license/domain"
```

---

## Principle 2: Composability

Modules are building blocks that combine flexibly to create different deployable servers.

**Rules:**
- ✅ Design modules to work independently or together
- ✅ Multiple servers in `cmd/` can compose different module combinations
- ✅ Use interfaces for loose coupling between layers
- ❌ Never create tight coupling between modules

```go
// ✅ GOOD: cmd/server/ composes all modules; cmd/grpcserver/ composes only gRPC-relevant ones
// cmd/server/bootstrap/http_services.go
func RegisterHTTPServices(router *mux.Router, db *sql.DB) {
    createHandler := createdi.NewHTTPHandler(db)
    searchHandler := searchdi.NewHTTPHandler(db)
    router.Handle("/specialists", createHandler)
    router.Handle("/specialists/search", searchHandler)
}

// cmd/grpcserver/bootstrap/grpc_services.go — only gRPC features
func RegisterGRPCServices(server *grpc.Server, db *sql.DB) {
    createService := createdi.NewGRPCService(db)
    pb.RegisterSpecialistServiceServer(server, createService)
}
```

---

## Principle 3: Independence

Modules operate autonomously without tight coupling in code or infrastructure.

**Rules:**
- ✅ Modules can be built, tested, and deployed independently
- ✅ Use interfaces and events for inter-module communication
- ✅ Each module's tests run in isolation (testcontainers for integration tests)
- ❌ Never create shared mutable state between modules
- ❌ Never use direct function calls between modules

```go
// ✅ GOOD: Use case depends on interface, not concrete repository
// features/create/application/interface.go
type Repository interface {
    Save(ctx context.Context, entity *domain.Specialist) error
    FindByEmail(ctx context.Context, email string) (*domain.Specialist, error)
}

type EventDispatcher interface {
    Dispatch(ctx context.Context, event event.Event) error
}

// ❌ BAD: Use case directly depends on concrete database package
import "myapp/internal/modules/specialist/features/create/adapters/outbound/database"
```

---

## Principle 4: Explicit Communication

All inter-module communication happens through well-defined contracts.

**Rules:**
- ✅ Define clear interfaces for all module interactions
- ✅ Use DTOs for data transfer between layers
- ✅ Use typed event payloads for async communication
- ❌ Never access other modules' internal data structures
- ❌ Never make assumptions about other modules' implementations

```go
// ✅ GOOD: Typed event payload for inter-module communication
// internal/common/event/event.go
type Event struct {
    Name      string
    Payload   interface{}
    OccurredAt time.Time
}

// features/create/application/constants.go
const (
    EventSpecialistCreated = "specialist.created"
    SpanNameCreateUseCase  = "CreateSpecialistUseCase.Execute"
)

// ✅ GOOD: Application-level DTO — not the domain entity
// features/create/application/dto.go
type CreateInput struct {
    Name    string
    Email   string
    License string
}

type CreateOutput struct {
    ID     string
    Status string
}
```

---

## Principle 5: Replaceability

Adapters can be substituted without affecting domain or application logic.

**Rules:**
- ✅ Application layer defines interfaces (ports); adapters implement them
- ✅ Swapping PostgreSQL for MySQL (or any other database) means changing only `adapters/outbound/database/`
- ✅ Swapping HTTP for gRPC means changing only `adapters/inbound/`
- ❌ Never let application logic depend on adapter-specific types (e.g., `sql.Row`, `elastic.SearchResult`, `kafka.Message`)

```go
// ✅ GOOD: Application defines the port
// features/search/application/interface.go
type SearchRepository interface {
    Search(ctx context.Context, input searchinput.Input) (*searchoutput.Output, error)
}

// Example: Elasticsearch adapter implements it (could be OpenSearch, Meilisearch, etc.)
// features/search/adapters/outbound/elasticsearch/repository.go
type ElasticsearchRepository struct {
    client *elasticsearch.Client
}

func (r *ElasticsearchRepository) Search(ctx context.Context, input searchinput.Input) (*searchoutput.Output, error) {
    query := r.buildQuery(input)  // Engine-specific logic stays in adapter
    // ...
}

// Could be replaced with any other search engine or database:
// features/search/adapters/outbound/database/repository.go
type PostgresSearchRepository struct {
    db *sql.DB
}

func (r *PostgresSearchRepository) Search(ctx context.Context, input searchinput.Input) (*searchoutput.Output, error) {
    // PostgreSQL-specific full-text search — same interface
}
```

---

## Principle 6: State Isolation ⚠️ CRITICAL

Each module owns and manages its own state without sharing databases with other modules.

**Rules:**
- ✅ Each module has its own database schema or dedicated tables
- ✅ Use events or APIs for cross-module data needs
- ✅ Replicate minimal data per module (string references, not foreign keys across modules)
- ❌ Never share database tables between modules
- ❌ Never access other modules' repositories
- ❌ Never use foreign keys across module boundaries

```go
// ✅ GOOD: Module owns its migration files (example: PostgreSQL — could be any SQL database)
// internal/platform/database/postgresql/migrations/001_create_specialists.sql
// CREATE TABLE specialists (
//     id UUID PRIMARY KEY,
//     name VARCHAR(255) NOT NULL,
//     email VARCHAR(255) UNIQUE NOT NULL,
//     license VARCHAR(100) UNIQUE NOT NULL,
//     status VARCHAR(50) NOT NULL DEFAULT 'pending'
// );

// ✅ GOOD: Repository scoped to its own module's tables
// features/create/adapters/outbound/database/repository.go
func (r *PostgresRepository) Save(ctx context.Context, entity *domain.Specialist) error {
    _, err := r.db.ExecContext(ctx,
        "INSERT INTO specialists (id, name, email, license, status) VALUES ($1, $2, $3, $4, $5)",
        entity.ID, entity.Name, entity.Email, entity.License, entity.Status,
    )
    return err
}

// ❌ BAD: Querying another module's table
// "SELECT * FROM users WHERE id = $1" — 'users' belongs to the identity module
```

---

## Principle 7: Observability

Each module provides individual visibility into its health, performance, and behavior.

**Rules:**
- ✅ Observability contracts are INTERFACES in `internal/common/observability/`
- ✅ Concrete implementations (e.g. slog, OpenTelemetry, Prometheus — swappable) live in `internal/platform/telemetry/`
- ✅ Application and listener layers inject logger/tracer/metrics interfaces
- ✅ Use span names from `constants.go` for consistent tracing
- ❌ Never import `log/slog` or `go.opentelemetry.io/otel` directly in domain or application layers

```go
// ✅ GOOD: Interface in common/
// internal/common/observability/logging.go
type Logger interface {
    Info(ctx context.Context, msg string, args ...any)
    Error(ctx context.Context, msg string, args ...any)
    Warn(ctx context.Context, msg string, args ...any)
}

// ✅ GOOD: Use case injects the interface
// features/create/application/usecase.go
type CreateUseCase struct {
    repo       Repository
    dispatcher EventDispatcher
    logger     observability.Logger
    tracer     observability.Tracer
}

func (uc *CreateUseCase) Execute(ctx context.Context, input CreateInput) (*CreateOutput, error) {
    ctx, span := uc.tracer.Start(ctx, SpanNameCreateUseCase)
    defer span.End()

    uc.logger.Info(ctx, "creating specialist", "email", input.Email)
    // ...
}

// ❌ BAD: Direct slog usage in application layer
import "log/slog"
slog.Info("creating specialist") // violates observability interface rule
```

---

## Principle 8: Fail Independence

Failures in one module don't cascade to other modules.

**Rules:**
- ✅ Use timeouts and context cancellation for all external calls
- ✅ Design graceful degradation for non-critical features
- ✅ Event listeners handle failures independently (retry, dead-letter)
- ✅ Each module's health can be checked independently
- ❌ Never let one module's failure bring down others
- ❌ Never create synchronous dependencies that cascade failures

```go
// ✅ GOOD: External gateway with timeout and error handling
// features/create/event_listeners/validate_license/adapters/outbound/external/gateway.go
func (g *LicenseGateway) Validate(ctx context.Context, license string) (bool, error) {
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    resp, err := g.client.Get(ctx, g.baseURL+"/validate/"+license)
    if err != nil {
        return false, fmt.Errorf("license validation failed: %w", err)
    }
    // ... handle response
}

// ✅ GOOD: Message broker consumer with error handling — doesn't crash the whole server
// (example: Kafka — could be RabbitMQ, SQS, NATS, etc.)
// features/create/event_listeners/validate_license/adapters/inbound/kafka/manager.go
func (m *ConsumerManager) handleMessage(ctx context.Context, msg *kafka.Message) {
    if err := m.handler.Handle(ctx, msg); err != nil {
        m.logger.Error(ctx, "failed to handle message", "error", err, "topic", msg.Topic)
        // Send to dead-letter queue or retry — don't panic
    }
}
```
