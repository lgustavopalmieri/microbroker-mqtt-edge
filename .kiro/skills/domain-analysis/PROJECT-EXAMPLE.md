# EXAMPLE OF Project Structure & Organization

## Root Layout

```
.
├── cmd/
│   ├── main.go                        # Application entrypoint
│   ├── server/                        # Main server (HTTP + gRPC + Kafka)
│   │   ├── .env
│   │   ├── bootstrap/                 # Initialization orchestration
│   │   │   ├── database.go
│   │   │   ├── elasticsearch.go
│   │   │   ├── grpc_services.go
│   │   │   ├── http_services.go
│   │   │   ├── kafka.go
│   │   │   ├── kafka_consumers.go
│   │   │   ├── observability.go
│   │   │   └── shutdown.go
│   │   └── config/
│   │       ├── config.go
│   │       ├── helpers.go
│   │       ├── load.go
│   │       └── validate.go
│   └── grpcserver/                    # Standalone gRPC server (empty, placeholder)
│       ├── bootstrap/
│       └── config/
├── internal/
│   ├── commom/                        # Shared cross-module utilities (note: "commom" typo is intentional)
│   ├── modules/                       # Business domain modules
│   └── platform/                      # Infrastructure adapters
├── tests/
│   └── k6/                            # Load/stress tests (k6 framework)
├── docker-compose.yml
├── Dockerfile
├── Makefile
├── go.mod / go.sum
└── .env / .env.example
```

## internal/commom/ — Shared Utilities

```
commom/
├── event/
│   ├── dipstacher.go                  # Event dispatcher implementation
│   ├── event.go                       # Event interface/struct
│   └── listener.go                    # Listener interface
├── observability/
│   ├── logging.go                     # Logger interface
│   ├── metrics.go                     # Metrics interface
│   ├── tracing.go                     # Tracer interface
│   └── mocks/
│       └── logger_mock.go             # Shared logger mock (gomock)
├── tests/                             # Test infrastructure helpers
│   ├── database/postgresql/           # Testcontainers PostgreSQL setup
│   ├── elasticsearch/                 # Testcontainers Elasticsearch setup + factory
│   └── event/kafka/                   # Testcontainers Kafka setup
├── utils/
│   ├── sanitize.go
│   └── sanitize_test.go
└── value-objects/
    └── pagination/cursor/             # Cursor-based pagination value object
        ├── input.go / output.go       # Input/Output structs
        ├── encode.go / decode.go      # Cursor encoding/decoding
        ├── validate_input.go          # Input validation
        ├── errors.go / utils.go
        ├── example_usage.go
        ├── encode_test.go / decode_test.go
        └── README.md
```

## internal/modules/specialist/ — Domain Module

### Domain Layer

```
domain/
├── entity.go                          # Specialist entity struct (package: domain)
├── errors.go                          # Shared domain errors
├── status.go                          # Status enum/constants (pending, active, etc.)
├── validate.go                        # Shared validation logic
├── validate_test.go
├── create/                            # Create domain logic (package: create)
│   ├── create.go                      # Factory function + CreateSpecialistInput
│   ├── errors.go                      # Create-specific errors
│   ├── uniqueness_errors.go           # Uniqueness constraint errors (email, license)
│   └── create_test.go
├── authorize_license/                 # License authorization domain logic
│   ├── authorize_license.go
│   ├── errors.go
│   └── authorize_license_test.go
├── update/                            # Update domain logic
│   ├── update.go
│   ├── validators.go
│   └── errors.go
└── search/                            # Search domain logic
    ├── errors.go
    ├── search_input/                  # Search input value object (sub-package)
    │   ├── input.go
    │   ├── field.go                   # Searchable field definitions
    │   ├── sort.go                    # Sort options
    │   ├── validate_input.go
    │   └── validate_input_test.go
    └── search_output/                 # Search output value object (sub-package)
        ├── output.go
        └── validate_output.go
```

Domain layer specifics:
- `domain/` (package `domain`) contains entity, status, errors, and shared validations
- Each sub-domain (`create/`, `update/`, `authorize_license/`) is a separate package that imports `domain`
- `search/` has extra depth: sub-packages `search_input/` and `search_output/` as value objects
- `create/` has `uniqueness_errors.go` separate from generic `errors.go` — pattern for constraint errors

### Features Layer

Each feature follows the `application/ + adapters/` structure (Clean Architecture / Hexagonal), with variations per feature.

The `adapters/` directory replaces the former `infra/` and is split into:
- `adapters/inbound/` — driving adapters (HTTP handlers, gRPC services)
- `adapters/outbound/` — driven adapters (database repositories, Elasticsearch repositories)

