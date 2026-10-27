package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthenticate(t *testing.T) {
	tokens := NewTokens(testSecret, time.Hour)
	good, _, err := tokens.Issue(Claims{UserID: "u1", DealershipID: "d1"})
	require.NoError(t, err)

	var seen Claims
	var seenOK bool
	protected := Authenticate(tokens)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen, seenOK = ClaimsFrom(r.Context())
		w.WriteHeader(http.StatusNoContent)
	}))

	tests := []struct {
		name   string
		header string
		want   int
	}{
		{"valid token", "Bearer " + good, http.StatusNoContent},
		{"no header", "", http.StatusUnauthorized},
		{"wrong scheme", "Basic " + good, http.StatusUnauthorized},
		{"empty bearer", "Bearer ", http.StatusUnauthorized},
		{"invalid token", "Bearer nope", http.StatusUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seen, seenOK = Claims{}, false
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/x", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			rec := httptest.NewRecorder()

			protected.ServeHTTP(rec, req)

			assert.Equal(t, tt.want, rec.Code)
			if tt.want == http.StatusNoContent {
				assert.True(t, seenOK)
				assert.Equal(t, Claims{UserID: "u1", DealershipID: "d1"}, seen)
				return
			}
			assert.False(t, seenOK, "the handler must not run for unauthenticated requests")
			assert.Equal(t, "Bearer", rec.Header().Get("WWW-Authenticate"))
		})
	}
}

func TestClaimsFromOutsideAuthenticate(t *testing.T) {
	_, ok := ClaimsFrom(t.Context())
	assert.False(t, ok)
}
