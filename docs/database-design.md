# Database Design

PostgreSQL 17. The schema is created by `backend/migrations/000001_init.up.sql` (golang-migrate format, with a matching `.down.sql`).

## ERD

```mermaid
erDiagram
    dealerships ||--o{ users : employs
    dealerships ||--o{ vehicles : stocks
    vehicles ||--o{ vehicle_actions : "has history"
    users ||--o{ vehicle_actions : records

    dealerships {
        uuid id PK
        text name UK
        timestamptz created_at
    }
    users {
        uuid id PK
        uuid dealership_id FK
        text username "unique, case-insensitive"
        text password_hash "bcrypt"
        timestamptz created_at
    }
    vehicles {
        uuid id PK
        uuid dealership_id FK
        text vin "unique per dealership"
        text make
        text model
        smallint year
        numeric price
        date stocked_at "age is derived from this"
        timestamptz created_at
    }
    vehicle_actions {
        uuid id PK
        uuid vehicle_id FK
        uuid dealership_id FK
        uuid user_id FK
        text action_type
        text note
        timestamptz created_at
    }
```

## Tables

| Table | Notes |
|---|---|
| `dealerships` | The tenant. `name` is unique and is the natural key used by the seed files. |
| `users` | Belongs to one dealership (A4). Username is unique across all dealerships on `lower(username)` because login has no dealership field. Only a bcrypt hash is stored. |
| `vehicles` | Current stock (A9). `VIN` must be 17 characters without I, O or Q and is unique per dealership. `stocked_at` is the only age-related column. |
| `vehicle_actions` | Append-only history (A7). Rows are never updated or deleted. |

### Design points

- **Age is derived, never stored (A2).** A vehicle is aging when `stocked_at < CURRENT_DATE - 90`, which means more than 90 days (A1). Written this way the predicate compares the bare column, so the index below can serve it. The `90` is passed as a query parameter, so making it configurable later changes only where the value comes from.
- **Tenant integrity is enforced by the database.** `vehicle_actions` carries `dealership_id` and a composite foreign key `(vehicle_id, dealership_id) -> vehicles (id, dealership_id)`. An action cannot be attached to a vehicle of another dealership, even through a buggy query.
- **Action types are a `CHECK` list**, not a lookup table, because the set is small and fixed (A6): `price_reduction_planned`, `promotion_planned`, `transfer_planned`, `wholesale_planned`, `hold`. The same values appear in the OpenAPI enum. Changing the list is a new migration.
- **Notes** are optional and capped at 500 characters.
- **Current status** is the newest action for the vehicle; there is no mutable status column.

## Indexes

| Index | Query it serves |
|---|---|
| `vehicles_dealership_stocked_at_idx (dealership_id, stocked_at)` | Inventory list scoped to a dealership, the aging filter and sort by age. |
| `vehicles_dealership_make_model_idx (dealership_id, make, model)` | Make and model filters. |
| `vehicle_actions_vehicle_created_idx (vehicle_id, created_at DESC)` | Latest action per vehicle and the history list for one vehicle. |
| `users_username_key (lower(username))` | Login lookup. |
| `users_dealership_id_idx`, unique `(dealership_id, vin)` | Foreign-key lookups and VIN uniqueness. |

Reference queries:

```sql
-- Aging stock for a dealership, oldest first, with each vehicle's latest action
SELECT v.*, CURRENT_DATE - v.stocked_at AS age_days, a.action_type, a.note
FROM vehicles v
LEFT JOIN LATERAL (
    SELECT action_type, note
    FROM vehicle_actions
    WHERE vehicle_id = v.id
    ORDER BY created_at DESC
    LIMIT 1
) a ON true
WHERE v.dealership_id = $1
  AND v.stocked_at < CURRENT_DATE - $2::int   -- $2 = 90
ORDER BY v.stocked_at;
```

## Migrations

Numbered SQL files in `backend/migrations/`, applied with golang-migrate (`migrate -path backend/migrations -database "$DATABASE_URL" up`). Applied migrations are never edited; changes are new files. Every migration has a down file.

## Seed data

CSV files in `backend/seed/`:

| File | Columns |
|---|---|
| `dealerships.csv` | `name` |
| `users.csv` | `dealership`, `username`, `password` (plaintext **demo** values, hashed by the loader) |
| `vehicles.csv` | `dealership`, `vin`, `make`, `model`, `year`, `price`, `age_days` |

Vehicles use `age_days` rather than a fixed date, and the loader sets `stocked_at = CURRENT_DATE - age_days`. Fixed dates would make every vehicle look older each day and eventually turn the whole demo inventory into aging stock.

The data has 3 dealerships, 6 users and 150 vehicles (50 each); about 27% are over 90 days old. Each dealership includes vehicles at exactly 90 and 91 days to exercise the boundary.

The loader inserts with `ON CONFLICT DO NOTHING` on the natural keys (dealership name, username, `(dealership, vin)`), so it can be re-run safely. It is implemented together with the backend.

## Verification

Run against a real PostgreSQL 17 container: the migration applies and reverses cleanly; the CSVs load and reload idempotently (150 vehicles both times); age 90 evaluates as not aging and 91 as aging; and the index is used for the aging predicate. Inserts with an invalid action type, a cross-dealership action, a malformed VIN and a case-variant duplicate username are each rejected by the database.
