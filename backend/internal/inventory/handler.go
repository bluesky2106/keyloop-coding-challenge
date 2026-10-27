package inventory

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/bluesky2106/keyloop-coding-challenge/backend/internal/auth"
	"github.com/bluesky2106/keyloop-coding-challenge/backend/internal/platform/httpx"
)

const (
	defaultPageSize = 25
	maxPageSize     = 100
)

// Reader is what the handlers need from the service.
type Reader interface {
	List(ctx context.Context, dealershipID string, f Filter) (Page, error)
	Get(ctx context.Context, dealershipID, id string) (Vehicle, error)
	Summary(ctx context.Context, dealershipID string) (Summary, error)
}

// Handler serves the inventory endpoints.
type Handler struct{ svc Reader }

// NewHandler returns a Handler for svc.
func NewHandler(svc Reader) *Handler { return &Handler{svc: svc} }

// List handles GET /vehicles.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFrom(r.Context())
	if !ok {
		unauthorized(w)
		return
	}
	f, fields := parseFilter(r.URL.Query())
	if len(fields) > 0 {
		httpx.ValidationError(w, "invalid query parameters", fields)
		return
	}
	page, err := h.svc.List(r.Context(), claims.DealershipID, f)
	if err != nil {
		httpx.Internal(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, page)
}

// Summary handles GET /vehicles/summary.
func (h *Handler) Summary(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFrom(r.Context())
	if !ok {
		unauthorized(w)
		return
	}
	s, err := h.svc.Summary(r.Context(), claims.DealershipID)
	if err != nil {
		httpx.Internal(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, s)
}

// Get handles GET /vehicles/{vehicleId}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFrom(r.Context())
	if !ok {
		unauthorized(w)
		return
	}
	id := chi.URLParam(r, "vehicleId")
	if !httpx.IsUUID(id) {
		httpx.ValidationError(w, "invalid vehicle id", map[string]string{"vehicleId": "must be a UUID"})
		return
	}
	v, err := h.svc.Get(r.Context(), claims.DealershipID, id)
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "not_found", "vehicle not found")
	case err != nil:
		httpx.Internal(w, err)
	default:
		httpx.JSON(w, http.StatusOK, v)
	}
}

func unauthorized(w http.ResponseWriter) {
	httpx.Error(w, http.StatusUnauthorized, "unauthorized", "missing or invalid access token")
}

// parseFilter reads and validates the list query string, applying the defaults.
func parseFilter(q url.Values) (Filter, map[string]string) {
	f := Filter{Sort: SortAgeDesc, Page: 1, PageSize: defaultPageSize}
	fields := map[string]string{}

	for name, dst := range map[string]*string{"make_id": &f.MakeID, "model_id": &f.ModelID} {
		if v := q.Get(name); v != "" {
			if !httpx.IsUUID(v) {
				fields[name] = "must be a UUID"
			}
			*dst = v
		}
	}

	if v := q.Get("aging"); v != "" {
		switch v {
		case "true", "false":
			b := v == "true"
			f.Aging = &b
		default:
			fields["aging"] = "must be true or false"
		}
	}

	f.MinAgeDays = parseAge(q, "min_age_days", fields)
	f.MaxAgeDays = parseAge(q, "max_age_days", fields)
	if f.MinAgeDays != nil && f.MaxAgeDays != nil && *f.MinAgeDays > *f.MaxAgeDays {
		fields["min_age_days"] = "must not be greater than max_age_days"
	}

	if v := q.Get("sort"); v != "" {
		if _, ok := orderBy[v]; !ok {
			fields["sort"] = "must be one of age_desc, age_asc, price_desc, price_asc"
		}
		f.Sort = v
	}

	if v := q.Get("page"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			fields["page"] = "must be an integer of at least 1"
		}
		f.Page = n
	}
	if v := q.Get("page_size"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > maxPageSize {
			fields["page_size"] = "must be an integer between 1 and 100"
		}
		f.PageSize = n
	}
	return f, fields
}

func parseAge(q url.Values, name string, fields map[string]string) *int {
	v := q.Get(name)
	if v == "" {
		return nil
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		fields[name] = "must be a non-negative integer"
		return nil
	}
	return &n
}
