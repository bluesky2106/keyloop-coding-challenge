---
name: go-backend
description: Conventions for the Go API in backend/. Use when writing or reviewing Go code, package layout, error handling, HTTP handlers, or Go tests.
---

# Go backend conventions

## Layout
- Organise by feature, not by layer: `internal/<feature>/` owns its handler, service and queries. No global `controllers/`, `services/`, `repositories/` folders.
- `cmd/api/main.go` only wires config, DB pool, routers and graceful shutdown.
- `internal/platform/` holds shared infrastructure (db, http server, logging). Features never import each other's internals; share via small interfaces defined by the consumer.

## Code
- Router: `chi`. DB: `pgx/v5` with `pgxpool`. Migrations: `golang-migrate`. Logging: stdlib `log/slog`.
- Pass `context.Context` as the first parameter everywhere; use it for every DB call.
- Wrap errors with `fmt.Errorf("doing x: %w", err)`; compare with `errors.Is/As`. Map domain errors to HTTP status in one place.
- Handlers: decode, validate, call service, encode. No SQL or business rules in handlers.
- Keep interfaces small and define them where they are used; do not create an interface for a single implementation without a test need.
- Config comes from environment variables, validated at startup; fail fast.

## Testing
- Backend statement coverage must be at least 90% (`make test` enforces it via `backend/scripts/check-coverage.sh`). Do not lower `COVERAGE_MIN`; add tests.
- Layer as handler -> service -> store. Each layer depends on a small interface defined where it is used, so a layer is unit-tested with a mock of the layer below (error paths such as a database failure are only reachable this way).
- Mocks are generated with mockery (`make mocks`, config in `backend/.mockery.yml`) into `mocks_test.go` beside the interface. Add new interfaces to the config; never hand-edit generated files. A mock cannot match a variadic call whose argument count varies (pgx `Exec`); use a small fake for that.
- SQL is tested against a real Postgres via `testdb.New(t)` (throwaway database, real migrations), never a mocked `pgx`. Read-only tests in one package may share a database built in `TestMain`.
- Use `testify` (`require` to stop, `assert` to continue). Table-driven tests with `t.Run`.
- Aging boundary must be tested: exactly 90 days is not aging, 91 is.
- Tests must assert the reason for a failure (error code, status, message), not merely that something failed.

## Verify before done
`make lint test` for the backend must pass (`golangci-lint run`, `go test ./... -race`).
