package inventory

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bluesky2106/keyloop-coding-challenge/backend/internal/testdb"
)

const threshold = 90

// stock is North's inventory: six vehicles of known age, make, model and price, plus one
// South vehicle that must never leak into North's results.
type stock struct {
	*testdb.Fixtures
	store                                    *PGStore
	honda, civic                             string
	age0, age30, age45, age90, age91, age200 string
	southVehicle                             string
}

func newStock(t *testing.T) *stock {
	t.Helper()
	pool := testdb.New(t)
	s := &stock{Fixtures: testdb.NewFixtures(t, pool), store: NewPGStore(pool)}
	s.honda, s.civic = s.AddModel("Honda", "Civic")

	s.age0 = s.AddVehicle(testdb.VehicleSpec{AgeDays: 0, Price: "20000.00"})
	s.age30 = s.AddVehicle(testdb.VehicleSpec{AgeDays: 30, Price: "22000.00"})
	s.age90 = s.AddVehicle(testdb.VehicleSpec{AgeDays: 90, Price: "24000.00"})
	s.age91 = s.AddVehicle(testdb.VehicleSpec{AgeDays: 91, Price: "26000.00"})
	s.age45 = s.AddVehicle(testdb.VehicleSpec{AgeDays: 45, Price: "28000.00", Model: s.civic})
	s.age200 = s.AddVehicle(testdb.VehicleSpec{AgeDays: 200, Price: "30000.00", Model: s.civic})
	s.southVehicle = s.AddVehicle(testdb.VehicleSpec{Dealership: s.South, AgeDays: 300})
	return s
}

func (s *stock) list(t *testing.T, f Filter) ([]Vehicle, int) {
	t.Helper()
	if f.Page == 0 {
		f.Page = 1
	}
	if f.PageSize == 0 {
		f.PageSize = 100
	}
	items, total, err := s.store.List(t.Context(), s.North, f, threshold)
	require.NoError(t, err)
	return items, total
}

func ids(vs []Vehicle) []string {
	out := make([]string, len(vs))
	for i, v := range vs {
		out[i] = v.ID
	}
	return out
}

func TestListDefaultsToOldestFirstAndScopesToTheDealership(t *testing.T) {
	s := newStock(t)

	items, total := s.list(t, Filter{})

	assert.Equal(t, 6, total)
	assert.Equal(t, []string{s.age200, s.age91, s.age90, s.age45, s.age30, s.age0}, ids(items))
	assert.NotContains(t, ids(items), s.southVehicle, "another dealership's stock must never appear")
}

func TestListReportsAgeAndTheAgingBoundary(t *testing.T) {
	s := newStock(t)
	items, _ := s.list(t, Filter{})
	byID := map[string]Vehicle{}
	for _, v := range items {
		byID[v.ID] = v
	}

	assert.Equal(t, 90, byID[s.age90].AgeDays)
	assert.False(t, byID[s.age90].IsAging, "exactly 90 days is not aging")
	assert.Equal(t, 91, byID[s.age91].AgeDays)
	assert.True(t, byID[s.age91].IsAging, "91 days is aging")
	assert.False(t, byID[s.age0].IsAging)
	assert.Equal(t, 0, byID[s.age0].AgeDays)
	assert.True(t, byID[s.age200].IsAging)
}

func TestListReturnsTheFullVehicleShape(t *testing.T) {
	s := newStock(t)
	items, _ := s.list(t, Filter{ModelID: s.civic, MaxAgeDays: ptr(45)})
	require.Len(t, items, 1)

	v := items[0]
	assert.Equal(t, s.age45, v.ID)
	assert.Regexp(t, `^1HGCM\d{12}$`, v.VIN)
	assert.Equal(t, "Honda", v.Make.Name)
	assert.Equal(t, s.honda, v.Make.ID)
	assert.Equal(t, "Civic", v.Model.Name)
	assert.Equal(t, s.civic, v.Model.ID)
	assert.Equal(t, 2024, v.Year)
	assert.Equal(t, "28000.00", v.Price)
	assert.Regexp(t, `^\d{4}-\d{2}-\d{2}$`, v.StockedAt)
	assert.Nil(t, v.CurrentAction)
}

