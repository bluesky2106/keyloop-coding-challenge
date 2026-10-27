package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/bluesky2106/keyloop-coding-challenge/backend/internal/platform/httpx"
)

type ctxKey struct{}

// ClaimsFrom returns the authenticated caller. The second result is false outside
// an Authenticate-protected route.
func ClaimsFrom(ctx context.Context) (Claims, bool) {
	c, ok := ctx.Value(ctxKey{}).(Claims)
	return c, ok
}

// Authenticate rejects requests without a valid Bearer token and stores the claims
// in the request context.
func Authenticate(tokens *Tokens) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw, ok := bearerToken(r.Header.Get("Authorization"))
			if !ok {
				unauthorized(w)
				return
			}
			claims, err := tokens.Parse(raw)
			if err != nil {
				unauthorized(w)
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, claims)))
		})
	}
}

// bearerToken extracts the token from an "Authorization: Bearer <token>" header. The
// scheme name is case-insensitive (RFC 7235).
func bearerToken(header string) (string, bool) {
	scheme, token, found := strings.Cut(header, " ")
	token = strings.TrimSpace(token)
	if !found || !strings.EqualFold(scheme, "Bearer") || token == "" {
		return "", false
	}
	return token, true
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", "Bearer")
	httpx.Error(w, http.StatusUnauthorized, "unauthorized", "missing or invalid access token")
}
