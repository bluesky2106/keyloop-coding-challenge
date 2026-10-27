package migrations_test

import (
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bluesky2106/keyloop-coding-challenge/backend/internal/testdb"
)

// These tests pin the integrity rules described in docs/database-design.md: the
// database, not the application, must reject bad data.
func TestSchemaConstraints(t *testing.T) {
	pool := testdb.New(t)
	ctx := t.Context()
	exec := func(sql string, args ...any) error {
		_, err := pool.Exec(ctx, sql, args...)
		return err
	}
	id := func(sql string, args ...any) string {
		t.Helper()
		var out string
		require.NoError(t, pool.QueryRow(ctx, sql+" RETURNING id::text", args...).Scan(&out))
		return out
	}

	north := id(`INSERT INTO dealerships (name) VALUES ('North')`)
	south := id(`INSERT INTO dealerships (name) VALUES ('South')`)
	user := id(`INSERT INTO users (dealership_id, username, password_hash) VALUES ($1, 'Alice', 'h')`, north)
	toyota := id(`INSERT INTO makes (name) VALUES ('Toyota')`)
	corolla := id(`INSERT INTO models (make_id, name) VALUES ($1, 'Corolla')`, toyota)
	const vin = "1HGCM82633A004352"
	vehicle := id(`INSERT INTO vehicles (dealership_id, vin, model_id, year, price, stocked_at)
		VALUES ($1, $2, $3, 2024, 100, CURRENT_DATE)`, north, vin, corolla)

	insertVehicle := func(dealership, vin string, model any, year int, price string) error {
		return exec(`INSERT INTO vehicles (dealership_id, vin, model_id, year, price, stocked_at)
			VALUES ($1, $2, $3, $4, $5, CURRENT_DATE)`, dealership, vin, model, year, price)
	}
	insertAction := func(vehicle, dealership, actionType, note string) error {
		return exec(`INSERT INTO vehicle_actions (vehicle_id, dealership_id, user_id, action_type, note)
			VALUES ($1, $2, $3, $4, $5)`, vehicle, dealership, user, actionType, note)
	}

	const (
		unique  = "23505"
		check   = "23514"
		notNull = "23502"
		foreign = "23503"
	)
	// Each case must fail for the stated reason, not merely fail.
	rejected := []struct {
		name string
		err  error
		code string
	}{
		{"duplicate dealership name", exec(`INSERT INTO dealerships (name) VALUES ('North')`), unique},
		{"username differing only in case",
			exec(`INSERT INTO users (dealership_id, username, password_hash) VALUES ($1, 'ALICE', 'h')`, south), unique},
		{"make differing only in case", exec(`INSERT INTO makes (name) VALUES ('TOYOTA')`), unique},
		{"model repeated within a make",
			exec(`INSERT INTO models (make_id, name) VALUES ($1, 'COROLLA')`, toyota), unique},
		{"vehicle without a model", insertVehicle(north, "2HGCM82633A004352", nil, 2024, "1"), notNull},
		{"vehicle with an unknown model",
			insertVehicle(north, "5HGCM82633A004352", "00000000-0000-0000-0000-000000000000", 2024, "1"), foreign},
		{"VIN repeated within a dealership", insertVehicle(north, vin, corolla, 2024, "1"), unique},
		{"VIN with a forbidden letter", insertVehicle(north, "IIIIIIIIIIIIIIIII", corolla, 2024, "1"), check},
		{"VIN of the wrong length", insertVehicle(north, "SHORT", corolla, 2024, "1"), check},
		{"year before 1980", insertVehicle(north, "3HGCM82633A004352", corolla, 1979, "1"), check},
		{"negative price", insertVehicle(north, "4HGCM82633A004352", corolla, 2024, "-1"), check},
		{"unknown action type", insertAction(vehicle, north, "bogus", ""), check},
		{"action on another dealership's vehicle", insertAction(vehicle, south, "hold", ""), foreign},
		{"note over 500 characters", insertAction(vehicle, north, "hold", strings.Repeat("x", 501)), check},
	}
	for _, tt := range rejected {
		var pgErr *pgconn.PgError
		if assert.ErrorAs(t, tt.err, &pgErr, tt.name) {
			assert.Equal(t, tt.code, pgErr.Code, "%s: %s", tt.name, pgErr.Message)
		}
	}

	t.Run("the same VIN is allowed in a different dealership", func(t *testing.T) {
		require.NoError(t, insertVehicle(south, vin, corolla, 2024, "1"))
	})
	t.Run("every action type is accepted", func(t *testing.T) {
		for _, a := range []string{
			"price_reduction_planned", "promotion_planned", "transfer_planned", "wholesale_planned", "hold",
		} {
			require.NoError(t, insertAction(vehicle, north, a, "ok"), a)
		}
	})
}
