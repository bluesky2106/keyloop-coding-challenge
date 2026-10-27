package actions

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestServicePassesTheThresholdToTheStore(t *testing.T) {
	n := NewAction{DealershipID: "d1", UserID: "u1", VehicleID: "v1", ActionType: "hold"}
	want := Action{ID: "a1"}
	store := NewMockStore(t)
	store.EXPECT().Create(mock.Anything, n, 90).Return(want, nil)

	got, err := NewService(store, 90).Create(t.Context(), n)

	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestServiceCreateReturnsStoreErrors(t *testing.T) {
	store := NewMockStore(t)
	store.EXPECT().Create(mock.Anything, mock.Anything, 30).Return(Action{}, ErrNotAging)

	_, err := NewService(store, 30).Create(t.Context(), NewAction{})

	require.ErrorIs(t, err, ErrNotAging)
}

func TestServiceList(t *testing.T) {
	store := NewMockStore(t)
	store.EXPECT().List(mock.Anything, "d1", "v1").Return([]Action{{ID: "a1"}}, nil).Once()
	store.EXPECT().List(mock.Anything, "d1", "v2").Return(nil, errors.New("boom")).Once()
	svc := NewService(store, 90)

	got, err := svc.List(t.Context(), "d1", "v1")
	require.NoError(t, err)
	assert.Equal(t, []Action{{ID: "a1"}}, got)

	_, err = svc.List(t.Context(), "d1", "v2")
	require.Error(t, err)
}
