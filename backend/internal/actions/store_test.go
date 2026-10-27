package actions

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bluesky2106/keyloop-coding-challenge/backend/internal/testdb"
)

const threshold = 90

func TestPGStoreCreate(t *testing.T) {
	pool := testdb.New(t)
	f := testdb.NewFixtures(t, pool)
	store := NewPGStore(pool)
	aging := f.AddVehicle(testdb.VehicleSpec{AgeDays: 91})
	atBoundary := f.AddVehicle(testdb.VehicleSpec{AgeDays: 90})
	young := f.AddVehicle(testdb.VehicleSpec{AgeDays: 3})
	foreign := f.AddVehicle(testdb.VehicleSpec{Dealership: f.South, AgeDays: 200})
	note := "10% off"

	create := func(vehicle string) (Action, error) {
		return store.Create(t.Context(), NewAction{
			DealershipID: f.North, UserID: f.NorthUser, VehicleID: vehicle, ActionType: "hold", Note: &note,
		}, threshold)
	}

	t.Run("an aging vehicle accepts an action", func(t *testing.T) {
		a, err := create(aging)
		require.NoError(t, err)
		assert.NotEmpty(t, a.ID)
		assert.Equal(t, aging, a.VehicleID)
		assert.Equal(t, "hold", a.ActionType)
		assert.Equal(t, "north.user", a.CreatedBy)
		require.NotNil(t, a.Note)
		assert.Equal(t, note, *a.Note)
		assert.False(t, a.CreatedAt.IsZero())
	})

	t.Run("a null note is stored as null", func(t *testing.T) {
		a, err := store.Create(t.Context(), NewAction{
			DealershipID: f.North, UserID: f.NorthUser, VehicleID: aging, ActionType: "hold",
		}, threshold)
		require.NoError(t, err)
		assert.Nil(t, a.Note)
	})

	t.Run("exactly 90 days is not aging", func(t *testing.T) {
		_, err := create(atBoundary)
		require.ErrorIs(t, err, ErrNotAging)
	})

	t.Run("a young vehicle is rejected", func(t *testing.T) {
		_, err := create(young)
		require.ErrorIs(t, err, ErrNotAging)
	})

	t.Run("the threshold decides, not a constant", func(t *testing.T) {
		_, err := store.Create(t.Context(), NewAction{
			DealershipID: f.North, UserID: f.NorthUser, VehicleID: young, ActionType: "hold",
		}, 2)
		require.NoError(t, err)
	})

	t.Run("another dealership's vehicle is not found", func(t *testing.T) {
		_, err := create(foreign)
		require.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("an unknown vehicle is not found", func(t *testing.T) {
		_, err := create("00000000-0000-0000-0000-000000000000")
		require.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("a rejected action writes nothing", func(t *testing.T) {
		var n int
		require.NoError(t, pool.QueryRow(t.Context(),
			`SELECT count(*) FROM vehicle_actions WHERE vehicle_id IN ($1, $2, $3)`, atBoundary, foreign, "00000000-0000-0000-0000-000000000000").Scan(&n))
		assert.Zero(t, n)
	})

	t.Run("an invalid action type is rejected by the database", func(t *testing.T) {
		_, err := store.Create(t.Context(), NewAction{
			DealershipID: f.North, UserID: f.NorthUser, VehicleID: aging, ActionType: "bogus",
		}, threshold)
		require.Error(t, err)
		assert.NotErrorIs(t, err, ErrNotFound)
		assert.NotErrorIs(t, err, ErrNotAging)
	})

	t.Run("the database accepts every action type the handler accepts", func(t *testing.T) {
		for typ := range validTypes {
			_, err := store.Create(t.Context(), NewAction{
				DealershipID: f.North, UserID: f.NorthUser, VehicleID: aging, ActionType: typ,
			}, threshold)
			require.NoError(t, err, typ)
		}
	})

	t.Run("database failure", func(t *testing.T) {
		broken := testdb.New(t)
		bf := testdb.NewFixtures(t, broken)
		v := bf.AddVehicle(testdb.VehicleSpec{AgeDays: 100})
		broken.Close()
		_, err := NewPGStore(broken).Create(t.Context(), NewAction{
			DealershipID: bf.North, UserID: bf.NorthUser, VehicleID: v, ActionType: "hold",
		}, threshold)
		require.Error(t, err)
		assert.NotErrorIs(t, err, ErrNotFound)
	})
}

func TestPGStoreList(t *testing.T) {
	pool := testdb.New(t)
	f := testdb.NewFixtures(t, pool)
	store := NewPGStore(pool)
	vehicle := f.AddVehicle(testdb.VehicleSpec{AgeDays: 120})
	foreign := f.AddVehicle(testdb.VehicleSpec{Dealership: f.South, AgeDays: 120})
	f.AddAction(vehicle, f.North, f.NorthUser, "hold", "first")
	f.AddAction(vehicle, f.North, f.NorthUser, "price_reduction_planned", "")
	f.AddAction(vehicle, f.North, f.NorthUser, "transfer_planned", "third")
	f.AddAction(foreign, f.South, f.SouthUser, "hold", "not yours")

	t.Run("newest first", func(t *testing.T) {
		got, err := store.List(t.Context(), f.North, vehicle)
		require.NoError(t, err)
		require.Len(t, got, 3)
		assert.Equal(t, "transfer_planned", got[0].ActionType)
		assert.Equal(t, "price_reduction_planned", got[1].ActionType)
		assert.Equal(t, "hold", got[2].ActionType)
		assert.Equal(t, "north.user", got[0].CreatedBy)
		assert.Nil(t, got[1].Note, "an empty note is stored as null")
	})

	t.Run("a vehicle with no history is an empty list", func(t *testing.T) {
		got, err := store.List(t.Context(), f.North, f.AddVehicle(testdb.VehicleSpec{AgeDays: 5}))
		require.NoError(t, err)
		assert.NotNil(t, got)
		assert.Empty(t, got)
	})

	t.Run("another dealership's vehicle is not found", func(t *testing.T) {
		_, err := store.List(t.Context(), f.North, foreign)
		require.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("an unknown vehicle is not found", func(t *testing.T) {
		_, err := store.List(t.Context(), f.North, strings.Repeat("0", 8)+"-0000-0000-0000-000000000000")
		require.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("database failure", func(t *testing.T) {
		broken := testdb.New(t)
		broken.Close()
		_, err := NewPGStore(broken).List(t.Context(), f.North, vehicle)
		require.Error(t, err)
		assert.NotErrorIs(t, err, ErrNotFound)
	})
}
