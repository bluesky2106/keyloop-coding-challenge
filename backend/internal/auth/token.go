package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ErrInvalidToken is returned for any token that is malformed, forged or expired.
var ErrInvalidToken = errors.New("invalid token")

// Claims identify the caller. DealershipID is the only source of tenant scope.
type Claims struct {
	UserID       string
	DealershipID string
}

type tokenClaims struct {
	DealershipID string `json:"did"`
	jwt.RegisteredClaims
}

// Tokens issues and verifies HS256 JWTs.
type Tokens struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

// NewTokens returns a Tokens that signs with secret and issues tokens valid for ttl.
func NewTokens(secret []byte, ttl time.Duration) *Tokens {
	return &Tokens{secret: secret, ttl: ttl, now: time.Now}
}

// Issue signs a token for the given user and returns it with its lifetime in seconds.
func (t *Tokens) Issue(c Claims) (token string, expiresIn int, err error) {
	now := t.now()
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, tokenClaims{
		DealershipID: c.DealershipID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   c.UserID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(t.ttl)),
		},
	}).SignedString(t.secret)
	if err != nil {
		return "", 0, fmt.Errorf("sign token: %w", err)
	}
	return signed, int(t.ttl.Seconds()), nil
}

// Parse verifies the signature (HS256 only) and expiry and returns the claims.
func (t *Tokens) Parse(token string) (Claims, error) {
	var tc tokenClaims
	_, err := jwt.ParseWithClaims(token, &tc,
		func(*jwt.Token) (any, error) { return t.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(t.now),
	)
	if err != nil || tc.Subject == "" || tc.DealershipID == "" {
		return Claims{}, ErrInvalidToken
	}
	return Claims{UserID: tc.Subject, DealershipID: tc.DealershipID}, nil
}
