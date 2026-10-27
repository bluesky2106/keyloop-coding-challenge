package inventory

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestServiceList(t *testing.T) {
	f := Filter{Sort: SortAgeDesc, Page: 2, PageSize: 10}
	store := NewMockStore(t)
	store.EXPECT().List(mock.Anything, "d1", f, 90).Return([]Vehicle{{ID: "v1"}}, 31, nil).Once()
	store.EXPECT().List(mock.Anything, "d2", f, 90).Return(nil, 0, errors.New("boom")).Once()
	svc := NewService(store, 90)

	got, err := svc.List(t.Context(), "d1", f)
	require.NoError(t, err)
	assert.Equal(t, Page{Items: []Vehicle{{ID: "v1"}}, Page: 2, PageSize: 10, Total: 31}, got)

	_, err = svc.List(t.Context(), "d2", f)
	require.Error(t, err)
}

func TestServiceGet(t *testing.T) {
	store := NewMockStore(t)
	store.EXPECT().Get(mock.Anything, "d1", "v1", 45).Return(Vehicle{ID: "v1"}, nil).Once()
	store.EXPECT().Get(mock.Anything, "d1", "v2", 45).Return(Vehicle{}, ErrNotFound).Once()
	svc := NewService(store, 45)

	got, err := svc.Get(t.Context(), "d1", "v1")
	require.NoError(t, err)
	assert.Equal(t, "v1", got.ID)

	_, err = svc.Get(t.Context(), "d1", "v2")
	require.ErrorIs(t, err, ErrNotFound)
}

func TestServiceSummaryReportsTheThreshold(t *testing.T) {
	store := NewMockStore(t)
	store.EXPECT().Counts(mock.Anything, "d1", 60).Return(50, 12, nil).Once()
	store.EXPECT().Counts(mock.Anything, "d2", 60).Return(0, 0, errors.New("boom")).Once()
	svc := NewService(store, 60)

	got, err := svc.Summary(t.Context(), "d1")
	require.NoError(t, err)
	assert.Equal(t, Summary{Total: 50, Aging: 12, AgingThresholdDays: 60}, got)

	_, err = svc.Summary(t.Context(), "d2")
	require.Error(t, err)
}