func TestListFilters(t *testing.T) {
	s := newStock(t)
	tests := []struct {
		name   string
		filter Filter
		want   func() []string
	}{
		{"aging only", Filter{Aging: ptr(true)}, func() []string { return []string{s.age200, s.age91} }},
		{"not aging only", Filter{Aging: ptr(false)}, func() []string { return []string{s.age90, s.age45, s.age30, s.age0} }},
		{"minimum age is inclusive", Filter{MinAgeDays: ptr(90)}, func() []string { return []string{s.age200, s.age91, s.age90} }},
		{"maximum age is inclusive", Filter{MaxAgeDays: ptr(30)}, func() []string { return []string{s.age30, s.age0} }},
		{"age range", Filter{MinAgeDays: ptr(30), MaxAgeDays: ptr(90)}, func() []string { return []string{s.age90, s.age45, s.age30} }},
		{"a single age", Filter{MinAgeDays: ptr(45), MaxAgeDays: ptr(45)}, func() []string { return []string{s.age45} }},
		{"by make", Filter{MakeID: s.honda}, func() []string { return []string{s.age200, s.age45} }},
		{"by model", Filter{ModelID: s.Corolla}, func() []string { return []string{s.age91, s.age90, s.age30, s.age0} }},
		{"make and model together", Filter{MakeID: s.honda, ModelID: s.civic}, func() []string { return []string{s.age200, s.age45} }},
		{"a model that is not the make's matches nothing", Filter{MakeID: s.honda, ModelID: s.Corolla}, func() []string { return []string{} }},
		{"aging and a conflicting age bound", Filter{Aging: ptr(true), MaxAgeDays: ptr(50)}, func() []string { return []string{} }},
		{"every filter combined", Filter{MakeID: s.Toyota, Aging: ptr(false), MinAgeDays: ptr(1), MaxAgeDays: ptr(90)},
			func() []string { return []string{s.age90, s.age30} }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items, total := s.list(t, tt.filter)

			assert.Equal(t, tt.want(), ids(items))
			assert.Equal(t, len(tt.want()), total, "total counts the same filtered set")
		})
	}
}

func TestListSorting(t *testing.T) {
	s := newStock(t)
	tests := []struct {
		sort string
		want []string
	}{
		{SortAgeDesc, []string{s.age200, s.age91, s.age90, s.age45, s.age30, s.age0}},
		{SortAgeAsc, []string{s.age0, s.age30, s.age45, s.age90, s.age91, s.age200}},
		{SortPriceDesc, []string{s.age200, s.age45, s.age91, s.age90, s.age30, s.age0}},
		{SortPriceAsc, []string{s.age0, s.age30, s.age90, s.age91, s.age45, s.age200}},
		{"", []string{s.age200, s.age91, s.age90, s.age45, s.age30, s.age0}}, // unknown falls back to age_desc
		{"; DROP TABLE vehicles", []string{s.age200, s.age91, s.age90, s.age45, s.age30, s.age0}},
	}
	for _, tt := range tests {
		t.Run("sort="+tt.sort, func(t *testing.T) {
			items, _ := s.list(t, Filter{Sort: tt.sort})
			assert.Equal(t, tt.want, ids(items))
		})
	}
}

func TestListPaginationNeverRepeatsOrSkipsRows(t *testing.T) {
	pool := testdb.New(t)
	f := testdb.NewFixtures(t, pool)
	store := NewPGStore(pool)
	want := map[string]bool{}
	for range 7 { // identical age and price: only the id tiebreaker keeps pages stable
		want[f.AddVehicle(testdb.VehicleSpec{AgeDays: 100, Price: "15000.00"})] = true
	}

	for _, sort := range []string{SortAgeDesc, SortAgeAsc, SortPriceDesc, SortPriceAsc} {
		seen := map[string]bool{}
		var order []string
		for page := 1; page <= 4; page++ {
			items, total, err := store.List(t.Context(), f.North, Filter{Sort: sort, Page: page, PageSize: 2}, threshold)
			require.NoError(t, err)
			assert.Equal(t, 7, total)
			for _, v := range items {
				assert.False(t, seen[v.ID], "%s: %s appeared twice", sort, v.ID)
				seen[v.ID] = true
				order = append(order, v.ID)
			}
		}
		assert.Equal(t, want, seen, sort)
		assert.IsIncreasing(t, order, "%s: ties are broken by id ascending", sort)
	}
}

