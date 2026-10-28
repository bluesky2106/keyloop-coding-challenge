# Intelligent Inventory Dashboard

Keyloop technical assessment, Scenario B. A multi-dealership dashboard that shows vehicle stock, flags aging stock (in stock for more than 90 days) and lets managers log an action against each aging vehicle.

**Stack:** Go (chi, pgx) · PostgreSQL 18 · React + TypeScript (Vite) · Docker

## Run it

Requires Docker with Compose.

```sh
cp .env.example .env   # once: demo settings, safe to use as they are
make up                # builds and starts the database, API, demo data and web app
```

Open <http://localhost:8080> and sign in with a demo user, for example `northside.manager` / `demo-northside-1`. Each of the three demo dealerships has users in `backend/seed/users.csv`, and each sees only its own 50 vehicles.

The API reference, where you can try every endpoint from the browser, is at <http://localhost:8080/docs> (see [Try the API](#try-the-api)).

```sh
make smoke             # optional: exercise the running app end to end
make down              # stop and remove the containers (the data is kept)
```

If port 8080 or 5432 is already taken on your machine, set `WEB_PORT` or `POSTGRES_PORT` in `.env`.

### One service at a time

| Command | Does |
|---|---|
| `make up` / `make down` | Everything: database, API, demo data, web |
| `make db-up` / `make db-down` | Only the database |
| `make api-up` / `make api-down` | Only the API (it needs the database) |
| `make web-up` / `make web-down` | Only the web app (it shows the API as unavailable until the API is up) |

### Developing on the host

Run the database in Docker and the apps on your machine, for fast feedback:

```sh
make db-up   # the database
make seed    # apply migrations and load the demo data (safe to repeat)
make run     # the API on :8080
make web     # the frontend dev server on http://localhost:5173, proxying /api to the API
make check   # lint, type-check and test everything; backend coverage must be >= 90%
make help    # every target
```

Requires Go 1.26+, Node 24+, [golangci-lint](https://golangci-lint.run) v2 and, to regenerate mocks, [mockery](https://vektra.github.io/mockery/) v3. `make check` needs the database (`make db-up`).

If you ran an earlier version of this repo, its Postgres data volume is incompatible with Postgres 18: run `docker compose down -v` once.

## Try the API

**In the browser:** open <http://localhost:8080/docs> (Swagger UI, started by `make up`).

1. Expand `POST /auth/login`, choose **Try it out**, enter a demo user, and **Execute**.
2. Copy the `access_token` from the response, choose **Authorize** at the top, paste it (without `Bearer`), and close the dialog.
3. Every other endpoint now sends the token. The token is not stored, so authorize again after reloading the page.

The page is served by the web container next to the API, so it needs the API to be running (`make up`) and does not exist when you run the frontend with `make web` on the host. The spec it shows is `docs/api/openapi.yaml`, copied into the image when it is built.

**From the command line:**

```sh
curl -s -X POST localhost:8080/api/v1/auth/login \
  -d '{"username":"northside.manager","password":"demo-northside-1"}'
curl -s "localhost:8080/api/v1/vehicles?aging=true&page_size=5" -H "Authorization: Bearer <access_token>"
curl -s -X POST localhost:8080/api/v1/vehicles/<vehicle_id>/actions -H "Authorization: Bearer <access_token>" \
  -d '{"action_type":"price_reduction_planned","note":"Drop 5% on Friday"}'
```

The full contract, including filters, sorting and pagination, is in [`docs/api/openapi.yaml`](docs/api/openapi.yaml).

## Documentation

- [System design](docs/system-design.md): scope, assumptions, architecture, decisions, future work
- [Database design](docs/database-design.md): schema, indexes, migrations, seed data
- [API spec](docs/api/openapi.yaml) (OpenAPI 3.1), browsable at `/docs` when the app is running
- [Validation](docs/validation.md): how the work was verified, what that found, and the known gaps
- [Assessment brief](docs/reference/keyloop_technical_assessment.md)

## Layout

```
backend/    Go API (cmd/api, cmd/seed, internal/<feature>, migrations, seed CSVs, Dockerfile)
frontend/   React + TypeScript app (Dockerfile, nginx.conf; the image is built from the repo root)
docs/       design documents, API spec (and its Swagger UI page), validation, reference material
scripts/    smoke test for the running containers
.claude/    project skills used when working with AI agents
```
