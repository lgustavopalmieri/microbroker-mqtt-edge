# Verification Reference (Go)

Detection commands, compliance checklists, and maturity assessment framework.

---

# Part 1: Detection Commands

Run these commands to detect violations. **Never skip — they reveal hidden issues.**

## Dependency Flow Violations (MOST CRITICAL — Run First)

```bash
# 1. Domain importing infrastructure packages (CRITICAL)
# Domain must NEVER import adapters, platform, or external infra packages
# (adapt the grep patterns below to match your project's actual infrastructure libraries)
rg "\".*adapters/|\".*platform/|\"database/sql|\"github.com/elastic|\"github.com/segmentio/kafka" internal/modules/*/domain/ --glob "*.go"

# 2. Application importing concrete adapters (CRITICAL)
# Application must depend ONLY on its own interfaces
rg "\".*adapters/" internal/modules/*/features/*/application/ --glob "*.go"

# 3. Cross-module domain imports (CRITICAL)
# Module A's domain must NOT import module B's domain
rg "\".*internal/modules/[^\"]*\"" internal/modules/*/domain/ --glob "*.go" | \
  awk -F: '{split($1,a,"/"); module=a[3]; if($0 ~ "modules/" && $0 !~ "modules/"module) print "❌ CROSS-MODULE: "$0}'

# 4. Direct observability imports in application layer
# Application must use interfaces, not concrete slog/otel
rg "\"log/slog\"|\"go.opentelemetry.io" internal/modules/*/features/*/application/ --glob "*.go" | grep -v "_test.go"
```

**Expected for all dependency flow checks**: Empty output (no violations)

## State Isolation (Principle 6)

```bash
# 1. Cross-module table access — look for SQL queries referencing other modules' tables
# This requires manual review, but you can search for suspicious patterns:
rg "FROM\s+\w+" internal/modules/*/features/*/adapters/outbound/database/ --glob "*.go" -o

# 2. Shared database connections across modules
# Each module should have its own connection or schema
# (adapt patterns to your database driver: pgx, mysql, mongo, etc.)
rg "sql.Open|pgx.Connect|pgxpool.New" internal/modules/ --glob "*.go"
```

## Handler Pattern Violations (Lean Handlers)

```bash
# 1. HTTP handlers with business logic indicators
# Handlers should only decode request, call use case, encode response
rg "\.Save\(|\.Update\(|\.Delete\(|\.FindBy" internal/modules/*/features/*/adapters/inbound/http_handler/ --glob "*.go" | grep -v "_test.go"

# 2. gRPC services with business logic indicators
rg "\.Save\(|\.Update\(|\.Delete\(|\.FindBy" internal/modules/*/features/*/adapters/inbound/grpc_service/ --glob "*.go" | grep -v "_test.go"

# 3. Handlers importing domain packages directly (should go through application DTOs)
rg "\".*domain/" internal/modules/*/features/*/adapters/inbound/ --glob "*.go" | grep -v "pb/" | grep -v "_test.go"
```

## Interface Compliance

```bash
# 1. Missing interface.go in application packages
find internal/modules/*/features/*/application -maxdepth 1 -type d | while read dir; do
  if [ ! -f "$dir/interface.go" ]; then echo "❌ Missing interface.go in $dir"; fi
done

# 2. Missing interface.go in listener packages
find internal/modules/*/features/*/event_listeners/*/listener -maxdepth 1 -type d 2>/dev/null | while read dir; do
  if [ ! -f "$dir/interface.go" ]; then echo "❌ Missing interface.go in $dir"; fi
done

# 3. Application packages without mocks/ directory
find internal/modules/*/features/*/application -maxdepth 1 -type d | while read dir; do
  if [ ! -d "$dir/mocks" ]; then echo "⚠️  Missing mocks/ in $dir"; fi
done
```

## File Naming Convention Compliance

```bash
# 1. Missing constructor separation (new_usecase.go)
find internal/modules/*/features/*/application -name "usecase.go" | while read f; do
  dir=$(dirname "$f")
  if [ ! -f "$dir/new_usecase.go" ]; then echo "⚠️  Missing new_usecase.go alongside $f"; fi
done

# 2. Missing constructor separation in listeners (new_handler.go)
find internal/modules/*/features/*/event_listeners/*/listener -name "handler.go" 2>/dev/null | while read f; do
  dir=$(dirname "$f")
  if [ ! -f "$dir/new_handler.go" ]; then echo "⚠️  Missing new_handler.go alongside $f"; fi
done

# 3. Missing di.go in inbound adapters
find internal/modules/*/features/*/adapters/inbound/http_handler -name "handler.go" | while read f; do
  dir=$(dirname "$f")
  if [ ! -f "$dir/di.go" ]; then echo "⚠️  Missing di.go alongside $f"; fi
done

# 4. Missing new.go in outbound adapters
find internal/modules/*/features/*/adapters/outbound/database -name "repository.go" | while read f; do
  dir=$(dirname "$f")
  if [ ! -f "$dir/new.go" ]; then echo "⚠️  Missing new.go alongside $f"; fi
done
```

