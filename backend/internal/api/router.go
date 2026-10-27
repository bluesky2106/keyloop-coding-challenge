// Package api assembles the HTTP router from the feature packages.
package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/bluesky2106/keyloop-coding-challenge/backend/internal/actions"
	"github.com/bluesky2106/keyloop-coding-challenge/backend/internal/auth"
	"github.com/bluesky2106/keyloop-coding-challenge/backend/internal/catalog"
	"github.com/bluesky2106/keyloop-coding-challenge/backend/internal/inventory"
	"github.com/bluesky2106/keyloop-coding-challenge/backend/internal/platform"
)

// Deps are the collaborators the routes are built from.
type Deps struct {
	Tokens    *auth.Tokens
	Auth      auth.Authenticator
	Catalog   catalog.Store
	Inventory inventory.Reader
	Actions   actions.Logger
}

// NewRouter returns the full HTTP handler: the health check plus /api/v1.
// Every route except login needs a valid token and lives inside the Authenticate group.
func NewRouter(d Deps) http.Handler {
	r := platform.NewRouter()
	authH := auth.NewHandler(d.Auth)
	catalogH := catalog.NewHandler(d.Catalog)
	inventoryH := inventory.NewHandler(d.Inventory)
	actionsH := actions.NewHandler(d.Actions)

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/login", authH.Login)

		r.Group(func(r chi.Router) {
			r.Use(auth.Authenticate(d.Tokens))
			r.Get("/me", authH.Me)
			r.Get("/makes", catalogH.List)
			r.Get("/vehicles", inventoryH.List)
			r.Get("/vehicles/summary", inventoryH.Summary)
			r.Get("/vehicles/{vehicleId}", inventoryH.Get)
			r.Get("/vehicles/{vehicleId}/actions", actionsH.List)
			r.Post("/vehicles/{vehicleId}/actions", actionsH.Create)
		})
	})
	return r
}