Note: `event_listeners/` inside features also use the `adapters/inbound/outbound` structure, matching the feature-level convention.

#### Feature: create

```
features/create/
├── application/
│   ├── usecase.go / usecase_test.go
│   ├── new_usecase.go
│   ├── interface.go
│   ├── dto.go
│   ├── constants.go
│   └── mocks/                         # 4 mocks: event_dispatcher, logger, repository, tracer
├── event_listeners/                   # Event-driven side effects
│   ├── send_credentials_email/        # (empty — placeholder)
│   └── validate_license/              # Kafka consumer listener (fully implemented)
│       ├── listener/                  # Handler logic (mirrors application/ pattern)
│       │   ├── handler.go / handler_test.go
│       │   ├── new_handler.go
│       │   ├── interface.go
│       │   ├── dto.go
│       │   ├── constants.go
│       │   └── mocks/                 # 4 mocks: event_dispatcher, logger, repository, tracer
│       └── adapters/
│           ├── inbound/
│           │   └── kafka/             # manager.go (consumer group management)
│           └── outbound/
│               ├── database/          # repository.go, new.go, errors.go, repository_test.go
│               └── external/          # gateway.go, new.go
└── adapters/
    ├── inbound/
    │   ├── http_handler/              # handler.go, handler_test.go, dto.go, di.go, mocks/
    │   ├── grpc_service/              # service.go, service_test.go, dto.go, di.go, mocks/, pb/, proto/
    │   └── handler_integration_test.go  # Integration test (at inbound/ root, not inside a sub-folder)
    └── outbound/
        └── database/                  # repository.go, new.go, errors.go, repository_test.go
```

Create specifics:
- Only feature with `event_listeners/` containing fully implemented Kafka listeners
- `validate_license/` is a mini-module with its own `listener/` + `adapters/` (inbound/kafka, outbound/database, outbound/external)
- `send_credentials_email/` exists as an empty placeholder
- `handler_integration_test.go` lives at the `adapters/inbound/` root, not inside `database/` or `http_handler/`
- `adapters/inbound/` has both `http_handler/` and `grpc_service/` (dual transport)

#### Feature: search

```
features/search/
├── application/
│   ├── usecase.go / usecase_test.go
│   ├── new_usecase.go
│   ├── interface.go
│   ├── dto.go
│   ├── constants.go
│   └── mocks/                         # 1 mock: repository_mock (no logger/tracer mocks)
└── adapters/
    ├── inbound/
    │   ├── http_handler/              # handler.go, handler_test.go, dto.go, di.go, mocks/
    │   └── grpc_service/              # service.go, service_test.go, dto.go, di.go, mocks/, pb/, proto/
    └── outbound/
        └── elasticsearch/             # Elasticsearch repository (does not use database/)
            ├── repository.go / repository_test.go
            ├── new.go / errors.go
            ├── builders.go            # Query builders
            ├── mappers.go             # Response mappers
            ├── dto.go                 # ES-specific DTOs
            └── README.md
```

Search specifics:
- No `event_listeners/`
- `adapters/outbound/` uses `elasticsearch/` instead of `database/` — read repository
- `elasticsearch/` has extra files: `builders.go`, `mappers.go` (query construction + response mapping)
- `application/mocks/` has only `repository_mock.go` (fewer dependencies than create/update)

#### Feature: update

```
features/update/
├── application/
│   ├── usecase.go / usecase_test.go
│   ├── new_usecase.go
│   ├── interface.go
│   ├── dto.go
│   ├── constants.go
│   └── mocks/                         # 4 mocks: event_dispatcher, logger, repository, tracer
├── event_listeners/
│   ├── send_status_email/             # (empty — placeholder)
│   └── update_data_repositories/      # Sync data to read stores
│       ├── command/                   # (empty — placeholder)
│       └── repositories/
│           └── elasticsearch/         # errors.go, new.go, repository.go
└── adapters/
    ├── inbound/
    │   ├── http_handler/              # handler.go, handler_test.go, dto.go, di.go, mocks/
    │   └── grpc_service/              # service.go, service_test.go, dto.go, di.go, mocks/, pb/, proto/
    └── outbound/
        └── database/                  # repository.go, new.go, errors.go, repository_test.go
```

Update specifics:
- `event_listeners/` has a different structure from create: `update_data_repositories/` uses `repositories/` (not `infra/`)
- `update_data_repositories/command/` is empty (placeholder)
- `send_status_email/` is empty (placeholder)
- No `handler_integration_test.go` at the `adapters/inbound/` root

## internal/platform/ — Infrastructure Adapters

