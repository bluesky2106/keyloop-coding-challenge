package testdb

import (
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// Fixtures inserts small, exact datasets for store tests. It starts with two dealerships
// (North and South), one user in each, and the make Toyota with the model Corolla.
type Fixtures struct {
	t    *testing.T
	pool *pgxpool.Pool
	n    int

	North, South         string // dealership ids
	NorthUser, SouthUser string // user ids
	Toyota, Corolla      string // make and model ids
}

// NewFixtures creates the base rows in pool, which must be migrated.
func NewFixtures(t *testing.T, pool *pgxpool.Pool) *Fixtures {
	t.Helper()
	f := &Fixtures{t: t, pool: pool}
	f.North = f.insert(`INSERT INTO dealerships (name) VALUES ('North')`)
	f.South = f.insert(`INSERT INTO dealerships (name) VALUES ('South')`)
	f.NorthUser = f.insert(`INSERT INTO users (dealership_id, username, password_hash) VALUES ($1, 'north.user', 'h')`, f.North)
	f.SouthUser = f.insert(`INSERT INTO users (dealership_id, username, password_hash) VALUES ($1, 'south.user', 'h')`, f.South)
	f.Toyota, f.Corolla = f.AddModel("Toyota", "Corolla")
	return f
}

func (f *Fixtures) insert(sql string, args ...any) string {
	f.t.Helper()
	var id string
	require.NoError(f.t, f.pool.QueryRow(f.t.Context(), sql+" RETURNING id::text", args...).Scan(&id))
	return id
}

// AddModel adds a model, creating its make if needed, and returns the make and model ids.
func (f *Fixtures) AddModel(makeName, model string) (makeID, modelID string) {
	f.t.Helper()
	err := f.pool.QueryRow(f.t.Context(), `SELECT id::text FROM makes WHERE lower(name) = lower($1)`, makeName).Scan(&makeID)
	if err != nil {
		makeID = f.insert(`INSERT INTO makes (name) VALUES ($1)`, makeName)
	}
	return makeID, f.insert(`INSERT INTO models (make_id, name) VALUES ($1, $2)`, makeID, model)
}

// VehicleSpec describes a vehicle to add. Zero values mean: North, Corolla, age 0 days,
// year 2024 and price 10000.00.
type VehicleSpec struct {
	Dealership string
	Model      string
	AgeDays    int
	Year       int
	Price      string
}

// AddVehicle inserts a vehicle that has been in stock for exactly spec.AgeDays days and
// returns its id.
func (f *Fixtures) AddVehicle(spec VehicleSpec) string {
	f.t.Helper()
	if spec.Dealership == "" {
		spec.Dealership = f.North
	}
	if spec.Model == "" {
		spec.Model = f.Corolla
	}
	if spec.Year == 0 {
		spec.Year = 2024
	}
	if spec.Price == "" {
		spec.Price = "10000.00"
	}
	f.n++
	vin := fmt.Sprintf("1HGCM%012d", f.n)
	return f.insert(`
		INSERT INTO vehicles (dealership_id, vin, model_id, year, price, stocked_at)
		VALUES ($1, $2, $3, $4, $5::numeric, CURRENT_DATE - $6::int)`,
		spec.Dealership, vin, spec.Model, spec.Year, spec.Price, spec.AgeDays)
}

// AddAction appends an action to a vehicle on behalf of a user of the same dealership.
func (f *Fixtures) AddAction(vehicleID, dealershipID, userID, actionType, note string) string {
	f.t.Helper()
	return f.insert(`
		INSERT INTO vehicle_actions (vehicle_id, dealership_id, user_id, action_type, note)
		VALUES ($1, $2, $3, $4, NULLIF($5, ''))`, vehicleID, dealershipID, userID, actionType, note)
}
