---
name: verify-go
description: Run the full pre-commit verification gate for this Go repo — build, vet, lint, and race-enabled tests. Use before committing or opening a PR, or to confirm the tree is green after a change.
---

# verify-go

This is the project's "before committing" gate from AGENTS.md. Run each step from the repo root and report results. Stop and fix on the first failure rather than pushing through.

Run, in order:

```bash
go build ./...
go vet ./...
golangci-lint run ./...
go test -race ./...
```

Optionally add `go test -cover ./...` if the user wants a coverage read.

- Report which steps passed and paste the actual failure output for any that didn't — never claim green without the test output.
- If a step fails, fix the cause (and add/adjust table-driven tests for code you changed), then re-run the whole gate.

> Note: this is additive to the built-in `/verify` skill, which instead runs the app and observes behavior. Use `/verify` for end-to-end behavioral checks; use `/verify-go` for the build/lint/test gate.
