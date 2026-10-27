package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bluesky2106/keyloop-coding-challenge/backend/internal/auth"
	"github.com/bluesky2106/keyloop-coding-challenge/backend/internal/seed"
	"github.com/bluesky2106/keyloop-coding-challenge/backend/internal/testdb"
)

type vehicleJSON struct {
	ID            string `json:"id"`
	VIN           string `json:"vin"`
	AgeDays       int    `json:"age_days"`
	IsAging       bool   `json:"is_aging"`
	CurrentAction *struct {
		ID         string  `json:"id"`
		ActionType string  `json:"action_type"`
		Note       *string `json:"note"`
		CreatedBy  string  `json:"created_by"`
	} `json:"current_action"`
	Make  struct{ ID, Name string } `json:"make"`
	Model struct{ ID, Name string } `json:"model"`
}

type pageJSON struct {
	Items    []vehicleJSON `json:"items"`
	Page     int           `json:"page"`
	PageSize int           `json:"page_size"`
	Total    int           `json:"total"`
}

type summaryJSON struct {
	Total              int `json:"total"`
	Aging              int `json:"aging"`
	AgingThresholdDays int `json:"aging_threshold_days"`
}

func getJSON[T any](t *testing.T, a *app, path, token string) T {
	t.Helper()
	rec := a.do(http.MethodGet, path, "", token)
	require.Equal(t, http.StatusOK, rec.Code, "%s: %s", path, rec.Body)
	var out T
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out), rec.Body.String())
	return out
}

func TestEveryProtectedRouteNeedsAToken(t *testing.T) {
	a := newApp(t)
	id := "3f2b8c1e-9a4d-4e7b-8c5a-1d2e3f4a5b6c"
	for method, paths := range map[string][]string{
		http.MethodGet: {
			"/api/v1/me", "/api/v1/makes", "/api/v1/vehicles", "/api/v1/vehicles/summary",
			"/api/v1/vehicles/" + id, "/api/v1/vehicles/" + id + "/actions",
		},
		http.MethodPost: {"/api/v1/vehicles/" + id + "/actions"},
	} {
		for _, path := range paths {
			rec := a.do(method, path, `{"action_type":"hold"}`, "")
			assert.Equal(t, http.StatusUnauthorized, rec.Code, "%s %s", method, path)
		}
	}
}

func TestMakes(t *testing.T) {
	a := newApp(t)
	token := a.login("northside.manager", "demo-northside-1")

	out := getJSON[struct {
		Items []struct {
			ID, Name string
			Models   []struct{ ID, Name string }
		}
	}](t, a, "/api/v1/makes", token)

	assert.Len(t, out.Items, 7)
	models := 0
	for i, m := range out.Items {
		models += len(m.Models)
		if i > 0 {
			assert.LessOrEqual(t, strings.ToLower(out.Items[i-1].Name), strings.ToLower(m.Name), "sorted by name")
		}
	}
	assert.Equal(t, 21, models)
}

func TestVehicleListAndSummaryAgree(t *testing.T) {
	a := newApp(t)
	token := a.login("northside.manager", "demo-northside-1")

	summary := getJSON[summaryJSON](t, a, "/api/v1/vehicles/summary", token)
	assert.Equal(t, 50, summary.Total, "each demo dealership has 50 vehicles")
	assert.Equal(t, agingThreshold, summary.AgingThresholdDays)
	assert.Positive(t, summary.Aging)

	all := getJSON[pageJSON](t, a, "/api/v1/vehicles?page_size=100", token)
	assert.Equal(t, summary.Total, all.Total)
	assert.Len(t, all.Items, 50)
	aging := 0
	for i, v := range all.Items {
		assert.Equal(t, v.AgeDays > agingThreshold, v.IsAging, "is_aging must follow age_days (%d)", v.AgeDays)
		if v.IsAging {
			aging++
		}
		if i > 0 {
			assert.GreaterOrEqual(t, all.Items[i-1].AgeDays, v.AgeDays, "default order is oldest first")
		}
	}
	assert.Equal(t, summary.Aging, aging)

	onlyAging := getJSON[pageJSON](t, a, "/api/v1/vehicles?aging=true&page_size=100", token)
	assert.Equal(t, summary.Aging, onlyAging.Total)
	for _, v := range onlyAging.Items {
		assert.True(t, v.IsAging)
	}

	notAging := getJSON[pageJSON](t, a, "/api/v1/vehicles?aging=false&page_size=100", token)
	assert.Equal(t, summary.Total-summary.Aging, notAging.Total)
}

func TestTheNinetyDayBoundaryInTheDemoData(t *testing.T) {
	a := newApp(t)
	token := a.login("northside.manager", "demo-northside-1")

	page := getJSON[pageJSON](t, a, "/api/v1/vehicles?min_age_days=90&max_age_days=91&page_size=100", token)

	flags := map[int]bool{}
	for _, v := range page.Items {
		flags[v.AgeDays] = v.IsAging
	}
	assert.Equal(t, map[int]bool{90: false, 91: true}, flags)
}

