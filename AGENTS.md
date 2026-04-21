# AGENTS.md

## Dev environment tips
- Use `go work use ./<module>` if working in a multi-module workspace, so tooling resolves dependencies correctly.
- Run `go mod tidy` after adding or removing dependencies to keep `go.mod` and `go.sum` clean.
- Use `go run ./cmd/<service>` to run a specific service entrypoint.
- Check the `module` directive in each `go.mod` to confirm the correct module path — don't rely on folder names alone.

## Testing instructions
- Find the CI plan in the `.github/workflows` folder.
- Run `go test ./...` from the project root to execute all tests.
- To focus on one test, use `go test -run "^TestFunctionName$" ./path/to/package`.
- Use `go test -v ./...` for verbose output when debugging failures.
- Use `go test -race ./...` to detect race conditions.
- Use `go test -cover ./...` to check coverage.
- Run `golangci-lint run ./...` to catch lint and static analysis issues.
- Fix any test, lint or vet errors until the whole suite is green.
- After moving files or changing imports, run `go vet ./...` and `golangci-lint run ./...` to be sure everything still passes.
- Add or update tests for the code you change, even if nobody asked. Prefer table-driven tests.

## PR instructions
- Title format: [<service/package>] <Title>
- Always run `golangci-lint run ./...` and `go test ./...` before committing.
