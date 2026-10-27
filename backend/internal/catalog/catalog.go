// Package catalog serves the shared make and model reference data.
package catalog

import (
	"context"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bluesky2106/keyloop-coding-challenge/backend/internal/platform/httpx"
)

// NamedRef is a catalog entry (a make or a model) referenced by id and name.
type NamedRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Make is a manufacturer with its models.
type Make struct {
	ID     string     `json:"id"`
	Name   string     `json:"name"`
	Models []NamedRef `json:"models"`
}

// Store reads the catalog.
type Store interface {
	// ListMakes returns every make with its models, both sorted by name.
	ListMakes(ctx context.Context) ([]Make, error)
}

// PGStore is the PostgreSQL implementation of Store.
type PGStore struct{ db *pgxpool.Pool }

// NewPGStore returns a Store backed by db.
func NewPGStore(db *pgxpool.Pool) *PGStore { return &PGStore{db: db} }

// ListMakes implements Store.
func (s *PGStore) ListMakes(ctx context.Context) ([]Make, error) {
	rows, err := s.db.Query(ctx, `
		SELECT k.id::text, k.name, m.id::text, m.name
		FROM makes k JOIN models m ON m.make_id = k.id
		ORDER BY lower(k.name), k.id, lower(m.name), m.id`)
	if err != nil {
		return nil, fmt.Errorf("list makes: %w", err)
	}
	defer rows.Close()

	makes := []Make{}
	for rows.Next() {
		var makeID, makeName string
		var model NamedRef
		if err := rows.Scan(&makeID, &makeName, &model.ID, &model.Name); err != nil {
			return nil, fmt.Errorf("scan make: %w", err)
		}
		if n := len(makes); n == 0 || makes[n-1].ID != makeID {
			makes = append(makes, Make{ID: makeID, Name: makeName, Models: []NamedRef{}})
		}
		last := &makes[len(makes)-1]
		last.Models = append(last.Models, model)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list makes: %w", err)
	}
	return makes, nil
}

// Handler serves GET /makes.
type Handler struct{ store Store }

// NewHandler returns a Handler reading from store.
func NewHandler(store Store) *Handler { return &Handler{store: store} }

// List handles GET /makes.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	makes, err := h.store.ListMakes(r.Context())
	if err != nil {
		httpx.Internal(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, struct {
		Items []Make `json:"items"`
	}{Items: makes})
}
