// Package api assembles the HTTP router from the feature packages.
package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/bluesky2106/keyloop-coding-challenge/backend/internal/auth"
	"github.com/bluesky2106/keyloop-coding-challenge/backend/internal/platform"
)

// NewRouter returns the full HTTP handler: the health check plus /api/v1.
// Routes that need a signed-in user go inside the Authenticate group.
func NewRouter(authSvc *auth.Service, tokens *auth.Tokens) http.Handler {
	r := platform.NewRouter()
	authH := auth.NewHandler(authSvc)

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/login", authH.Login)

		r.Group(func(r chi.Router) {
			r.Use(auth.Authenticate(tokens))
			r.Get("/me", authH.Me)
		})
	})
	return r
}
