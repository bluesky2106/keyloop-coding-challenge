# System Design: Intelligent Inventory Dashboard

## 1. Goal and scope

Give dealership managers an overview of their vehicle stock, surface **aging stock** (in inventory for more than 90 days) and let them record the action planned for each aging vehicle.

**In scope**
- Filterable inventory list (make, model, age, and status)
- Automatic aging identification, prominently displayed
- Action log per aging vehicle, persisted with history
- Multiple dealerships with username/password login (JWT) and strict data isolation
- CSV-based mock data

**Out of scope** (see [Future evolution](#7-future-evolution))
- Live push updates, user management UI, vehicle create/edit, DMS integrations

## 2. Assumptions

The brief allows ambiguity as long as it is documented.

| # | Assumption |
|---|---|
| A1 | "Age" is the number of whole days since the vehicle entered the dealership's stock (`stocked_at`). A vehicle is *aging* when age is **greater than 90** days (90 exactly is not aging). |
| A2 | Age is computed at query time from `stocked_at` and is never stored, so it cannot go stale. |
| A3 | The 90-day threshold is a single constant in this version (see 7.1 for the configurable design). |
| A4 | Each user belongs to exactly one dealership and sees only that dealership's vehicles. |
| A5 | Authentication is username + password; the API issues a short-lived JWT. No registration, password reset or roles beyond "dealership user". Users are seeded. |
| A6 | Actions are chosen from a fixed list (*Price Reduction Planned*, *Promotion Planned*, *Transfer Planned*, *Wholesale Planned*, *Hold*) with an optional free-text note. |
| A7 | Action history is append-only. A vehicle's *current* status is its most recent action. |
| A8 | "Real-time" means fresh data on page load and manual refresh. No push updates. |
| A9 | Only vehicles currently in stock are listed. Sold vehicles are out of scope. |
| A10 | Mock data (dealerships, users, vehicles) is stored as CSV and loaded idempotently. All credentials in it are fake demo values. |

## 3. Architecture

```mermaid
flowchart LR
    Browser["React + TypeScript SPA"] -->|"HTTPS JSON, Bearer JWT"| API["Go API (chi)"]
    API -->|"pgx / SQL"| DB[("PostgreSQL")]
    CSV["seed CSV files"] -->|"idempotent load"| DB
```

| Component | Responsibility |
|---|---|
| **SPA** | Login, inventory table with URL-driven filters, aging highlight, action dialog and history. |
| **API** | Authentication, request validation, tenant scoping, inventory queries, action recording. Stateless, so it scales horizontally. |
| **PostgreSQL** | Source of truth. Enforces integrity with constraints and foreign keys. |
| **Seed loader** | Loads dealerships, users and vehicles from CSV on demand. |

The API is organised by feature (`auth`, `inventory`, `actions`) with shared infrastructure in `platform`. This keeps related code together and avoids a layered folder structure that spreads one feature over many directories.

## 4. Key decisions

| Decision | Options | Choice | Trade-off |
|---|---|---|---|
| Backend language | Go, Node, .NET | **Go** | Small, fast, easy to deploy; more verbose than Node. |
| Database | SQLite, PostgreSQL | **PostgreSQL** | Real concurrency, constraints and indexing; needs a container to run. |
| Age handling | Stored column, computed | **Computed at query time** | Always correct and no batch job; the query must be indexed well. |
| Current status | Mutable column, append-only history | **Append-only history**, latest row is current | Full audit trail; reads need a "latest per vehicle" query. |
| Auth | Sessions, JWT | **JWT (HS256, short expiry)** | Stateless and simple; no server-side revocation in v1. |
| Tenancy | Per-dealership schema, shared schema | **Shared schema, `dealership_id` on every row** | Simple and cheap; isolation depends on every query filtering by tenant (enforced in the repository layer and tested). |
| API contract | Code-first, spec-first | **OpenAPI spec first**, TypeScript types generated | Frontend and backend cannot drift; adds a generation step. |
| Freshness | Polling, SSE, refresh on load | **Refresh on load** | Meets the requirement with no extra infrastructure. |

## 5. Security and multi-tenancy

- Passwords hashed with bcrypt. JWT carries `user_id` and `dealership_id`.
- `dealership_id` is taken **only from the token**, never from the request. Repository methods require it as a parameter, so a query cannot be written without a tenant.
- Requesting another dealership's vehicle returns `404`, which does not reveal that it exists.
- All SQL is parameterised. Input is validated at the handler boundary and list endpoints cap page size.
- Secrets come from environment variables. Demo credentials exist only in seed data.

## 6. Quality approach

- Lint and format: `golangci-lint` v2 for Go; ESLint (type-checked strict rules), Prettier and `tsc` for the frontend. Run everything with `make check`; CI runs the same command.
- Tests: unit tests for pure logic (aging boundary at 90/91 days), integration tests against a real Postgres, component tests for the frontend.
- Evidence of verification is recorded in `docs/validation.md`.

## 7. Future evolution

1. **Configurable aging threshold.** Add `aging_threshold_days` to `dealerships` (default 90) so each dealership sets its own rule, and optionally per-make overrides in a small `aging_rules` table. The query already takes the threshold as a parameter, so only its source changes; the API gains a settings endpoint.
2. **Live updates.** Server-Sent Events from the API, fed by Postgres `LISTEN/NOTIFY`, would push action and stock changes to open dashboards.
3. **Scale.** The API is stateless and can run behind a load balancer. Read replicas and a materialised aging view become useful at very large inventories.
4. **Security hardening.** Refresh tokens with rotation, rate limiting on login, role-based access (manager vs. viewer).
5. **Operations.** Structured logs are already used; add metrics, tracing and a deployment pipeline.
