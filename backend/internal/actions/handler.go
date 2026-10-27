package actions

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"

	"github.com/bluesky2106/keyloop-coding-challenge/backend/internal/auth"
	"github.com/bluesky2106/keyloop-coding-challenge/backend/internal/platform/httpx"
)

const maxNoteLen = 500

// validTypes must match the CHECK constraint on vehicle_actions.action_type and the
// ActionType enum in docs/api/openapi.yaml.
var validTypes = map[string]bool{
	"price_reduction_planned": true,
	"promotion_planned":       true,
	"transfer_planned":        true,
	"wholesale_planned":       true,
	"hold":                    true,
}

// Logger is what the handlers need from the service.
type Logger interface {
	List(ctx context.Context, dealershipID, vehicleID string) ([]Action, error)
	Create(ctx context.Context, n NewAction) (Action, error)
}

// Handler serves the action endpoints.
type Handler struct{ svc Logger }

// NewHandler returns a Handler for svc.
func NewHandler(svc Logger) *Handler { return &Handler{svc: svc} }

// List handles GET /vehicles/{vehicleId}/actions.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	claims, vehicleID, ok := callerAndVehicle(w, r)
	if !ok {
		return
	}
	items, err := h.svc.List(r.Context(), claims.DealershipID, vehicleID)
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, struct {
		Items []Action `json:"items"`
	}{Items: items})
}

type createRequest struct {
	ActionType string  `json:"action_type"`
	Note       *string `json:"note"`
}

// Create handles POST /vehicles/{vehicleId}/actions.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	claims, vehicleID, ok := callerAndVehicle(w, r)
	if !ok {
		return
	}
	var req createRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.ValidationError(w, "request body must be a JSON object with action_type and an optional note", nil)
		return
	}
	note, fields := validate(req)
	if len(fields) > 0 {
		httpx.ValidationError(w, "invalid action", fields)
		return
	}

	a, err := h.svc.Create(r.Context(), NewAction{
		DealershipID: claims.DealershipID,
		UserID:       claims.UserID,
		VehicleID:    vehicleID,
		ActionType:   req.ActionType,
		Note:         note,
	})
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, a)
}

// validate checks the request and returns the note to store: trimmed, and nil when blank.
func validate(req createRequest) (*string, map[string]string) {
	fields := map[string]string{}
	if !validTypes[req.ActionType] {
		fields["action_type"] = "must be one of price_reduction_planned, promotion_planned, " +
			"transfer_planned, wholesale_planned, hold"
	}
	var note *string
	if req.Note != nil {
		trimmed := strings.TrimSpace(*req.Note)
		switch {
		case utf8.RuneCountInString(trimmed) > maxNoteLen:
			fields["note"] = "must be at most 500 characters"
		case strings.ContainsRune(trimmed, 0):
			// Postgres text cannot hold NUL; without this the insert fails with a 500.
			fields["note"] = "must not contain NUL characters"
		}
		if trimmed != "" {
			note = &trimmed
		}
	}
	return note, fields
}

// callerAndVehicle returns the authenticated caller and the vehicle id from the path. It
// writes the error response itself and reports false when either is unusable.
func callerAndVehicle(w http.ResponseWriter, r *http.Request) (auth.Claims, string, bool) {
	claims, ok := auth.ClaimsFrom(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "missing or invalid access token")
		return auth.Claims{}, "", false
	}
	id := chi.URLParam(r, "vehicleId")
	if !httpx.IsUUID(id) {
		httpx.ValidationError(w, "invalid vehicle id", map[string]string{"vehicleId": "must be a UUID"})
		return auth.Claims{}, "", false
	}
	return claims, id, true
}

func writeErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "not_found", "vehicle not found")
	case errors.Is(err, ErrNotAging):
		httpx.Error(w, http.StatusConflict, "vehicle_not_aging", "only aging vehicles accept actions")
	default:
		httpx.Internal(w, err)
	}
}
