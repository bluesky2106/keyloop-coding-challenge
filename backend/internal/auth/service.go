// Package auth implements login, JWT issuing and verification, and request authentication.
package auth

import (
	"context"
	"errors"

	"golang.org/x/crypto/bcrypt"
)

// ErrInvalidCredentials is returned for an unknown user and a wrong password alike, so
// responses do not reveal which usernames exist. Response time is not equalised: an
// unknown username skips the bcrypt comparison, so rate-limit login before exposing it.
var ErrInvalidCredentials = errors.New("invalid credentials")

// ErrUserNotFound is returned by a Store when no user matches, and by Service.Me when a
// valid token refers to a user that no longer exists.
var ErrUserNotFound = errors.New("user not found")

// Service authenticates users against a Store.
type Service struct {
	store  Store
	tokens *Tokens
}

// NewService returns a Service that reads users from store and issues tokens with tokens.
func NewService(store Store, tokens *Tokens) *Service {
	return &Service{store: store, tokens: tokens}
}

// LoginResult is a freshly issued access token.
type LoginResult struct {
	AccessToken string
	ExpiresIn   int
}

// Login verifies the credentials and issues a token carrying the user's dealership.
func (s *Service) Login(ctx context.Context, username, password string) (LoginResult, error) {
	user, err := s.store.FindUserByUsername(ctx, username)
	switch {
	case errors.Is(err, ErrUserNotFound):
		return LoginResult{}, ErrInvalidCredentials
	case err != nil:
		return LoginResult{}, err
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return LoginResult{}, ErrInvalidCredentials
	}

	token, expiresIn, err := s.tokens.Issue(Claims{UserID: user.ID, DealershipID: user.DealershipID})
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

// Me loads the caller's profile.
func (s *Service) Me(ctx context.Context, c Claims) (Me, error) {
	return s.store.FindProfile(ctx, c)
}
