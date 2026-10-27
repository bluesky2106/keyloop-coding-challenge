---
name: postgres-design
description: PostgreSQL schema, migration, indexing and query rules for this project. Use when designing tables, writing migrations or SQL, or tuning queries.
---

# Postgres design rules

- Schema changes only through numbered SQL migrations in `backend/migrations/` (up and down). Never edit an applied migration.
- Enforce integrity in the database: `NOT NULL`, `FOREIGN KEY`, `CHECK`, `UNIQUE`. Do not rely on app code alone.
- Primary keys: `uuid` (`gen_random_uuid()`). Timestamps: `timestamptz`, default `now()`.
- Status/action types: a `CHECK` or enum-like lookup, one source of truth shared with the API spec.
- **Age is never stored.** Store `stocked_at date`; compute age at query time (`CURRENT_DATE - stocked_at`). The 90-day threshold is one constant passed as a query parameter.
- **Every query on tenant data filters by `dealership_id`.** Index it first in composite indexes.
- Add an index only with a named query it serves; document it in `docs/database-design.md`. Check with `EXPLAIN` on realistic data.
- Action history is append-only; "current status" is the latest row per vehicle (`DISTINCT ON` or a lateral join), not a mutated column.
- Use parameterised queries only. Never build SQL with string concatenation of user input.
- Seed data loads from `backend/seed/*.csv` idempotently (`ON CONFLICT DO NOTHING`).
