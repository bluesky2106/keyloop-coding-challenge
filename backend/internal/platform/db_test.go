package platform_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bluesky2106/keyloop-coding-challenge/backend/internal/platform"
	"github.com/bluesky2106/keyloop-coding-challenge/backend/internal/testdb"
)

func TestConnect(t *testing.T) {
	t.Run("reachable database", func(t *testing.T) {
		pool := testdb.New(t)
		again, err := platform.Connect(t.Context(), pool.Config().ConnString())
		require.NoError(t, err)
		again.Close()
	})

	t.Run("malformed url", func(t *testing.T) {
		_, err := platform.Connect(t.Context(), "://not a url")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "open pool")
	})

	t.Run("unreachable server", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()
		_, err := platform.Connect(ctx, "postgres://u@127.0.0.1:1/db?sslmode=disable&connect_timeout=1")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ping database")
	})
}

func TestMigrate(t *testing.T) {
	t.Run("is idempotent", func(t *testing.T) {
		pool := testdb.New(t) // already migrated once
		require.NoError(t, platform.Migrate(pool.Config().ConnString()), "second run must be a no-op")
	})

	t.Run("applies the schema to an empty database", func(t *testing.T) {
		pool := testdb.NewEmpty(t)
		require.NoError(t, platform.Migrate(pool.Config().ConnString()))
		var n int
		require.NoError(t, pool.QueryRow(t.Context(), `SELECT count(*) FROM vehicles`).Scan(&n))
	})

	t.Run("reports a migration that cannot apply", func(t *testing.T) {
		pool := testdb.NewEmpty(t)
		_, err := pool.Exec(t.Context(), `CREATE TABLE dealerships (id int)`)
		require.NoError(t, err)

		err = platform.Migrate(pool.Config().ConnString())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "apply migrations")
	})

	t.Run("rejects a database URL that is not postgres://", func(t *testing.T) {
		err := platform.Migrate("host=localhost dbname=db")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must start with postgres://")
	})

	t.Run("unreachable database", func(t *testing.T) {
		err := platform.Migrate("postgres://u@127.0.0.1:1/db?sslmode=disable&connect_timeout=1")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "init migrate")
	})
}
