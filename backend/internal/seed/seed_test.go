package seed

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/bluesky2106/keyloop-coding-challenge/backend/internal/testdb"
)

const realSeedDir = "../../seed"

func init() { hashCost = bcrypt.MinCost }

// validFiles is the smallest dataset that satisfies every cross-file reference.
func validFiles() map[string]string {
	return map[string]string{
		"dealerships.csv": "name\nAcme\n",
		"models.csv":      "make,model\nToyota,Corolla\n",
		"users.csv":       "dealership,username,password\nAcme,alice,pw\n",
		"vehicles.csv": "dealership,vin,make,model,year,price,age_days\n" +
			"Acme,11111111111111111,Toyota,Corolla,2024,100.00,5\n",
	}
}

func writeDir(t *testing.T, override map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	files := validFiles()
	for name, body := range override {
		files[name] = body
	}
	for name, body := range files {
		if body == "<missing>" {
			continue
		}
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600))
	}
	return dir
}

func TestReadAcceptsValidDataset(t *testing.T) {
	ds, err := read(writeDir(t, nil))
	require.NoError(t, err)
	assert.Equal(t, []string{"Acme"}, ds.dealerships)
	assert.Equal(t, []modelRow{{"Toyota", "Corolla"}}, ds.models)
	assert.Equal(t, []userRow{{"Acme", "alice", "pw"}}, ds.users)
	assert.Equal(t, []vehicleRow{{"Acme", "11111111111111111", "Toyota", "Corolla", "2024", "100.00", "5"}}, ds.vehicles)
}

