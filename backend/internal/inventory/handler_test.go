package inventory

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/bluesky2106/keyloop-coding-challenge/backend/internal/actions"
	"github.com/bluesky2106/keyloop-coding-challenge/backend/internal/auth"
	"github.com/bluesky2106/keyloop-coding-challenge/backend/internal/catalog"
)

const (
	vehicleID = "3f2b8c1e-9a4d-4e7b-8c5a-1d2e3f4a5b6c"
	makeID    = "11111111-1111-1111-1111-111111111111"
	modelID   = "22222222-2222-2222-2222-222222222222"
)

var caller = auth.Claims{UserID: "user-1", DealershipID: "dealer-1"}

func serve(t *testing.T, svc Reader, signedIn bool, target string) *httptest.ResponseRecorder {
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
	r.Get("/vehicles", h.List)
	r.Get("/vehicles/summary", h.Summary)
	r.Get("/vehicles/{vehicleId}", h.Get)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, target, nil))
	return rec
}

func ptr[T any](v T) *T { return &v }

func TestHandlerListParsesTheQuery(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  Filter
	}{
		{"defaults", "", Filter{Sort: SortAgeDesc, Page: 1, PageSize: 25}},
		{
			"every filter",
			"?make_id=" + makeID + "&model_id=" + modelID + "&aging=true&min_age_days=10&max_age_days=200" +
				"&sort=price_asc&page=3&page_size=50",
			Filter{
				MakeID: makeID, ModelID: modelID, Aging: ptr(true), MinAgeDays: ptr(10), MaxAgeDays: ptr(200),
				Sort: SortPriceAsc, Page: 3, PageSize: 50,
			},
		},
		{"not aging", "?aging=false", Filter{Aging: ptr(false), Sort: SortAgeDesc, Page: 1, PageSize: 25}},
		{"age bounds of zero", "?min_age_days=0&max_age_days=0",
			Filter{MinAgeDays: ptr(0), MaxAgeDays: ptr(0), Sort: SortAgeDesc, Page: 1, PageSize: 25}},
		{"equal bounds", "?min_age_days=5&max_age_days=5",
			Filter{MinAgeDays: ptr(5), MaxAgeDays: ptr(5), Sort: SortAgeDesc, Page: 1, PageSize: 25}},
		{"largest page size", "?page_size=100", Filter{Sort: SortAgeDesc, Page: 1, PageSize: 100}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewMockReader(t)
			svc.EXPECT().List(mock.Anything, "dealer-1", tt.want).Return(Page{Items: []Vehicle{}}, nil)

			rec := serve(t, svc, true, "/vehicles"+tt.query)

			assert.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		})
	}
}

func TestHandlerListRejectsInvalidQueries(t *testing.T) {
	tests := []struct {
		name, query, field string
	}{
		{"make id not a uuid", "?make_id=abc", "make_id"},
		{"model id not a uuid", "?model_id=abc", "model_id"},
		{"aging not a boolean", "?aging=yes", "aging"},
		{"negative min age", "?min_age_days=-1", "min_age_days"},
		{"min age not a number", "?min_age_days=soon", "min_age_days"},
		{"negative max age", "?max_age_days=-5", "max_age_days"},
		{"min greater than max", "?min_age_days=100&max_age_days=50", "min_age_days"},
		{"unknown sort", "?sort=colour", "sort"},
		{"page zero", "?page=0", "page"},
		{"page not a number", "?page=first", "page"},
		{"page size zero", "?page_size=0", "page_size"},
		{"page size over the cap", "?page_size=101", "page_size"},
		{"page size not a number", "?page_size=lots", "page_size"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := serve(t, NewMockReader(t), true, "/vehicles"+tt.query)

			assert.Equal(t, http.StatusBadRequest, rec.Code)
			assert.Contains(t, rec.Body.String(), "validation_error")
			assert.Contains(t, rec.Body.String(), `"`+tt.field+`"`)
		})
	}

	t.Run("every problem is reported at once", func(t *testing.T) {
		rec := serve(t, NewMockReader(t), true, "/vehicles?aging=maybe&sort=x&page=0")
		for _, field := range []string{"aging", "sort", "page"} {
			assert.Contains(t, rec.Body.String(), `"`+field+`"`)
		}
	})
}