func TestListPageBeyondTheEndIsEmptyButKeepsTheTotal(t *testing.T) {
	s := newStock(t)

	items, total, err := s.store.List(t.Context(), s.North, Filter{Page: 99, PageSize: 10}, threshold)

	require.NoError(t, err)
	assert.NotNil(t, items)
	assert.Empty(t, items)
	assert.Equal(t, 6, total)
}

func TestListShowsTheLatestActionOnly(t *testing.T) {
	s := newStock(t)
	s.AddAction(s.age91, s.North, s.NorthUser, "hold", "first")
	s.AddAction(s.age91, s.North, s.NorthUser, "price_reduction_planned", "latest")

	items, _ := s.list(t, Filter{})

	byID := map[string]Vehicle{}
	for _, v := range items {
		byID[v.ID] = v
	}
	require.NotNil(t, byID[s.age91].CurrentAction)
	assert.Equal(t, "price_reduction_planned", byID[s.age91].CurrentAction.ActionType)
	assert.Equal(t, "latest", *byID[s.age91].CurrentAction.Note)
	assert.Equal(t, "north.user", byID[s.age91].CurrentAction.CreatedBy)
	assert.Equal(t, s.age91, byID[s.age91].CurrentAction.VehicleID)
	assert.Nil(t, byID[s.age200].CurrentAction, "a vehicle with no history has no current action")
	assert.Len(t, items, 6, "history must not multiply rows")
}

func TestGet(t *testing.T) {
	s := newStock(t)
	s.AddAction(s.age91, s.North, s.NorthUser, "hold", "")

	t.Run("an aging vehicle with its action", func(t *testing.T) {
		v, err := s.store.Get(t.Context(), s.North, s.age91, threshold)
		require.NoError(t, err)
		assert.Equal(t, s.age91, v.ID)
		assert.True(t, v.IsAging)
		assert.Equal(t, 91, v.AgeDays)
		require.NotNil(t, v.CurrentAction)
		assert.Equal(t, "hold", v.CurrentAction.ActionType)
		assert.Nil(t, v.CurrentAction.Note)
	})

	t.Run("a young vehicle", func(t *testing.T) {
		v, err := s.store.Get(t.Context(), s.North, s.age30, threshold)
		require.NoError(t, err)
		assert.False(t, v.IsAging)
		assert.Nil(t, v.CurrentAction)
	})

	t.Run("the threshold is a parameter", func(t *testing.T) {
		v, err := s.store.Get(t.Context(), s.North, s.age30, 20)
		require.NoError(t, err)
		assert.True(t, v.IsAging)
	})

	t.Run("another dealership's vehicle is not found", func(t *testing.T) {
		_, err := s.store.Get(t.Context(), s.North, s.southVehicle, threshold)
		require.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("an unknown id is not found", func(t *testing.T) {
		_, err := s.store.Get(t.Context(), s.North, "00000000-0000-0000-0000-000000000000", threshold)
		require.ErrorIs(t, err, ErrNotFound)
	})
}

func TestCounts(t *testing.T) {
	s := newStock(t)

	total, aging, err := s.store.Counts(t.Context(), s.North, threshold)
	require.NoError(t, err)
	assert.Equal(t, 6, total)
	assert.Equal(t, 2, aging)

	total, aging, err = s.store.Counts(t.Context(), s.South, threshold)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, 1, aging)

	total, aging, err = s.store.Counts(t.Context(), s.North, 29)
	require.NoError(t, err)
	assert.Equal(t, 6, total)
	assert.Equal(t, 5, aging, "a lower threshold flags more vehicles")
}

func TestDatabaseFailuresAreReportedNotHidden(t *testing.T) {
	pool := testdb.New(t)
	f := testdb.NewFixtures(t, pool)
	id := f.AddVehicle(testdb.VehicleSpec{})
	pool.Close()
	store := NewPGStore(pool)

	_, _, err := store.List(t.Context(), f.North, Filter{Page: 1, PageSize: 10}, threshold)
	require.Error(t, err)

	_, err = store.Get(t.Context(), f.North, id, threshold)
	require.Error(t, err)
	assert.NotErrorIs(t, err, ErrNotFound, "a broken database is not a missing vehicle")

	_, _, err = store.Counts(t.Context(), f.North, threshold)
	require.Error(t, err)
}
