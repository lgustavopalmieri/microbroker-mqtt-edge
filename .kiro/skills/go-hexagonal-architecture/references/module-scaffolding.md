# Module Scaffolding & Evaluation Reference (Go)

---

# Part 1: Module Creation

## Requirements Gathering

Before generating code, gather:
1. **Module name** (lowercase, e.g., "specialist", "inventory", "billing")
2. **Root entity** (the aggregate root, e.g., "Specialist", "Order", "Invoice")
3. **Initial features** (comma-separated, e.g., "create, search, update")
4. **Status enum** (if applicable, e.g., "pending, active, suspended, inactive")
5. **External integrations** (any third-party services or other modules?)
6. **Event-driven side effects** (will features emit domain events that trigger listeners?)
7. **Transport protocols** (HTTP only, gRPC only, or both?)
8. **Outbound storage** (e.g. PostgreSQL, MySQL, Elasticsearch, OpenSearch — these are swappable at any time thanks to the interface-based adapter pattern)

## Module Structure

Every module lives under `internal/modules/<module-name>/` and has two top-level concerns:

```
internal/modules/<module-name>/
├── domain/                                # Domain layer (entity, status, errors, validation)
│   ├── entity.go                          # Root entity struct (package: domain)
│   ├── errors.go                          # Shared domain errors
│   ├── status.go                          # Status enum/constants (if applicable)
│   ├── validate.go                        # Shared validation logic
│   ├── validate_test.go
│   └── <feature-name>/                    # Feature-specific domain logic (separate Go package)
│       ├── <feature-name>.go              # Factory function + input struct
│       ├── errors.go                      # Feature-specific errors
│       └── <feature-name>_test.go
└── features/                              # Features layer
    └── <feature-name>/                    # One folder per feature (see feature-scaffolding.md)
        ├── application/
        ├── adapters/
        └── event_listeners/               # Optional
```

## Domain Layer Generation

### entity.go

```go
package domain

import "time"

type <Entity> struct {
    ID        string
    Name      string
    Email     string
    Status    Status
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

### status.go

```go
package domain

type Status string

const (
    StatusPending   Status = "pending"
    StatusActive    Status = "active"
    StatusSuspended Status = "suspended"
    StatusInactive  Status = "inactive"
)

func (s Status) IsValid() bool {
    switch s {
    case StatusPending, StatusActive, StatusSuspended, StatusInactive:
        return true
    }
    return false
}
```

### errors.go

```go
package domain

import "errors"

var (
    ErrNotFound       = errors.New("entity not found")
    ErrInvalidStatus  = errors.New("invalid status")
    ErrAlreadyExists  = errors.New("entity already exists")
)
```

### validate.go

```go
package domain

import "fmt"

func ValidateEmail(email string) error {
    if email == "" {
        return fmt.Errorf("email is required")
    }
    // ... validation logic
    return nil
}
```

### Feature-Specific Domain Sub-Package

Each feature that has domain logic gets its own Go package under `domain/`:

```go
// domain/create/create.go
package create

import (
    "fmt"
    "myapp/internal/modules/<module>/domain"
)

type Create<Entity>Input struct {
    Name    string
    Email   string
    License string
}

func NewEntity(input Create<Entity>Input) (*domain.<Entity>, error) {
    if err := validate(input); err != nil {
        return nil, err
    }
    return &domain.<Entity>{
        ID:     generateID(),
        Name:   input.Name,
        Email:  input.Email,
        Status: domain.StatusPending,
    }, nil
}
```

```go
// domain/create/errors.go
package create

import "errors"

var (
    ErrNameRequired    = errors.New("name is required")
    ErrEmailRequired   = errors.New("email is required")
    ErrLicenseRequired = errors.New("license is required")
)
```

For features with complex I/O (e.g., search), use sub-packages:

```
domain/search/
├── errors.go
├── search_input/
│   ├── input.go
│   ├── field.go           # Searchable field definitions
│   ├── sort.go            # Sort options
│   ├── validate_input.go
│   └── validate_input_test.go
└── search_output/
    ├── output.go
    └── validate_output.go
```

### Uniqueness Constraint Errors

When a feature has uniqueness constraints (e.g., unique email, unique license), create a separate file:

```go
// domain/create/uniqueness_errors.go
package create

import "errors"

