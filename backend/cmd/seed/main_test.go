package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bluesky2106/keyloop-coding-challenge/backend/internal/testdb"
)

func envOf(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

func TestRunRequiresDatabaseURL(t *testing.T) {
	err := run(t.Context(), envOf(nil))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DATABASE_URL")
}

func TestRunFailsWhenTheDatabaseIsUnreachable(t *testing.T) {
	err := run(t.Context(), envOf(map[string]string{
		"DATABASE_URL": "postgres://u@127.0.0.1:1/db?sslmode=disable&connect_timeout=1",
	}))
	require.Error(t, err)
}

func TestRunLoadsTheDemoData(t *testing.T) {
	pool := testdb.New(t)
	env := envOf(map[string]string{
		"DATABASE_URL": pool.Config().ConnString(),
		"SEED_DIR":     "../../seed",
	})

	require.NoError(t, run(t.Context(), env))

	var n int
	require.NoError(t, pool.QueryRow(t.Context(), `SELECT count(*) FROM vehicles`).Scan(&n))
	assert.Equal(t, 150, n)
}

func TestRunReportsMissingSeedFiles(t *testing.T) {
	pool := testdb.New(t)
	// SEED_DIR unset: the default "seed" directory does not exist next to this package.
	err := run(t.Context(), envOf(map[string]string{"DATABASE_URL": pool.Config().ConnString()}))
	require.Error(t, err)
}
