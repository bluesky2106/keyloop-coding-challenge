package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// User is the stored credential record used to verify a login.
type User struct {
	ID           string
	DealershipID string
	PasswordHash string
}

// Store reads users from persistent storage. Both methods return ErrUserNotFound
// when nothing matches.
type Store interface {
	// FindUserByUsername matches the username case-insensitively.
	FindUserByUsername(ctx context.Context, username string) (User, error)
	// FindProfile loads the user and dealership named in the claims. It matches both,
	// so a token never resolves to another tenant's data.
	FindProfile(ctx context.Context, c Claims) (Me, error)
}

// PGStore is the PostgreSQL implementation of Store.
type PGStore struct{ db *pgxpool.Pool }

// NewPGStore returns a Store backed by db.
func NewPGStore(db *pgxpool.Pool) *PGStore { return &PGStore{db: db} }

// FindUserByUsername implements Store.
func (s *PGStore) FindUserByUsername(ctx context.Context, username string) (User, error) {
	var u User
	err := s.db.QueryRow(ctx,
		`SELECT id::text, dealership_id::text, password_hash FROM users WHERE lower(username) = lower($1)`,
		username).Scan(&u.ID, &u.DealershipID, &u.PasswordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("find user: %w", err)
	}
	return u, nil
}

// FindProfile implements Store.
func (s *PGStore) FindProfile(ctx context.Context, c Claims) (Me, error) {
	var m Me
	err := s.db.QueryRow(ctx, `
		SELECT u.id::text, u.username, d.id::text, d.name
		FROM users u JOIN dealerships d ON d.id = u.dealership_id
		WHERE u.id = $1::uuid AND u.dealership_id = $2::uuid`,
		c.UserID, c.DealershipID).Scan(&m.UserID, &m.Username, &m.DealershipID, &m.DealershipName)
	if errors.Is(err, pgx.ErrNoRows) {
		return Me{}, ErrUserNotFound
	}
	if err != nil {
		return Me{}, fmt.Errorf("load profile: %w", err)
	}
	return m, nil
}
