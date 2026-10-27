// Package seed loads the demo dataset from CSV files into the database.
//
// Loading is idempotent: every insert is keyed on a natural key and skips rows that
// already exist, so running it repeatedly never duplicates or overwrites data.
package seed

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type modelRow struct{ make, model string }

type userRow struct{ dealership, username, password string }

type vehicleRow struct {
	dealership, vin, make, model, year, price, ageDays string
}

// Load reads dealerships.csv, models.csv, users.csv and vehicles.csv from dir and
// inserts them in a single transaction.
func Load(ctx context.Context, pool *pgxpool.Pool, dir string) error {
	dealerships, err := readCSV(dir, "dealerships.csv", []string{"name"})
	if err != nil {
		return err
	}
	modelRecs, err := readCSV(dir, "models.csv", []string{"make", "model"})
	if err != nil {
		return err
	}
	userRecs, err := readCSV(dir, "users.csv", []string{"dealership", "username", "password"})
	if err != nil {
		return err
	}
	vehicleRecs, err := readCSV(dir, "vehicles.csv",
		[]string{"dealership", "vin", "make", "model", "year", "price", "age_days"})
	if err != nil {
		return err
	}

	models := make([]modelRow, len(modelRecs))
	for i, r := range modelRecs {
		models[i] = modelRow{r[0], r[1]}
	}
	users := make([]userRow, len(userRecs))
	for i, r := range userRecs {
		users[i] = userRow{r[0], r[1], r[2]}
	}
	vehicles := make([]vehicleRow, len(vehicleRecs))
	for i, r := range vehicleRecs {
		vehicles[i] = vehicleRow{r[0], r[1], r[2], r[3], r[4], r[5], r[6]}
	}

	if err := checkReferences(dealerships, models, users, vehicles); err != nil {
		return err
	}

	hashes := make([][]byte, len(users))
	for i, u := range users {
		h, err := bcrypt.GenerateFromPassword([]byte(u.password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("hash password for %q: %w", u.username, err)
		}
		hashes[i] = h
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := insert(ctx, tx, dealerships, models, users, hashes, vehicles); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

func insert(
	ctx context.Context, tx pgx.Tx,
	dealerships [][]string, models []modelRow, users []userRow, hashes [][]byte, vehicles []vehicleRow,
) error {
	for _, d := range dealerships {
		if _, err := tx.Exec(ctx,
			`INSERT INTO dealerships (name) VALUES ($1) ON CONFLICT (name) DO NOTHING`, d[0]); err != nil {
			return fmt.Errorf("insert dealership %q: %w", d[0], err)
		}
	}
	for _, m := range models {
		if _, err := tx.Exec(ctx,
			`INSERT INTO makes (name) VALUES ($1) ON CONFLICT (lower(name)) DO NOTHING`, m.make); err != nil {
			return fmt.Errorf("insert make %q: %w", m.make, err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO models (make_id, name)
			SELECT id, $2 FROM makes WHERE lower(name) = lower($1)
			ON CONFLICT (make_id, lower(name)) DO NOTHING`, m.make, m.model); err != nil {
			return fmt.Errorf("insert model %q %q: %w", m.make, m.model, err)
		}
	}
	for i, u := range users {
		if _, err := tx.Exec(ctx, `
			INSERT INTO users (dealership_id, username, password_hash)
			SELECT id, $2, $3 FROM dealerships WHERE name = $1
			ON CONFLICT (lower(username)) DO NOTHING`, u.dealership, u.username, string(hashes[i])); err != nil {
			return fmt.Errorf("insert user %q: %w", u.username, err)
		}
	}
	for _, v := range vehicles {
		// stocked_at is derived from age_days so the demo data never goes stale.
		if _, err := tx.Exec(ctx, `
			INSERT INTO vehicles (dealership_id, vin, model_id, year, price, stocked_at)
			SELECT d.id, $2, m.id, $5::smallint, $6::numeric, CURRENT_DATE - $7::int
			FROM dealerships d
			JOIN makes k ON lower(k.name) = lower($3)
			JOIN models m ON m.make_id = k.id AND lower(m.name) = lower($4)
			WHERE d.name = $1
			ON CONFLICT (dealership_id, vin) DO NOTHING`,
			v.dealership, v.vin, v.make, v.model, v.year, v.price, v.ageDays); err != nil {
			return fmt.Errorf("insert vehicle %q: %w", v.vin, err)
		}
	}
	return nil
}

// checkReferences fails early, with a clear message, if a row points at something that
// is not defined in the other files. Otherwise the SQL joins would skip it silently.
func checkReferences(dealerships [][]string, models []modelRow, users []userRow, vehicles []vehicleRow) error {
	dealerSet := map[string]bool{}
	for _, d := range dealerships {
		dealerSet[d[0]] = true
	}
	modelSet := map[string]bool{}
	for _, m := range models {
		modelSet[strings.ToLower(m.make+"\x00"+m.model)] = true
	}
	for _, u := range users {
		if !dealerSet[u.dealership] {
			return fmt.Errorf("user %q references unknown dealership %q", u.username, u.dealership)
		}
	}
	for _, v := range vehicles {
		if !dealerSet[v.dealership] {
			return fmt.Errorf("vehicle %q references unknown dealership %q", v.vin, v.dealership)
		}
		if !modelSet[strings.ToLower(v.make+"\x00"+v.model)] {
			return fmt.Errorf("vehicle %q references unknown model %q %q", v.vin, v.make, v.model)
		}
	}
	return nil
}

// readCSV returns the data rows of dir/name after checking the header matches want.
func readCSV(dir, name string, want []string) ([][]string, error) {
	f, err := os.Open(filepath.Join(dir, name)) //nolint:gosec // dir is operator-supplied configuration
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", name, err)
	}
	defer func() { _ = f.Close() }()

	r := csv.NewReader(f)
	r.FieldsPerRecord = len(want)
	recs, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", name, err)
	}
	if len(recs) == 0 || strings.Join(recs[0], ",") != strings.Join(want, ",") {
		return nil, fmt.Errorf("%s: header must be %s", name, strings.Join(want, ","))
	}
	return recs[1:], nil
}
