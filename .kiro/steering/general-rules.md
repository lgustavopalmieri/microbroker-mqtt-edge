---
inclusion: always
description: Code generation, setup or configuration steps, or library/API documentation.
---

# Context7 MCP

This means you should automatically use the Context7 MCP tools to resolve library id and get library docs without me having to explicitly ask.

# Writing implementation plans

Remember that you are writing high quality and maintainable code while avoiding overengineering. You must be pragmatic and follow the guidelines in the docs first before blindly following industry standards.

Implementation plans should always include build, lint and tests when necessary.
To build run `go build ./...`, to lint run `golangci-lint run ./...` and to test run `go test ./...`

# Implementation and Testing

IMPORTANT always include tests to cover important paths! You should always make sure that the plans include a test suite that covers the happy paths and edge cases. Your tests should be high quality and give confidence while covering most of the implementation. Use Go's standard `testing` package and table-driven tests when appropriate.

# When dealing with database entities and migrations

Never create migrations manually, always use the project's migration tool (e.g. `goose`, `migrate`, or `atlas`).
To run migrations use the corresponding CLI command (e.g. `goose up`, `migrate -path ./migrations -database $DB_URL up`).
