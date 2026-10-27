// Package auth implements login, JWT issuing and verification, and request authentication.
package auth

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// ErrInvalidCredentials is returned for an unknown user and a wrong password alike, so
// callers cannot tell which usernames exist.
var ErrInvalidCredentials = errors.New("invalid credentials")

// ErrUserNotFound is returned when a valid token refers to a user that no longer exists.
var ErrUserNotFound = errors.New("user not found")

// Service authenticates users against the database.
type Service struct {
	db     *pgxpool.Pool
	tokens *Tokens
	// dummyHash is compared against when the username is unknown, so a failed login
	// costs the same time whether or not the user exists.
	dummyHash []byte
}

// NewService returns a Service backed by db that issues tokens with tokens.
func NewService(db *pgxpool.Pool, tokens *Tokens) (*Service, error) {
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		return nil, fmt.Errorf("random: %w", err)
	}
	dummy, err := bcrypt.GenerateFromPassword(random, bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("dummy hash: %w", err)
	}
	return &Service{db: db, tokens: tokens, dummyHash: dummy}, nil
}

// LoginResult is a freshly issued access token.
type LoginResult struct {
	AccessToken string
	ExpiresIn   int
}

// Login verifies the credentials and issues a token carrying the user's dealership.
func (s *Service) Login(ctx context.Context, username, password string) (LoginResult, error) {
	var id, dealershipID, hash string
	err := s.db.QueryRow(ctx,
		`SELECT id::text, dealership_id::text, password_hash FROM users WHERE lower(username) = lower($1)`,
		username).Scan(&id, &dealershipID, &hash)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		_ = bcrypt.CompareHashAndPassword(s.dummyHash, []byte(password))
		return LoginResult{}, ErrInvalidCredentials
	case err != nil:
		return LoginResult{}, fmt.Errorf("find user: %w", err)
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return LoginResult{}, ErrInvalidCredentials
	}

	token, expiresIn, err := s.tokens.Issue(Claims{UserID: id, DealershipID: dealershipID})
	if err != nil {
		return LoginResult{}, err
	}
	return LoginResult{AccessToken: token, ExpiresIn: expiresIn}, nil
}

// Me describes the authenticated user and their dealership.
type Me struct {
	UserID         string
	Username       string
	DealershipID   string
	DealershipName string
}

// Me loads the caller's profile. The query matches both the user and the dealership
// from the token, so a token never resolves to another tenant's data.
func (s *Service) Me(ctx context.Context, c Claims) (Me, error) {
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