func TestReadRejectsInvalidInput(t *testing.T) {
	const vehicleHeader = "dealership,vin,make,model,year,price,age_days\n"
	tests := []struct {
		name     string
		override map[string]string
		wantErr  string
	}{
		{"missing dealerships file", map[string]string{"dealerships.csv": "<missing>"}, "open dealerships.csv"},
		{"missing models file", map[string]string{"models.csv": "<missing>"}, "open models.csv"},
		{"missing users file", map[string]string{"users.csv": "<missing>"}, "open users.csv"},
		{"missing vehicles file", map[string]string{"vehicles.csv": "<missing>"}, "open vehicles.csv"},
		{"wrong header", map[string]string{"dealerships.csv": "title\nAcme\n"}, "header must be name"},
		{"empty file", map[string]string{"models.csv": ""}, "header must be make,model"},
		{"wrong column count", map[string]string{"users.csv": "dealership,username,password\nAcme,alice\n"}, "read users.csv"},
		{
			"user with unknown dealership",
			map[string]string{"users.csv": "dealership,username,password\nNope,alice,pw\n"},
			`unknown dealership "Nope"`,
		},
		{
			"vehicle with unknown dealership",
			map[string]string{"vehicles.csv": vehicleHeader + "Nope,11111111111111111,Toyota,Corolla,2024,1.00,5\n"},
			`unknown dealership "Nope"`,
		},
		{
			"vehicle with unknown model",
			map[string]string{"vehicles.csv": vehicleHeader + "Acme,11111111111111111,Toyota,Missing,2024,1.00,5\n"},
			`unknown model "Toyota" "Missing"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := read(writeDir(t, tt.override))
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

// fakeTx records Exec calls and fails the failAt-th one (1-based; 0 never fails).
type fakeTx struct {
	pgx.Tx
	failAt, calls int
	commitErr     error
	committed     bool
	rolledBack    bool
	userHash      string
}

var errBoom = errors.New("boom")

func (f *fakeTx) Exec(_ context.Context, _ string, args ...any) (pgconn.CommandTag, error) {
	f.calls++
	if f.calls == f.failAt {
		return pgconn.CommandTag{}, errBoom
	}
	if len(args) == 3 { // the users insert: (dealership, username, password_hash)
		f.userHash, _ = args[2].(string)
	}
	return pgconn.CommandTag{}, nil
}

func (f *fakeTx) Commit(context.Context) error {
	f.committed = f.commitErr == nil
	return f.commitErr
}

func (f *fakeTx) Rollback(context.Context) error {
	f.rolledBack = true
	return nil
}

func dbReturning(t *testing.T, tx pgx.Tx, err error) *MockDB {
	t.Helper()
	db := NewMockDB(t)
	db.EXPECT().Begin(mock.Anything).Return(tx, err)
	return db
}

func TestLoadWritesEveryTableInOneTransaction(t *testing.T) {
	tx := &fakeTx{}

	require.NoError(t, Load(t.Context(), dbReturning(t, tx, nil), writeDir(t, nil)))

	assert.Equal(t, 5, tx.calls, "dealership, make, model, user and vehicle inserts")
	assert.True(t, tx.committed)
	require.NoError(t, bcrypt.CompareHashAndPassword([]byte(tx.userHash), []byte("pw")),
		"the password must be stored as a bcrypt hash")
}

func TestLoadFailures(t *testing.T) {
	t.Run("an insert fails", func(t *testing.T) {
		wants := []string{
			`insert dealership "Acme"`, `insert make "Toyota"`, `insert model "Toyota" "Corolla"`,
			`insert user "alice"`, `insert vehicle "11111111111111111"`,
		}
		for i, want := range wants {
			tx := &fakeTx{failAt: i + 1}

			err := Load(t.Context(), dbReturning(t, tx, nil), writeDir(t, nil))

			require.ErrorIs(t, err, errBoom)
			assert.Contains(t, err.Error(), want)
			assert.False(t, tx.committed, "a failed load must not commit")
			assert.True(t, tx.rolledBack)
		}
	})

	t.Run("commit fails", func(t *testing.T) {
		err := Load(t.Context(), dbReturning(t, &fakeTx{commitErr: errBoom}, nil), writeDir(t, nil))
		require.ErrorIs(t, err, errBoom)
		assert.Contains(t, err.Error(), "commit")
	})

	t.Run("begin fails", func(t *testing.T) {
		err := Load(t.Context(), dbReturning(t, nil, errBoom), writeDir(t, nil))
		require.ErrorIs(t, err, errBoom)
		assert.Contains(t, err.Error(), "begin")
	})

	t.Run("invalid files never reach the database", func(t *testing.T) {
		err := Load(t.Context(), NewMockDB(t), writeDir(t, map[string]string{"models.csv": "<missing>"}))
		require.Error(t, err)
	})

	t.Run("unusable bcrypt cost", func(t *testing.T) {
		old := hashCost
		hashCost = 100
		t.Cleanup(func() { hashCost = old })

		err := Load(t.Context(), NewMockDB(t), writeDir(t, nil))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "hash password")
	})
}

func TestLoadRealDatasetIsIdempotent(t *testing.T) {
	pool := testdb.New(t)
	ctx := t.Context()
	queries := []string{
		"SELECT count(*) FROM dealerships", "SELECT count(*) FROM users",
		"SELECT count(*) FROM makes", "SELECT count(*) FROM models", "SELECT count(*) FROM vehicles",
	}
	counts := func() []int {
		out := make([]int, len(queries))
		for i, q := range queries {
			require.NoError(t, pool.QueryRow(ctx, q).Scan(&out[i]), q)
		}
		return out
	}

	require.NoError(t, Load(ctx, pool, realSeedDir))
	first := counts()
	assert.Equal(t, []int{3, 6, 7, 21, 150}, first)

	require.NoError(t, Load(ctx, pool, realSeedDir))
	assert.Equal(t, first, counts(), "loading twice must not duplicate rows")

	var hash string
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT password_hash FROM users WHERE username = 'northside.manager'`).Scan(&hash))
	require.NoError(t, bcrypt.CompareHashAndPassword([]byte(hash), []byte("demo-northside-1")))

	// Boundary vehicles: 90 days is not aging, 91 is.
	var aging90, aging91 bool
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT bool_or(CURRENT_DATE - stocked_at = 90 AND stocked_at < CURRENT_DATE - 90),
		       bool_or(CURRENT_DATE - stocked_at = 91 AND stocked_at < CURRENT_DATE - 90)
		FROM vehicles`).Scan(&aging90, &aging91))
	assert.False(t, aging90, "age 90 must not be aging")
	assert.True(t, aging91, "age 91 must be aging")
}