## Observability Compliance (Principle 7)

```bash
# 1. Use cases without tracer usage
find internal/modules/*/features/*/application -name "usecase.go" -exec grep -L "tracer\|Tracer" {} \;

# 2. Use cases without logger usage
find internal/modules/*/features/*/application -name "usecase.go" -exec grep -L "logger\|Logger" {} \;

# 3. Missing constants.go (span names, event names)
find internal/modules/*/features/*/application -maxdepth 1 -type d | while read dir; do
  if [ ! -f "$dir/constants.go" ]; then echo "⚠️  Missing constants.go in $dir"; fi
done
```

## Build & Lint Verification

```bash
# Always run these after any structural change
go build ./...
go vet ./...
golangci-lint run ./...
go test ./...
go test -race ./...
```

---

# Part 2: Compliance Checklists

## New Feature Checklist

```
□ Domain sub-package created (if feature has domain logic)
□ Domain sub-package has zero infrastructure imports
□ application/interface.go defines all port interfaces
□ application/usecase.go contains only orchestration logic
□ application/new_usecase.go has constructor separated from logic
□ application/dto.go has feature-specific DTOs (not shared across layers)
□ application/constants.go has span names, event names, error messages
□ application/mocks/ has gomock-generated mocks for all interfaces
□ adapters/inbound/http_handler/ has handler.go, dto.go, di.go
□ adapters/inbound/grpc_service/ has service.go, dto.go, di.go, pb/, proto/
□ adapters/outbound/database/ has repository.go, new.go, errors.go
□ Handlers only call use case methods — no business logic
□ Repositories implement application interfaces
□ Event listeners (if any) are independent mini-modules
□ Unit tests cover happy paths and edge cases (table-driven)
□ Integration tests use testcontainers
□ go build ./... passes
□ golangci-lint run ./... passes
□ go test ./... passes
```

## New Module Checklist

```
□ domain/entity.go — root entity struct
□ domain/status.go — status enum (if applicable)
□ domain/errors.go — shared domain errors
□ domain/validate.go — shared validation + tests
□ domain/<feature>/ — feature-specific domain sub-packages
□ features/<feature>/ — one folder per feature
□ Each feature passes the New Feature Checklist above
□ Domain layer has zero infrastructure imports
□ No cross-module domain imports
□ go build ./... passes
□ golangci-lint run ./... passes
□ go test ./... passes
```

## Refactoring Checklist

```
□ Preserve existing module boundaries
□ Maintain existing interface contracts
□ Fat handler → move logic to use case, remove direct repo calls
□ Missing interfaces → add interface.go with port definitions
□ Concrete observability → replace slog/otel with interface injection
□ Mixed constructor/logic → separate into new_usecase.go + usecase.go
□ Mocks alongside source → move to mocks/ sub-folder
□ Shared DTOs → create per-layer DTOs
□ Run detection commands (still clean after refactor)
□ All tests pass
```

## Pre-Commit Checklist

```bash
# Run these before every commit
go build ./...
go vet ./...
golangci-lint run ./...
go test ./...

# Quick architecture check
rg "\".*adapters/" internal/modules/*/features/*/application/ --glob "*.go"
rg "\".*adapters/|\".*platform/" internal/modules/*/domain/ --glob "*.go"
```

---

# Part 3: Maturity Assessment Framework

## Assessment Process

1. Run all detection commands from Part 1
2. Score each principle (1-10)
3. Apply weighted scoring
4. Determine maturity level
5. Generate prioritized recommendations

## Scoring per Principle

- **10 — Excellent**: Full compliance, best practices followed
- **8-9 — Good**: Strong compliance, minor improvements
- **6-7 — Acceptable**: Partial compliance, some violations
- **4-5 — Needs Improvement**: Multiple violations, requires attention
- **1-3 — Critical**: Major violations, immediate action required

## Weighted Scoring Table

| Principle | Weight | Notes |
|-----------|--------|-------|
| 1. Well-Defined Boundaries | 1.0 | |
| 2. Composability | 0.8 | |
| 3. Independence | 1.0 | |
| 4. Explicit Communication | 1.0 | |
| 5. Replaceability | 0.8 | |
| 6. State Isolation ⚠️ | **1.5** | Highest weight — most critical |
| 7. Observability | 0.9 | |
| 8. Fail Independence | 0.9 | |

Total possible: 100 weighted points (each principle max score × weight, normalized to 100)

## Maturity Level Definitions

