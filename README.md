# Intelligent Inventory Dashboard

Keyloop technical assessment, Scenario B. A multi-dealership dashboard that shows vehicle stock, flags aging stock (over 90 days) and lets managers log actions against aging vehicles.

**Stack:** Go (chi, pgx) · PostgreSQL · React + TypeScript (Vite)

## Documentation
- [System design](docs/system-design.md): scope, assumptions, architecture, decisions
- [Database design](docs/database-design.md): schema, indexes, migrations, seed data
- API spec (`docs/api/openapi.yaml`) and validation notes: added as the implementation lands
- [Assessment brief](docs/reference/keyloop_technical_assessment.md)

## Getting started
Requires Go 1.26+, Node 24+, Docker and [golangci-lint](https://golangci-lint.run) v2.

```sh
cp .env.example .env   # once
make up      # start Postgres
make check   # lint, type-check and test backend + frontend
make help    # all targets
```

## Layout
```
backend/    Go API (cmd/api, internal/<feature>, migrations, seed CSVs)
frontend/   React + TypeScript app
docs/       design documents, API spec, reference material
.claude/    project skills used when working with AI agents
```
