// Package platform holds shared HTTP and infrastructure wiring.
package platform

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// NewRouter returns the root router with the routes common to every deployment.
func NewRouter() chi.Router {
	r := chi.NewRouter()
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	return r
}
