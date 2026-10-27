package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func post(t *testing.T, h http.HandlerFunc, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h(rec, httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/login", strings.NewReader(body)))
	return rec
}

func errorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var e struct {
		Error struct {
			Code   string            `json:"code"`
			Fields map[string]string `json:"fields"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &e), rec.Body.String())
	return e.Error.Code
}

func TestHandlerLogin(t *testing.T) {
	valid := `{"username":"  alice  ","password":"secret"}`

	t.Run("success trims the username", func(t *testing.T) {
		svc := NewMockAuthenticator(t)
		svc.EXPECT().Login(mock.Anything, "alice", "secret").
			Return(LoginResult{AccessToken: "tok", ExpiresIn: 60}, nil)

		rec := post(t, NewHandler(svc).Login, valid)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.JSONEq(t, `{"access_token":"tok","token_type":"Bearer","expires_in":60}`, rec.Body.String())
	})

	t.Run("invalid credentials", func(t *testing.T) {
		svc := NewMockAuthenticator(t)
		svc.EXPECT().Login(mock.Anything, "alice", "secret").Return(LoginResult{}, ErrInvalidCredentials)

		rec := post(t, NewHandler(svc).Login, valid)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Equal(t, "invalid_credentials", errorCode(t, rec))
	})

	t.Run("unexpected failure is a generic 500", func(t *testing.T) {
		svc := NewMockAuthenticator(t)
		svc.EXPECT().Login(mock.Anything, "alice", "secret").
			Return(LoginResult{}, errors.New("db password=hunter2 refused"))

		rec := post(t, NewHandler(svc).Login, valid)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Equal(t, "internal_error", errorCode(t, rec))
		assert.NotContains(t, rec.Body.String(), "hunter2", "internal detail must not leak")
	})

	invalid := map[string]struct{ body, field string }{
		"missing username": {`{"password":"x"}`, "username"},
		"blank username":   {`{"username":"  ","password":"x"}`, "username"},
		"long username":    {`{"username":"` + strings.Repeat("a", 101) + `","password":"x"}`, "username"},
		"missing password": {`{"username":"u"}`, "password"},
		"long password":    {`{"username":"u","password":"` + strings.Repeat("a", 201) + `"}`, "password"},
	}
	for name, tt := range invalid {
		t.Run(name, func(t *testing.T) {
			rec := post(t, NewHandler(NewMockAuthenticator(t)).Login, tt.body)

			assert.Equal(t, http.StatusBadRequest, rec.Code)
			assert.Equal(t, "validation_error", errorCode(t, rec))
			assert.Contains(t, rec.Body.String(), `"`+tt.field+`"`)
		})
	}

	for name, body := range map[string]string{
		"not json": "nope", "unknown field": `{"username":"u","password":"p","x":1}`, "array": "[]",
	} {
		t.Run(name, func(t *testing.T) {
			rec := post(t, NewHandler(NewMockAuthenticator(t)).Login, body)

			assert.Equal(t, http.StatusBadRequest, rec.Code)
			assert.Equal(t, "validation_error", errorCode(t, rec))
		})
	}
}

func TestHandlerMe(t *testing.T) {
	claims := Claims{UserID: "u1", DealershipID: "d1"}
	get := func(t *testing.T, svc Authenticator, ctx context.Context) *httptest.ResponseRecorder {
		t.Helper()
		rec := httptest.NewRecorder()
		NewHandler(svc).Me(rec, httptest.NewRequestWithContext(ctx, http.MethodGet, "/me", nil))
		return rec
	}
	withClaims := WithClaims(context.Background(), claims)

	t.Run("success", func(t *testing.T) {
		svc := NewMockAuthenticator(t)
		svc.EXPECT().Me(mock.Anything, claims).
			Return(Me{UserID: "u1", Username: "alice", DealershipID: "d1", DealershipName: "Acme"}, nil)

		rec := get(t, svc, withClaims)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.JSONEq(t,
			`{"user":{"id":"u1","username":"alice"},"dealership":{"id":"d1","name":"Acme"}}`, rec.Body.String())
	})

	t.Run("no claims in the context", func(t *testing.T) {
		rec := get(t, NewMockAuthenticator(t), context.Background())
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("user no longer exists", func(t *testing.T) {
		svc := NewMockAuthenticator(t)
		svc.EXPECT().Me(mock.Anything, claims).Return(Me{}, ErrUserNotFound)

		rec := get(t, svc, withClaims)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Equal(t, "unauthorized", errorCode(t, rec))
	})

	t.Run("unexpected failure", func(t *testing.T) {
		svc := NewMockAuthenticator(t)
		svc.EXPECT().Me(mock.Anything, claims).Return(Me{}, errors.New("boom"))

		rec := get(t, svc, withClaims)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Equal(t, "internal_error", errorCode(t, rec))
	})
}
