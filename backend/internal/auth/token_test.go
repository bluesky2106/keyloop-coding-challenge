package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var testSecret = []byte(strings.Repeat("k", 32))

func TestTokenRoundTrip(t *testing.T) {
	tk := NewTokens(testSecret, time.Hour)
	token, expiresIn, err := tk.Issue(Claims{UserID: "u1", DealershipID: "d1"})
	if err != nil {
		t.Fatal(err)
	}
	if expiresIn != 3600 {
		t.Fatalf("expiresIn = %d, want 3600", expiresIn)
	}
	got, err := tk.Parse(token)
	if err != nil || got != (Claims{UserID: "u1", DealershipID: "d1"}) {
		t.Fatalf("Parse = %+v, %v", got, err)
	}
}

func TestParseRejects(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	tk := NewTokens(testSecret, time.Hour)
	tk.now = func() time.Time { return now }
	valid, _, err := tk.Issue(Claims{UserID: "u1", DealershipID: "d1"})
	if err != nil {
		t.Fatal(err)
	}

	otherKey := NewTokens([]byte(strings.Repeat("x", 32)), time.Hour)
	otherKey.now = tk.now
	forged, _, _ := otherKey.Issue(Claims{UserID: "u1", DealershipID: "d1"})

	none, err := jwt.NewWithClaims(jwt.SigningMethodNone, tokenClaims{
		DealershipID:     "d1",
		RegisteredClaims: jwt.RegisteredClaims{Subject: "u1", ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour))},
	}).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatal(err)
	}

	noExp, err := jwt.NewWithClaims(jwt.SigningMethodHS256, tokenClaims{
		DealershipID:     "d1",
		RegisteredClaims: jwt.RegisteredClaims{Subject: "u1"},
	}).SignedString(testSecret)
	if err != nil {
		t.Fatal(err)
	}

	noDealership, err := jwt.NewWithClaims(jwt.SigningMethodHS256, tokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{Subject: "u1", ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour))},
	}).SignedString(testSecret)
	if err != nil {
		t.Fatal(err)
	}

	later := NewTokens(testSecret, time.Hour)
	later.now = func() time.Time { return now.Add(2 * time.Hour) }

	tests := []struct {
		name   string
		tokens *Tokens
		token  string
	}{
		{"garbage", tk, "not-a-token"},
		{"wrong key", tk, forged},
		{"alg none", tk, none},
		{"no expiry", tk, noExp},
		{"no dealership claim", tk, noDealership},
		{"expired", later, valid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := tt.tokens.Parse(tt.token); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}
