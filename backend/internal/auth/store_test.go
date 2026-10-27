package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bluesky2106/keyloop-coding-challenge/backend/internal/testdb"
)

func TestPGStore(t *testing.T) {
	pool := testdb.New(t)
	ctx := t.Context()
	var dealershipID, otherID, userID string
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO dealerships (name) VALUES ('Acme') RETURNING id::text`).Scan(&dealershipID))
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO dealerships (name) VALUES ('Other') RETURNING id::text`).Scan(&otherID))
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO users (dealership_id, username, password_hash) VALUES ($1, 'Alice', 'hash') RETURNING id::text`,
		dealershipID).Scan(&userID))
	store := NewPGStore(pool)

	t.Run("find user ignores case", func(t *testing.T) {
		u, err := store.FindUserByUsername(ctx, "aLiCe")
		require.NoError(t, err)
		assert.Equal(t, User{ID: userID, DealershipID: dealershipID, PasswordHash: "hash"}, u)
	})

	t.Run("find unknown user", func(t *testing.T) {
		_, err := store.FindUserByUsername(ctx, "ghost")
		require.ErrorIs(t, err, ErrUserNotFound)
	})

	t.Run("profile", func(t *testing.T) {
		m, err := store.FindProfile(ctx, Claims{UserID: userID, DealershipID: dealershipID})
		require.NoError(t, err)
		assert.Equal(t, Me{UserID: userID, Username: "Alice", DealershipID: dealershipID, DealershipName: "Acme"}, m)
	})

	t.Run("profile does not cross dealerships", func(t *testing.T) {
		_, err := store.FindProfile(ctx, Claims{UserID: userID, DealershipID: otherID})
		require.ErrorIs(t, err, ErrUserNotFound)
	})

	t.Run("database failure is not reported as not found", func(t *testing.T) {
		pool.Close()
		_, err := store.FindUserByUsername(ctx, "alice")
		require.Error(t, err)
		assert.NotErrorIs(t, err, ErrUserNotFound)
		_, err = store.FindProfile(ctx, Claims{UserID: userID, DealershipID: dealershipID})
		require.Error(t, err)
		assert.NotErrorIs(t, err, ErrUserNotFound)
	})
}
