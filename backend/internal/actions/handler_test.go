package actions

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/bluesky2106/keyloop-coding-challenge/backend/internal/auth"
)

const (
	vehicleID = "3f2b8c1e-9a4d-4e7b-8c5a-1d2e3f4a5b6c"
	basePath  = "/vehicles/" + vehicleID + "/actions"
)

var caller = auth.Claims{UserID: "user-1", DealershipID: "dealer-1"}

// serve routes a request through a real chi router so {vehicleId} is populated. When
// signedIn is false the claims are absent, as if Authenticate had been skipped.
func serve(t *testing.T, svc Logger, signedIn bool, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	h := NewHandler(svc)
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if signedIn {
				req = req.WithContext(auth.WithClaims(req.Context(), caller))
			}
			next.ServeHTTP(w, req)
		})
	})
	r.Get("/vehicles/{vehicleId}/actions", h.List)
	r.Post("/vehicles/{vehicleId}/actions", h.Create)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body)))
	return rec
}

func TestHandlerList(t *testing.T) {
	created := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	note := "drop by 5%"

	t.Run("returns the history in the items envelope", func(t *testing.T) {
		svc := NewMockLogger(t)
		svc.EXPECT().List(mock.Anything, "dealer-1", vehicleID).Return([]Action{
			{ID: "a1", VehicleID: vehicleID, ActionType: "hold", Note: &note, CreatedBy: "alice", CreatedAt: created},
		}, nil)

		rec := serve(t, svc, true, http.MethodGet, basePath, "")

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.JSONEq(t, `{"items":[{"id":"a1","vehicle_id":"`+vehicleID+`","action_type":"hold",
			"note":"drop by 5%","created_by":"alice","created_at":"2026-01-02T03:04:05Z"}]}`, rec.Body.String())
	})

	t.Run("unknown or foreign vehicle", func(t *testing.T) {
		svc := NewMockLogger(t)
		svc.EXPECT().List(mock.Anything, mock.Anything, mock.Anything).Return(nil, ErrNotFound)

		rec := serve(t, svc, true, http.MethodGet, basePath, "")

		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Contains(t, rec.Body.String(), `"not_found"`)
	})

	t.Run("unexpected failure", func(t *testing.T) {
		svc := NewMockLogger(t)
		svc.EXPECT().List(mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("boom"))

		rec := serve(t, svc, true, http.MethodGet, basePath, "")

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.NotContains(t, rec.Body.String(), "boom")
	})

	t.Run("malformed vehicle id", func(t *testing.T) {
		rec := serve(t, NewMockLogger(t), true, http.MethodGet, "/vehicles/nope/actions", "")

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "validation_error")
	})

	t.Run("not signed in", func(t *testing.T) {
		rec := serve(t, NewMockLogger(t), false, http.MethodGet, basePath, "")
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}

func TestHandlerCreate(t *testing.T) {
	created := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	trimmed := "drop by 5%"

	t.Run("records the action as the caller, trimming the note", func(t *testing.T) {
		svc := NewMockLogger(t)
		svc.EXPECT().Create(mock.Anything, NewAction{
			DealershipID: "dealer-1", UserID: "user-1", VehicleID: vehicleID,
			ActionType: "price_reduction_planned", Note: &trimmed,
		}).Return(Action{
			ID: "a1", VehicleID: vehicleID, ActionType: "price_reduction_planned",
			Note: &trimmed, CreatedBy: "alice", CreatedAt: created,
		}, nil)

		rec := serve(t, svc, true, http.MethodPost, basePath,
			`{"action_type":"price_reduction_planned","note":"  drop by 5%  "}`)

		assert.Equal(t, http.StatusCreated, rec.Code)
		assert.Contains(t, rec.Body.String(), `"created_by":"alice"`)
	})

	t.Run("a blank or missing note is stored as null", func(t *testing.T) {
		for _, body := range []string{
			`{"action_type":"hold"}`, `{"action_type":"hold","note":null}`, `{"action_type":"hold","note":"   "}`,
		} {
			svc := NewMockLogger(t)
			svc.EXPECT().Create(mock.Anything, NewAction{
				DealershipID: "dealer-1", UserID: "user-1", VehicleID: vehicleID, ActionType: "hold",
			}).Return(Action{ID: "a1"}, nil)

			rec := serve(t, svc, true, http.MethodPost, basePath, body)

			assert.Equal(t, http.StatusCreated, rec.Code, body)
		}
	})

	t.Run("a 500 character note is accepted, 501 is not", func(t *testing.T) {
		long := strings.Repeat("é", 500) // counted in characters, not bytes
		svc := NewMockLogger(t)
		svc.EXPECT().Create(mock.Anything, mock.Anything).Return(Action{ID: "a1"}, nil).Once()
		assert.Equal(t, http.StatusCreated,
			serve(t, svc, true, http.MethodPost, basePath, `{"action_type":"hold","note":"`+long+`"}`).Code)

		rec := serve(t, NewMockLogger(t), true, http.MethodPost, basePath,
			`{"action_type":"hold","note":"`+long+`x"}`)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), `"note"`)
	})

	t.Run("every action type is accepted", func(t *testing.T) {
		for typ := range validTypes {
			svc := NewMockLogger(t)
			svc.EXPECT().Create(mock.Anything, mock.Anything).Return(Action{ID: "a1"}, nil)
			rec := serve(t, svc, true, http.MethodPost, basePath, `{"action_type":"`+typ+`"}`)
			assert.Equal(t, http.StatusCreated, rec.Code, typ)
		}
	})

	invalid := map[string]struct{ body, field string }{
		"unknown action type": {`{"action_type":"bogus"}`, "action_type"},
		"missing action type": {`{"note":"x"}`, "action_type"},
		"empty action type":   {`{"action_type":""}`, "action_type"},
		"NUL in note":         {`{"action_type":"hold","note":"a\u0000b"}`, "note"},
	}
	for name, tt := range invalid {
		t.Run(name, func(t *testing.T) {
			rec := serve(t, NewMockLogger(t), true, http.MethodPost, basePath, tt.body)

			assert.Equal(t, http.StatusBadRequest, rec.Code)
			assert.Contains(t, rec.Body.String(), `"`+tt.field+`"`)
		})
	}

	for name, body := range map[string]string{
		"not json": "nope", "unknown field": `{"action_type":"hold","vehicle_id":"x"}`, "array": "[]",
	} {
		t.Run(name, func(t *testing.T) {
			rec := serve(t, NewMockLogger(t), true, http.MethodPost, basePath, body)

			assert.Equal(t, http.StatusBadRequest, rec.Code)
			assert.Contains(t, rec.Body.String(), "validation_error")
		})
	}

	t.Run("vehicle is not aging", func(t *testing.T) {
		svc := NewMockLogger(t)
		svc.EXPECT().Create(mock.Anything, mock.Anything).Return(Action{}, ErrNotAging)

		rec := serve(t, svc, true, http.MethodPost, basePath, `{"action_type":"hold"}`)

		assert.Equal(t, http.StatusConflict, rec.Code)
		assert.Contains(t, rec.Body.String(), `"vehicle_not_aging"`)
	})

	t.Run("unknown or foreign vehicle", func(t *testing.T) {
		svc := NewMockLogger(t)
		svc.EXPECT().Create(mock.Anything, mock.Anything).Return(Action{}, ErrNotFound)

		rec := serve(t, svc, true, http.MethodPost, basePath, `{"action_type":"hold"}`)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("unexpected failure", func(t *testing.T) {
		svc := NewMockLogger(t)
		svc.EXPECT().Create(mock.Anything, mock.Anything).Return(Action{}, errors.New("boom"))

		rec := serve(t, svc, true, http.MethodPost, basePath, `{"action_type":"hold"}`)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("malformed vehicle id never reaches the service", func(t *testing.T) {
		rec := serve(t, NewMockLogger(t), true, http.MethodPost, "/vehicles/nope/actions", `{"action_type":"hold"}`)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("not signed in", func(t *testing.T) {
		rec := serve(t, NewMockLogger(t), false, http.MethodPost, basePath, `{"action_type":"hold"}`)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}
