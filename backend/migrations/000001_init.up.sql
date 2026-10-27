CREATE TABLE dealerships (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name       text NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE users (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    dealership_id uuid NOT NULL REFERENCES dealerships (id),
    username      text NOT NULL,
    password_hash text NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now()
);

-- Usernames are case-insensitive and globally unique: login has no dealership field.
CREATE UNIQUE INDEX users_username_key ON users (lower(username));
CREATE INDEX users_dealership_id_idx ON users (dealership_id);

-- Shared reference data (not tenant data): no dealership_id.
CREATE TABLE makes (
    id   uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name text NOT NULL
);

CREATE UNIQUE INDEX makes_name_key ON makes (lower(name));

CREATE TABLE models (
    id      uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    make_id uuid NOT NULL REFERENCES makes (id),
    name    text NOT NULL
);

CREATE UNIQUE INDEX models_make_name_key ON models (make_id, lower(name));

CREATE TABLE vehicles (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    dealership_id uuid NOT NULL REFERENCES dealerships (id),
    vin           text NOT NULL CHECK (vin ~ '^[A-HJ-NPR-Z0-9]{17}$'),
    model_id      uuid NOT NULL REFERENCES models (id),
    year          smallint NOT NULL CHECK (year BETWEEN 1980 AND 2100),
    price         numeric(10, 2) NOT NULL CHECK (price >= 0),
    stocked_at    date NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    UNIQUE (dealership_id, vin),
    -- Target for the composite foreign key from vehicle_actions.
    UNIQUE (id, dealership_id)
);

-- Serves the aging filter and age sort: WHERE dealership_id = $1 ORDER BY stocked_at.
CREATE INDEX vehicles_dealership_stocked_at_idx ON vehicles (dealership_id, stocked_at);
-- Serves the model filter; the make filter reaches it through models.make_id.
CREATE INDEX vehicles_dealership_model_idx ON vehicles (dealership_id, model_id);

CREATE TABLE vehicle_actions (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    vehicle_id    uuid NOT NULL,
    dealership_id uuid NOT NULL,
    user_id       uuid NOT NULL REFERENCES users (id),
    action_type   text NOT NULL CHECK (action_type IN (
                      'price_reduction_planned',
                      'promotion_planned',
                      'transfer_planned',
                      'wholesale_planned',
                      'hold'
                  )),
    note          text CHECK (char_length(note) <= 500),
    created_at    timestamptz NOT NULL DEFAULT now(),
    -- A vehicle's actions must belong to the vehicle's own dealership.
    FOREIGN KEY (vehicle_id, dealership_id) REFERENCES vehicles (id, dealership_id)
);

-- Serves "latest action per vehicle" and the per-vehicle history list.
CREATE INDEX vehicle_actions_vehicle_created_idx ON vehicle_actions (vehicle_id, created_at DESC);
