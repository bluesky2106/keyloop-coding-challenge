package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/bluesky2106/keyloop-coding-challenge/backend/internal/auth"
	"github.com/bluesky2106/keyloop-coding-challenge/backend/internal/seed"
	"github.com/bluesky2106/keyloop-coding-challenge/backend/internal/testdb"
)

var secret = []byte(strings.Repeat("t", 32))

type app struct {
	t       *testing.T
	handler http.Handler
}

// shared is built once in TestMain: every test only reads, so they can share one seeded
// database instead of paying for a fresh migration and seed each time.
var shared http.Handler

func TestMain(m *testing.M) {
	os.Exit(run(m))
}

func run(m *testing.M) int {
	pool, cleanup, err := testdb.Setup(true)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer cleanup()
	if err := seed.Load(context.Background(), pool, "../../seed"); err != nil {
		fmt.Fprintln(os.Stderr, "seed:", err)
		return 1
	}
	tokens := auth.NewTokens(secret, time.Hour)
	shared = NewRouter(auth.NewService(auth.NewPGStore(pool), tokens), tokens)
	return m.Run()
}

func newApp(t *testing.T) *app {
	t.Helper()
	return &app{t: t, handler: shared}
}

func (a *app) do(method, path, body, bearer string) *httptest.ResponseRecorder {
	a.t.Helper()
	req := httptest.NewRequestWithContext(a.t.Context(), method, path, strings.NewReader(body))
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	rec := httptest.NewRecorder()
	a.handler.ServeHTTP(rec, req)
	return rec
}

func (a *app) login(username, password string) string {
	a.t.Helper()
	rec := a.do(http.MethodPost, "/api/v1/auth/login",
		`{"username":"`+username+`","password":"`+password+`"}`, "")
	if rec.Code != http.StatusOK {
		a.t.Fatalf("login %s: status %d body %s", username, rec.Code, rec.Body)
	}
	var out struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		a.t.Fatal(err)
	}
	return out.AccessToken
}

type errBody struct {
	Error struct {
		Code   string            `json:"code"`
		Fields map[string]string `json:"fields"`
	} `json:"error"`
}

func decodeErr(t *testing.T, rec *httptest.ResponseRecorder) errBody {
	t.Helper()
	var e errBody
	if err := json.Unmarshal(rec.Body.Bytes(), &e); err != nil {
		t.Fatalf("decode error body %q: %v", rec.Body, err)
	}
	return e
}