var (
    ErrEmailAlreadyExists   = errors.New("email already exists")
    ErrLicenseAlreadyExists = errors.New("license already exists")
)
```

## Generation Order

1. Gather requirements
2. Create `domain/entity.go` — root entity struct
3. Create `domain/status.go` — status enum (if applicable)
4. Create `domain/errors.go` — shared domain errors
5. Create `domain/validate.go` + `validate_test.go` — shared validation
6. For each feature, create `domain/<feature>/` sub-package
7. For each feature, follow `references/feature-scaffolding.md`
8. Run verification commands from `references/verification.md`

## Post-Generation Checklist

- [ ] Domain layer has zero imports from `adapters/`, `platform/`, or external infrastructure
- [ ] Each feature-specific domain sub-package is a separate Go package
- [ ] Entity struct lives in the root `domain/` package
- [ ] Status constants are defined in `domain/status.go`
- [ ] Shared errors are in `domain/errors.go`; feature-specific errors in `domain/<feature>/errors.go`
- [ ] Uniqueness constraint errors have their own file when distinct from generic errors
- [ ] All domain logic has unit tests
- [ ] `go build ./...` passes
- [ ] `golangci-lint run ./...` passes
- [ ] `go test ./...` passes

---

# Part 2: Module Evaluation

## When to Evaluate

Use this when a module is growing and may need restructuring or splitting.

## Evaluation Process

### Step 1: Gather Module Information

- List all features in `features/`
- List all domain sub-packages in `domain/`
- List all entities and their relationships
- Identify external dependencies (gateways, external APIs)
- Check event listeners and their complexity

### Step 2: Identify Sub-Domain Signals

Look for natural groupings based on:

| Signal | Description |
|--------|-------------|
| Different User Personas | Admin vs customer; internal vs external |
| Different Execution Models | Sync REST vs async message consumers vs real-time events |
| Different Technical Characteristics | Read-heavy vs write-heavy; CPU-bound vs I/O-bound |
| Different Change Velocities | Experimental vs stable; different team ownership |
| Independent Deployment Potential | Could this be a separate microservice? |

### Step 3: Measure Cohesion and Coupling

**Cohesion Score (1-5, higher is better):**
- 5: Single, clear responsibility — all features serve same entity/aggregate
- 4: Related responsibilities — changes affect same domain concepts
- 3: Some overlap but features can work independently
- 2: Loosely related, serve different purposes
- 1: Unrelated, grouped arbitrarily

**Coupling Score (1-5, lower is better):**
- 1: Features never interact
- 2: Occasional communication via events
- 3: Regular communication through well-defined interfaces
- 4: Frequent direct calls, shared state
- 5: Tightly coupled, can't function independently

**Decision Matrix:**
```
High Cohesion (4-5) + Low Coupling (1-2)   → STRONG CANDIDATE for splitting into separate modules
High Cohesion (4-5) + High Coupling (4-5)  → KEEP TOGETHER
Low Cohesion (1-2)  + Any Coupling          → REFACTOR first, don't split
```

### Step 4: 6-Criteria Test

| # | Criterion | Question |
|---|-----------|----------|
| 1 | User Persona | Does this serve fundamentally different users? |
| 2 | Access Control | Does this need different authorization models? |
| 3 | Execution Model | Different protocols (REST vs gRPC vs message consumer)? |
| 4 | Scaling Needs | Different scaling characteristics? |
| 5 | Deployment | Could this be deployed independently? |
| 6 | Failure Isolation | Can this fail without affecting other parts? |

**Decision:**
- ✅ 4+ criteria met → STRONG recommendation for splitting
- ⚠️ 2-3 criteria met → CONSIDER splitting (evaluate trade-offs)
- ❌ 0-1 criteria met → KEEP as single module

## Red Flags: When NOT to Split

- ❌ "The module feels big" — size alone is not a reason
- ❌ "To make code easier to find" — folder organization problem, not domain problem
- ❌ "Features are tightly coupled" — high coupling means they belong together
- ❌ "To match team structure" — don't let org chart drive architecture

## Green Lights: When TO Split

- ✅ These serve different user types (admin vs customer)
- ✅ These have different failure modes (background can fail without affecting API)
- ✅ These scale differently (CPU-intensive vs simple CRUD)
- ✅ These could logically be separate microservices
- ✅ These have different change velocities

## Output Format for Evaluation

```markdown
# Module Evaluation: <module-name>

## Current Structure
- Features: {count}, Domain sub-packages: {count}, Event listeners: {count}

## Identified Groupings
### Group 1: {name}
- Cohesion Score: {1-5}
- User Persona: {who uses this}
- Execution Model: {sync/async}

## Coupling Analysis
- Between Group 1 and Group 2: Coupling Score {1-5}

## 6-Criteria Test
| Criterion | Met? |
|-----------|------|
| User Persona | ✅/❌ |
| Access Control | ✅/❌ |
| Execution Model | ✅/❌ |
| Scaling Needs | ✅/❌ |
| Deployment | ✅/❌ |
| Failure Isolation | ✅/❌ |

Total: {X}/6

## Recommendation: [Split / Keep Together]
**Rationale**: {explanation}
**Proposed Structure** (if splitting): ...
**Trade-offs**: ...
```