| Level | Score | Characteristics |
|-------|-------|-----------------|
| Immature | 0-40 | Critical violations, no clear boundaries, shared state, multiple P0 issues |
| Developing | 41-65 | Some boundaries defined, known critical issues, can deploy with risk |
| Mature | 66-85 | Strong boundaries, good compliance, safe independent deployment, mostly P2/P3 |
| Advanced | 86-100 | Excellent compliance, best practices throughout, zero critical violations |

## Report Template

```markdown
# Architecture Maturity Assessment Report

**Assessment Date**: [Date]
**Project**: [Project Name]

---

## Executive Summary

- **Overall Maturity Level**: [Immature/Developing/Mature/Advanced]
- **Critical Issues**: [Count] P0 violations
- **Compliance Score**: [X/100] weighted points
- **Key Strengths**: ...
- **Key Weaknesses**: ...

---

## Principle-by-Principle Assessment

### Principle 6: State Isolation ⚠️ CRITICAL

**Status**: [✅ Compliant / ⚠️ Partial / ❌ Violated]
**Score**: X/10

**Cross-Module Domain Imports Detection**:
```
[Output from detection command]
```

**Evidence**: [Specific violations with file paths]
**Recommendations**: [Actionable fixes]

[Repeat for all 8 principles]

---

## Detection Command Results

[Output from each command in Part 1]

---

## Maturity Scoring

| Principle | Score | Weight | Weighted Score |
|-----------|-------|--------|----------------|
| 1. Boundaries | X/10 | 1.0 | X |
| ... | | | |
| 6. State Isolation | X/10 | 1.5 | X |
| **TOTAL** | | | **X/100** |

---

## Recommendations by Priority

### P0 — Critical (Fix Immediately)
[Violations that block safe operation — dependency flow violations, cross-module imports]

### P1 — High (Fix This Sprint)
[Violations causing technical debt — missing interfaces, concrete observability]

### P2 — Medium (Next 2-3 Sprints)
[Improvements for better compliance — missing mocks, naming conventions]

### P3 — Low (Nice to Have)
[Enhancements when capacity allows — additional tests, documentation]
```

---

# Part 4: CI/CD Integration

## GitHub Actions Workflow

```yaml
# .github/workflows/architecture-check.yml
name: Architecture Verification
on: [push, pull_request]

jobs:
  verify:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version-file: 'go.mod'

      - name: Check dependency flow — domain importing infrastructure
        run: |
          # Adapt these patterns to match your project's actual infrastructure libraries
          VIOLATIONS=$(grep -r '".*adapters/\|".*platform/\|"database/sql\|"github.com/elastic\|"github.com/segmentio/kafka' internal/modules/*/domain/ --include="*.go" || true)
          if [ ! -z "$VIOLATIONS" ]; then
            echo "❌ Domain layer importing infrastructure packages:"
            echo "$VIOLATIONS"
            exit 1
          fi

      - name: Check dependency flow — application importing concrete adapters
        run: |
          VIOLATIONS=$(grep -r '".*adapters/' internal/modules/*/features/*/application/ --include="*.go" || true)
          if [ ! -z "$VIOLATIONS" ]; then
            echo "❌ Application layer importing concrete adapters:"
            echo "$VIOLATIONS"
            exit 1
          fi

      - name: Check direct observability in application
        run: |
          VIOLATIONS=$(grep -r '"log/slog"\|"go.opentelemetry.io' internal/modules/*/features/*/application/ --include="*.go" | grep -v "_test.go" || true)
          if [ ! -z "$VIOLATIONS" ]; then
            echo "⚠️  Direct observability imports in application layer:"
            echo "$VIOLATIONS"
          fi

      - name: Build
        run: go build ./...

      - name: Vet
        run: go vet ./...

      - name: Lint
        uses: golangci/golangci-lint-action@v6
        with:
          version: latest

      - name: Test
        run: go test ./...

      - name: Race detection
        run: go test -race ./...
```

## Makefile Targets

```makefile
.PHONY: arch-check lint test build

arch-check:
	@echo "🔍 Checking architecture compliance..."
	@VIOLATIONS=$$(grep -r '".*adapters/\|".*platform/' internal/modules/*/domain/ --include="*.go" 2>/dev/null); \
	if [ ! -z "$$VIOLATIONS" ]; then echo "❌ Domain importing infrastructure:"; echo "$$VIOLATIONS"; exit 1; fi
	@VIOLATIONS=$$(grep -r '".*adapters/' internal/modules/*/features/*/application/ --include="*.go" 2>/dev/null); \
	if [ ! -z "$$VIOLATIONS" ]; then echo "❌ Application importing adapters:"; echo "$$VIOLATIONS"; exit 1; fi
	@echo "✅ Architecture check passed"

lint:
	golangci-lint run ./...

test:
	go test ./...

build:
	go build ./...

verify: arch-check build lint test
	@echo "✅ All checks passed"
```
