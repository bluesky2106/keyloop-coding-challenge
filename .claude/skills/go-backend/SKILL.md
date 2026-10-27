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
- Table-driven tests with `t.Run`. Pure logic (aging rule, filter building) gets unit tests.
- Repository and handler tests run against a real Postgres (testcontainers or the compose DB), never a mocked `pgx`.
- Aging boundary must be tested: exactly 90 days is not aging, 91 is.

## Verify before done
`make lint test` for the backend must pass (`golangci-lint run`, `go test ./... -race`).