func TestFilteringByMakeAndModelAndPaging(t *testing.T) {
	a := newApp(t)
	token := a.login("northside.manager", "demo-northside-1")
	makes := getJSON[struct {
		Items []struct {
			ID     string
			Models []struct{ ID string }
		}
	}](t, a, "/api/v1/makes", token)
	makeID, modelID := makes.Items[0].ID, makes.Items[0].Models[0].ID

	byMake := getJSON[pageJSON](t, a, "/api/v1/vehicles?page_size=100&make_id="+makeID, token)
	require.NotEmpty(t, byMake.Items)
	for _, v := range byMake.Items {
		assert.Equal(t, makeID, v.Make.ID)
	}
	byModel := getJSON[pageJSON](t, a, "/api/v1/vehicles?page_size=100&model_id="+modelID, token)
	for _, v := range byModel.Items {
		assert.Equal(t, modelID, v.Model.ID)
	}
	assert.LessOrEqual(t, byModel.Total, byMake.Total)

	first := getJSON[pageJSON](t, a, "/api/v1/vehicles?page=1&page_size=20", token)
	second := getJSON[pageJSON](t, a, "/api/v1/vehicles?page=2&page_size=20", token)
	assert.Equal(t, 1, first.Page)
	assert.Equal(t, 2, second.Page)
	assert.Equal(t, 20, first.PageSize)
	seen := map[string]bool{}
	for _, v := range append(first.Items, second.Items...) {
		assert.False(t, seen[v.ID], "a vehicle appeared on two pages")
		seen[v.ID] = true
	}
}

func TestDealershipsNeverSeeEachOthersVehicles(t *testing.T) {
	a := newApp(t)
	north := a.login("northside.manager", "demo-northside-1")
	summit := a.login("summit.manager", "demo-summit-1")

	northIDs := map[string]bool{}
	for _, v := range getJSON[pageJSON](t, a, "/api/v1/vehicles?page_size=100", north).Items {
		northIDs[v.ID] = true
	}
	summitVehicles := getJSON[pageJSON](t, a, "/api/v1/vehicles?page_size=100", summit).Items
	require.NotEmpty(t, summitVehicles)
	for _, v := range summitVehicles {
		assert.False(t, northIDs[v.ID], "vehicle %s is in both dealerships' lists", v.ID)
	}

	foreign := summitVehicles[0].ID
	for _, tc := range []struct{ method, path, body string }{
		{http.MethodGet, "/api/v1/vehicles/" + foreign, ""},
		{http.MethodGet, "/api/v1/vehicles/" + foreign + "/actions", ""},
		{http.MethodPost, "/api/v1/vehicles/" + foreign + "/actions", `{"action_type":"hold"}`},
	} {
		rec := a.do(tc.method, tc.path, tc.body, north)
		assert.Equal(t, http.StatusNotFound, rec.Code, "%s %s: %s", tc.method, tc.path, rec.Body)
		assert.Equal(t, "not_found", decodeErr(t, rec).Error.Code)
	}
}

func TestVehicleLookups(t *testing.T) {
	a := newApp(t)
	token := a.login("northside.manager", "demo-northside-1")
	first := getJSON[pageJSON](t, a, "/api/v1/vehicles?page_size=1", token).Items[0]

	got := getJSON[vehicleJSON](t, a, "/api/v1/vehicles/"+first.ID, token)
	assert.Equal(t, first.ID, got.ID)
	assert.Equal(t, first.VIN, got.VIN)

	assert.Equal(t, http.StatusNotFound,
		a.do(http.MethodGet, "/api/v1/vehicles/00000000-0000-0000-0000-000000000000", "", token).Code)
	rec := a.do(http.MethodGet, "/api/v1/vehicles/not-a-uuid", "", token)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "validation_error", decodeErr(t, rec).Error.Code)
}

func TestInvalidQueriesAreRejected(t *testing.T) {
	a := newApp(t)
	token := a.login("northside.manager", "demo-northside-1")
	for _, q := range []string{"aging=maybe", "sort=colour", "page=0", "page_size=500", "min_age_days=9&max_age_days=1", "make_id=1",
		"max_age_days=2147483647", "min_age_days=100001", "page=9223372036854775807"} {
		rec := a.do(http.MethodGet, "/api/v1/vehicles?"+q, "", token)
		assert.Equal(t, http.StatusBadRequest, rec.Code, q)
		assert.Equal(t, "validation_error", decodeErr(t, rec).Error.Code, q)
	}
}

// Extreme but valid values must not turn into server errors.
func TestExtremeValidValuesAreHandled(t *testing.T) {
	a := newApp(t)
	token := a.login("northside.manager", "demo-northside-1")

	farPage := getJSON[pageJSON](t, a, "/api/v1/vehicles?page=2147483647&page_size=100", token)
	assert.Empty(t, farPage.Items)
	assert.Equal(t, 50, farPage.Total, "the total survives a page past the end")

	oldest := getJSON[pageJSON](t, a, "/api/v1/vehicles?max_age_days=100000&min_age_days=100000", token)
	assert.Empty(t, oldest.Items)
}