func TestHealthz(t *testing.T) {
	a := newApp(t)
	if rec := a.do(http.MethodGet, "/healthz", "", ""); rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestLoginAndMe(t *testing.T) {
	a := newApp(t)

	rec := a.do(http.MethodPost, "/api/v1/auth/login",
		`{"username":"northside.manager","password":"demo-northside-1"}`, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("login status = %d body %s", rec.Code, rec.Body)
	}
	var login struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &login); err != nil {
		t.Fatal(err)
	}
	if login.TokenType != "Bearer" || login.ExpiresIn != 3600 || login.AccessToken == "" {
		t.Fatalf("unexpected login response: %+v", login)
	}

	rec = a.do(http.MethodGet, "/api/v1/me", "", login.AccessToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("me status = %d body %s", rec.Code, rec.Body)
	}
	var me struct {
		User       struct{ ID, Username string } `json:"user"`
		Dealership struct{ ID, Name string }     `json:"dealership"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &me); err != nil {
		t.Fatal(err)
	}
	if me.User.Username != "northside.manager" || me.Dealership.Name != "Northside Motors" ||
		me.User.ID == "" || me.Dealership.ID == "" {
		t.Fatalf("unexpected me: %+v", me)
	}
}

func TestUsernameIsCaseInsensitive(t *testing.T) {
	a := newApp(t)
	a.login("NorthSide.Manager", "demo-northside-1")
}

func TestUsersSeeOnlyTheirOwnDealership(t *testing.T) {
	a := newApp(t)
	for user, pass := range map[string]string{
		"northside.sales": "demo-northside-2", "summit.sales": "demo-summit-2",
	} {
		token := a.login(user, pass)
		rec := a.do(http.MethodGet, "/api/v1/me", "", token)
		want := map[string]string{"northside.sales": "Northside Motors", "summit.sales": "Summit Cars"}[user]
		if !strings.Contains(rec.Body.String(), `"name":"`+want+`"`) {
			t.Fatalf("%s: body %s, want dealership %s", user, rec.Body, want)
		}
	}
}

func TestLoginFailures(t *testing.T) {
	a := newApp(t)

	wrongPass := a.do(http.MethodPost, "/api/v1/auth/login",
		`{"username":"northside.manager","password":"nope"}`, "")
	unknown := a.do(http.MethodPost, "/api/v1/auth/login",
		`{"username":"ghost","password":"nope"}`, "")
	for name, rec := range map[string]*httptest.ResponseRecorder{"wrong password": wrongPass, "unknown user": unknown} {
		if rec.Code != http.StatusUnauthorized || decodeErr(t, rec).Error.Code != "invalid_credentials" {
			t.Fatalf("%s: status %d body %s", name, rec.Code, rec.Body)
		}
	}
	if wrongPass.Body.String() != unknown.Body.String() {
		t.Fatal("responses differ between wrong password and unknown user: usernames can be enumerated")
	}
}

func TestLoginValidation(t *testing.T) {
	a := newApp(t)
	tests := []struct {
		name, body, field string
	}{
		{"empty username", `{"username":"","password":"x"}`, "username"},
		{"missing password", `{"username":"u"}`, "password"},
		{"blank username", `{"username":"   ","password":"x"}`, "username"},
		{"long username", `{"username":"` + strings.Repeat("a", 101) + `","password":"x"}`, "username"},
		{"long password", `{"username":"u","password":"` + strings.Repeat("a", 201) + `"}`, "password"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := a.do(http.MethodPost, "/api/v1/auth/login", tt.body, "")
			e := decodeErr(t, rec)
			if rec.Code != http.StatusBadRequest || e.Error.Code != "validation_error" || e.Error.Fields[tt.field] == "" {
				t.Fatalf("status %d body %s", rec.Code, rec.Body)
			}
		})
	}
	for name, body := range map[string]string{
		"not json": `nope`, "unknown field": `{"username":"u","password":"p","admin":true}`, "array": `[]`,
	} {
		rec := a.do(http.MethodPost, "/api/v1/auth/login", body, "")
		if rec.Code != http.StatusBadRequest || decodeErr(t, rec).Error.Code != "validation_error" {
			t.Fatalf("%s: status %d body %s", name, rec.Code, rec.Body)
		}
	}
}

func TestMeRejectsBadTokens(t *testing.T) {
	a := newApp(t)
	good := a.login("northside.manager", "demo-northside-1")
	claims, err := auth.NewTokens(secret, time.Hour).Parse(good)
	if err != nil {
		t.Fatal(err)
	}
	expired, _, err := auth.NewTokens(secret, -time.Minute).Issue(claims)
	if err != nil {
		t.Fatal(err)
	}
	wrongKey, _, err := auth.NewTokens([]byte(strings.Repeat("x", 32)), time.Hour).Issue(claims)
	if err != nil {
		t.Fatal(err)
	}
	// Valid signature, but no such user: must not authenticate.
	ghost, _, err := auth.NewTokens(secret, time.Hour).Issue(auth.Claims{
		UserID:       "00000000-0000-0000-0000-000000000001",
		DealershipID: claims.DealershipID,
	})
	if err != nil {
		t.Fatal(err)
	}
	// Valid user but a different dealership than the user's own: must not authenticate.
	mismatch, _, err := auth.NewTokens(secret, time.Hour).Issue(auth.Claims{
		UserID:       claims.UserID,
		DealershipID: "00000000-0000-0000-0000-000000000002",
	})
	if err != nil {
		t.Fatal(err)
	}

	for name, token := range map[string]string{
		"no token": "", "garbage": "abc", "expired": expired, "wrong key": wrongKey,
		"unknown user": ghost, "dealership mismatch": mismatch,
	} {
		rec := a.do(http.MethodGet, "/api/v1/me", "", token)
		if rec.Code != http.StatusUnauthorized || decodeErr(t, rec).Error.Code != "unauthorized" {
			t.Fatalf("%s: status %d body %s", name, rec.Code, rec.Body)
		}
		if rec.Header().Get("WWW-Authenticate") != "Bearer" {
			t.Fatalf("%s: missing WWW-Authenticate header", name)
		}
	}
}