func TestHandlerListResponse(t *testing.T) {
	note := "drop price"
	svc := NewMockReader(t)
	svc.EXPECT().List(mock.Anything, "dealer-1", mock.Anything).Return(Page{
		Items: []Vehicle{
			{
				ID: "v1", VIN: "1HGCM82633A004352", Make: catalog.NamedRef{ID: "k1", Name: "Honda"},
				Model: catalog.NamedRef{ID: "m1", Name: "Civic"}, Year: 2024, Price: "24000.00",
				StockedAt: "2026-01-01", AgeDays: 120, IsAging: true,
				CurrentAction: &actions.Action{ID: "a1", VehicleID: "v1", ActionType: "hold", Note: &note, CreatedBy: "alice"},
			},
			{ID: "v2", Make: catalog.NamedRef{ID: "k1"}, Model: catalog.NamedRef{ID: "m1"}},
		},
		Page: 1, PageSize: 25, Total: 2,
	}, nil)

	rec := serve(t, svc, true, "/vehicles")

	assert.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	for _, want := range []string{
		`"total":2`, `"page_size":25`, `"is_aging":true`, `"age_days":120`, `"stocked_at":"2026-01-01"`,
		`"price":"24000.00"`, `"created_by":"alice"`, `"current_action":null`,
	} {
		assert.Contains(t, body, want)
	}
}

func TestHandlerListFailures(t *testing.T) {
	t.Run("service failure is a generic 500", func(t *testing.T) {
		svc := NewMockReader(t)
		svc.EXPECT().List(mock.Anything, mock.Anything, mock.Anything).Return(Page{}, errors.New("secret"))

		rec := serve(t, svc, true, "/vehicles")

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.NotContains(t, rec.Body.String(), "secret")
	})

	t.Run("not signed in", func(t *testing.T) {
		assert.Equal(t, http.StatusUnauthorized, serve(t, NewMockReader(t), false, "/vehicles").Code)
	})
}

func TestHandlerSummary(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := NewMockReader(t)
		svc.EXPECT().Summary(mock.Anything, "dealer-1").Return(Summary{Total: 50, Aging: 13, AgingThresholdDays: 90}, nil)

		rec := serve(t, svc, true, "/vehicles/summary")

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.JSONEq(t, `{"total":50,"aging":13,"aging_threshold_days":90}`, rec.Body.String())
	})

	t.Run("failure", func(t *testing.T) {
		svc := NewMockReader(t)
		svc.EXPECT().Summary(mock.Anything, mock.Anything).Return(Summary{}, errors.New("boom"))
		assert.Equal(t, http.StatusInternalServerError, serve(t, svc, true, "/vehicles/summary").Code)
	})

	t.Run("not signed in", func(t *testing.T) {
		assert.Equal(t, http.StatusUnauthorized, serve(t, NewMockReader(t), false, "/vehicles/summary").Code)
	})
}

func TestHandlerGet(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := NewMockReader(t)
		svc.EXPECT().Get(mock.Anything, "dealer-1", vehicleID).Return(Vehicle{ID: vehicleID, VIN: "V"}, nil)

		rec := serve(t, svc, true, "/vehicles/"+vehicleID)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), `"vin":"V"`)
	})

	t.Run("unknown or foreign vehicle", func(t *testing.T) {
		svc := NewMockReader(t)
		svc.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Return(Vehicle{}, ErrNotFound)

		rec := serve(t, svc, true, "/vehicles/"+vehicleID)

		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Contains(t, rec.Body.String(), `"not_found"`)
	})

	t.Run("unexpected failure", func(t *testing.T) {
		svc := NewMockReader(t)
		svc.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Return(Vehicle{}, errors.New("boom"))
		assert.Equal(t, http.StatusInternalServerError, serve(t, svc, true, "/vehicles/"+vehicleID).Code)
	})

	t.Run("malformed id never reaches the service", func(t *testing.T) {
		rec := serve(t, NewMockReader(t), true, "/vehicles/not-a-uuid")

		require.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "validation_error")
	})

	t.Run("not signed in", func(t *testing.T) {
		assert.Equal(t, http.StatusUnauthorized, serve(t, NewMockReader(t), false, "/vehicles/"+vehicleID).Code)
	})
}