// The write flow gets its own database: it appends actions, and the tests above assume
// the shared one is untouched.
func TestLoggingActionsEndToEnd(t *testing.T) {
	pool := testdb.New(t)
	require.NoError(t, seed.Load(context.Background(), pool, "../../seed"))
	tokens := auth.NewTokens(secret, time.Hour)
	a := &app{t: t, handler: NewRouter(newDeps(pool, tokens))}
	token := a.login("northside.manager", "demo-northside-1")

	agingVehicle := getJSON[pageJSON](t, a, "/api/v1/vehicles?aging=true&page_size=1", token).Items[0]
	youngVehicle := getJSON[pageJSON](t, a, "/api/v1/vehicles?aging=false&page_size=1", token).Items[0]
	actionsPath := func(id string) string { return "/api/v1/vehicles/" + id + "/actions" }
	post := func(id, body string) *httptest.ResponseRecorder {
		return a.do(http.MethodPost, actionsPath(id), body, token)
	}

	t.Run("a new vehicle has no history and no current action", func(t *testing.T) {
		hist := getJSON[struct{ Items []json.RawMessage }](t, a, actionsPath(agingVehicle.ID), token)
		assert.Empty(t, hist.Items)
		assert.Nil(t, agingVehicle.CurrentAction)
	})

	t.Run("logging an action records who did it", func(t *testing.T) {
		rec := post(agingVehicle.ID, `{"action_type":"price_reduction_planned","note":"  cut 5%  "}`)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		var created struct {
			ID         string  `json:"id"`
			ActionType string  `json:"action_type"`
			CreatedBy  string  `json:"created_by"`
			Note       *string `json:"note"`
			VehicleID  string  `json:"vehicle_id"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
		assert.Equal(t, "price_reduction_planned", created.ActionType)
		assert.Equal(t, "northside.manager", created.CreatedBy)
		assert.Equal(t, agingVehicle.ID, created.VehicleID)
		require.NotNil(t, created.Note)
		assert.Equal(t, "cut 5%", *created.Note)
	})

	t.Run("the action is now the vehicle's current action everywhere", func(t *testing.T) {
		one := getJSON[vehicleJSON](t, a, "/api/v1/vehicles/"+agingVehicle.ID, token)
		require.NotNil(t, one.CurrentAction)
		assert.Equal(t, "price_reduction_planned", one.CurrentAction.ActionType)

		page := getJSON[pageJSON](t, a, "/api/v1/vehicles?aging=true&page_size=100", token)
		for _, v := range page.Items {
			if v.ID == agingVehicle.ID {
				require.NotNil(t, v.CurrentAction)
				assert.Equal(t, one.CurrentAction.ID, v.CurrentAction.ID)
			}
		}
	})

	t.Run("history is append-only and newest first", func(t *testing.T) {
		require.Equal(t, http.StatusCreated, post(agingVehicle.ID, `{"action_type":"hold"}`).Code)

		hist := getJSON[struct {
			Items []struct {
				ActionType string `json:"action_type"`
				Note       *string
			}
		}](t, a, actionsPath(agingVehicle.ID), token)

		require.Len(t, hist.Items, 2)
		assert.Equal(t, "hold", hist.Items[0].ActionType)
		assert.Nil(t, hist.Items[0].Note)
		assert.Equal(t, "price_reduction_planned", hist.Items[1].ActionType)
		one := getJSON[vehicleJSON](t, a, "/api/v1/vehicles/"+agingVehicle.ID, token)
		assert.Equal(t, "hold", one.CurrentAction.ActionType, "the current action is the newest")
	})

	t.Run("a vehicle that is not aging is refused", func(t *testing.T) {
		rec := post(youngVehicle.ID, `{"action_type":"hold"}`)

		assert.Equal(t, http.StatusConflict, rec.Code)
		assert.Equal(t, "vehicle_not_aging", decodeErr(t, rec).Error.Code)
		hist := getJSON[struct{ Items []json.RawMessage }](t, a, actionsPath(youngVehicle.ID), token)
		assert.Empty(t, hist.Items, "a refused action must not be stored")
	})

	t.Run("invalid input is refused", func(t *testing.T) {
		for _, body := range []string{`{"action_type":"bogus"}`, `{}`, `nope`, `{"action_type":"hold","note":"` + strings.Repeat("x", 501) + `"}`,
			`{"action_type":"hold","note":"a\u0000b"}`} {
			rec := post(agingVehicle.ID, body)
			assert.Equal(t, http.StatusBadRequest, rec.Code, body)
			assert.Equal(t, "validation_error", decodeErr(t, rec).Error.Code, body)
		}
		assert.Equal(t, http.StatusBadRequest, post("not-a-uuid", `{"action_type":"hold"}`).Code)
	})
}
