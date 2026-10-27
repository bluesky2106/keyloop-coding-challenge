package httpx

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	JSON(rec, http.StatusCreated, map[string]int{"n": 1})

	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	assert.JSONEq(t, `{"n":1}`, rec.Body.String())
}

func TestJSONWithUnencodableValueKeepsTheStatus(t *testing.T) {
	rec := httptest.NewRecorder()
	JSON(rec, http.StatusOK, make(chan int))
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestError(t *testing.T) {
	rec := httptest.NewRecorder()
	Error(rec, http.StatusNotFound, "not_found", "no such vehicle")

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.JSONEq(t, `{"error":{"code":"not_found","message":"no such vehicle"}}`, rec.Body.String())
}

func TestValidationError(t *testing.T) {
	rec := httptest.NewRecorder()
	ValidationError(rec, "bad input", map[string]string{"name": "is required"})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.JSONEq(t,
		`{"error":{"code":"validation_error","message":"bad input","fields":{"name":"is required"}}}`,
		rec.Body.String())
}

func TestInternalHidesTheCause(t *testing.T) {
	rec := httptest.NewRecorder()
	Internal(rec, assert.AnError)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.NotContains(t, rec.Body.String(), assert.AnError.Error())
	assert.Contains(t, rec.Body.String(), "internal_error")
}

func TestDecodeJSON(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}
	decode := func(body string) (payload, error) {
		var p payload
		r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", strings.NewReader(body))
		return p, DecodeJSON(httptest.NewRecorder(), r, &p)
	}

	p, err := decode(`{"name":"x"}`)
	require.NoError(t, err)
	assert.Equal(t, "x", p.Name)

	_, err = decode(`{"name":"x"}{"name":"y"}`)
	require.Error(t, err, "a second JSON value must be rejected")

	_, err = decode(`{"name":"x"} garbage`)
	require.Error(t, err, "trailing data must be rejected")

	_, err = decode("{\"name\":\"x\"}\n  \n")
	require.NoError(t, err, "trailing whitespace is fine")

	_, err = decode(`{"name":"x","extra":1}`)
	require.Error(t, err, "unknown fields must be rejected")

	_, err = decode(`{"name":"` + strings.Repeat("a", 1<<20) + `"}`)
	require.Error(t, err, "bodies over 1 MiB must be rejected")
}

func TestIsUUID(t *testing.T) {
	valid := []string{
		"00000000-0000-0000-0000-000000000000",
		"3f2b8c1e-9a4d-4e7b-8c5a-1d2e3f4a5b6c",
		"3F2B8C1E-9A4D-4E7B-8C5A-1D2E3F4A5B6C",
	}
	for _, s := range valid {
		assert.True(t, IsUUID(s), s)
	}
	invalid := []string{
		"", "not-a-uuid", "3f2b8c1e9a4d4e7b8c5a1d2e3f4a5b6c",
		"3f2b8c1e-9a4d-4e7b-8c5a-1d2e3f4a5b6",   // too short
		"3f2b8c1e-9a4d-4e7b-8c5a-1d2e3f4a5b6cc", // too long
		"3f2b8c1e-9a4d-4e7b-8c5a-1d2e3f4a5b6g",  // not hex
		"3f2b8c1e_9a4d-4e7b-8c5a-1d2e3f4a5b6c",  // wrong separator
	}
	for _, s := range invalid {
		assert.False(t, IsUUID(s), s)
	}
}
