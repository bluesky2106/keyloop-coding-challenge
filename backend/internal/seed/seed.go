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
	"golang.org/x/crypto/bcrypt"
)

// DB starts transactions. *pgxpool.Pool satisfies it.
type DB interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// hashCost is the bcrypt cost for demo passwords. Tests lower it to stay fast.
var hashCost = bcrypt.DefaultCost

type modelRow struct{ make, model string }

type userRow struct{ dealership, username, password string }

type vehicleRow struct {
	dealership, vin, make, model, year, price, ageDays string
}

type dataset struct {
	dealerships []string
	models      []modelRow
	users       []userRow
	vehicles    []vehicleRow
}

// Load reads dealerships.csv, models.csv, users.csv and vehicles.csv from dir and
// inserts them in a single transaction. Nothing is written if any file is invalid.
func Load(ctx context.Context, db DB, dir string) error {
	ds, err := read(dir)
	if err != nil {
		return err
	}

	hashes := make([]string, len(ds.users))
	for i, u := range ds.users {
		h, err := bcrypt.GenerateFromPassword([]byte(u.password), hashCost)
		if err != nil {
			return fmt.Errorf("hash password for %q: %w", u.username, err)
		}
		hashes[i] = string(h)
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := insert(ctx, tx, ds, hashes); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

func insert(ctx context.Context, tx pgx.Tx, ds dataset, hashes []string) error {
	for _, name := range ds.dealerships {
		if _, err := tx.Exec(ctx,
			`INSERT INTO dealerships (name) VALUES ($1) ON CONFLICT (name) DO NOTHING`, name); err != nil {
			return fmt.Errorf("insert dealership %q: %w", name, err)
		}
	}
	for _, m := range ds.models {
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
	for i, u := range ds.users {
		if _, err := tx.Exec(ctx, `
			INSERT INTO users (dealership_id, username, password_hash)
			SELECT id, $2, $3 FROM dealerships WHERE name = $1
			ON CONFLICT (lower(username)) DO NOTHING`, u.dealership, u.username, hashes[i]); err != nil {
			return fmt.Errorf("insert user %q: %w", u.username, err)
		}
	}
	for _, v := range ds.vehicles {
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

// read parses the four CSV files and checks that every row points at something defined
// in the other files. Otherwise the SQL joins would skip such rows silently.
func read(dir string) (dataset, error) {
	var ds dataset

	rows, err := readCSV(dir, "dealerships.csv", []string{"name"})
	if err != nil {
		return ds, err
	}
	dealerSet := map[string]bool{}
	for _, r := range rows {
		ds.dealerships = append(ds.dealerships, r[0])
		dealerSet[r[0]] = true
	}

	rows, err = readCSV(dir, "models.csv", []string{"make", "model"})
	if err != nil {
		return ds, err
	}
	modelSet := map[string]bool{}
	for _, r := range rows {
		ds.models = append(ds.models, modelRow{r[0], r[1]})
		modelSet[modelKey(r[0], r[1])] = true
	}

	rows, err = readCSV(dir, "users.csv", []string{"dealership", "username", "password"})
	if err != nil {
		return ds, err
	}
	for _, r := range rows {
		if !dealerSet[r[0]] {
			return ds, fmt.Errorf("user %q references unknown dealership %q", r[1], r[0])
		}
		ds.users = append(ds.users, userRow{r[0], r[1], r[2]})
	}

	rows, err = readCSV(dir, "vehicles.csv",
		[]string{"dealership", "vin", "make", "model", "year", "price", "age_days"})
	if err != nil {
		return ds, err
	}
	for _, r := range rows {
		if !dealerSet[r[0]] {
			return ds, fmt.Errorf("vehicle %q references unknown dealership %q", r[1], r[0])
		}
		if !modelSet[modelKey(r[2], r[3])] {
			return ds, fmt.Errorf("vehicle %q references unknown model %q %q", r[1], r[2], r[3])
		}
		ds.vehicles = append(ds.vehicles, vehicleRow{r[0], r[1], r[2], r[3], r[4], r[5], r[6]})
	}
	return ds, nil
}

func modelKey(makeName, model string) string { return strings.ToLower(makeName + "\x00" + model) }

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