```
platform/
├── database/postgresql/
│   ├── connection.go                  # Connection pool setup
│   ├── migrations.go                  # Migration runner
│   └── migrations/                    # SQL migration files (ordered: 001_, 002_, 003_)
├── elasticsearch/
│   ├── client.go                      # ES client setup
│   └── indexes/
│       ├── registry.go                # Index registry
│       └── specialists.go             # Specialist index mapping
├── kafka/
│   ├── producer.go
│   └── consumer.go
├── server/
│   ├── grpcserver.go
│   ├── httpserver.go
│   └── metrics_server.go             # Prometheus metrics endpoint
└── telemetry/
    ├── otel.go                        # OpenTelemetry bootstrap
    ├── tracing_provider.go
    ├── prometheus.go
    ├── slog.go                        # Structured logging setup
    ├── grpc_metrics.go                # gRPC interceptors
    └── provider.go                    # Telemetry provider aggregate
```

## tests/k6/ — Load & Stress Tests

```
tests/k6/
├── Makefile
├── docker-compose.k6.yml
├── commom/
│   ├── factories/specialist.js        # Test data factory
│   ├── grpc-client.js                 # gRPC client helper
│   ├── http-client.js                 # HTTP client helper
│   ├── http-stress-runner.js          # HTTP stress test runner
│   └── stress-runner.js              # gRPC stress test runner
└── modules/specialist/create/
    ├── stress/                        # gRPC stress tests
    │   ├── config.js
    │   ├── create-specialist-test.js
    │   ├── validations.js
    │   └── proto/specialist.proto     # Proto file duplicated for k6
    └── stress-http/                   # HTTP stress tests
        ├── config.js
        ├── create-specialist-test.js
        └── validations.js
```

## Structural Variations Between Features

```yaml
create:
  event_listeners: yes (validate_license fully implemented, send_credentials_email empty)
  adapters/outbound/database: yes
  adapters/outbound/elasticsearch: no
  adapters/inbound/http_handler: yes
  adapters/inbound/grpc_service: yes
  handler_integration_test: yes (at adapters/inbound/ root)
  application/mocks: 4 (event_dispatcher, logger, repository, tracer)
  domain sub-packages: create/ (with separate uniqueness_errors)

search:
  event_listeners: no
  adapters/outbound/database: no
  adapters/outbound/elasticsearch: yes (with builders, mappers, own dto)
  adapters/inbound/http_handler: yes
  adapters/inbound/grpc_service: yes
  handler_integration_test: no
  application/mocks: 1 (repository)
  domain sub-packages: search_input/ + search_output/ (value objects as sub-packages)

update:
  event_listeners: yes (update_data_repositories partial, send_status_email empty)
  adapters/outbound/database: yes
  adapters/outbound/elasticsearch: no
  adapters/inbound/http_handler: yes
  adapters/inbound/grpc_service: yes
  handler_integration_test: no
  application/mocks: 4 (event_dispatcher, logger, repository, tracer)
  domain sub-packages: update/ (with validators separate from update.go)
```

## File Naming Patterns by Layer

- `usecase.go` + `new_usecase.go` — application layer (logic and constructor separated)
- `handler.go` + `new_handler.go` — listener layer (same pattern)
- `repository.go` + `new.go` — adapters/outbound/database and adapters/outbound/elasticsearch
- `gateway.go` + `new.go` — event_listeners/*/adapters/outbound/external
- `service.go` + `di.go` — adapters/inbound/grpc_service (DI separated from service)
- `handler.go` + `di.go` — adapters/inbound/http_handler
- `dto.go` — present in all layers (application, listener, adapters/inbound/grpc_service, adapters/inbound/http_handler, adapters/outbound/elasticsearch)
- `constants.go` — application and listener (span names, event names, error messages)
- `errors.go` — domain, application (via constants.go), adapters/outbound/database, adapters/outbound/elasticsearch
- `interface.go` — application and listener (dependency contracts)

## Package Dependencies Flow

```
domain (entity, status, errors, validate)
    ↑
domain/<feature>/ (factory, validation, feature-specific errors)
    ↑
features/<feature>/application/ (use case orchestration, interfaces, DTOs)
    ↑
features/<feature>/adapters/inbound/ (grpc_service, http_handler)
features/<feature>/adapters/outbound/ (database, elasticsearch)

features/<feature>/event_listeners/<listener>/listener/ (handler, interfaces, DTOs)
    ↑
features/<feature>/event_listeners/<listener>/adapters/ (inbound/kafka, outbound/database, outbound/external)
```

Event listeners are independent mini-modules within a feature, with their own listener/ + adapters/ separation.