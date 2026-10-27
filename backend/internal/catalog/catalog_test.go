package catalog

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/bluesky2106/keyloop-coding-challenge/backend/internal/seed"
	"github.com/bluesky2106/keyloop-coding-challenge/backend/internal/testdb"
)

func TestHandlerList(t *testing.T) {
	get := func(t *testing.T, store Store) *httptest.ResponseRecorder {
		t.Helper()
		rec := httptest.NewRecorder()
		NewHandler(store).List(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/makes", nil))
		return rec
	}

	t.Run("wraps the makes in items", func(t *testing.T) {
		store := NewMockStore(t)
		store.EXPECT().ListMakes(mock.Anything).Return([]Make{
			{ID: "k1", Name: "Honda", Models: []NamedRef{{ID: "m1", Name: "Civic"}}},
		}, nil)

		rec := get(t, store)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.JSONEq(t, `{"items":[{"id":"k1","name":"Honda","models":[{"id":"m1","name":"Civic"}]}]}`, rec.Body.String())
	})

	t.Run("an empty catalog is an empty list, not null", func(t *testing.T) {
		store := NewMockStore(t)
		store.EXPECT().ListMakes(mock.Anything).Return([]Make{}, nil)

		assert.JSONEq(t, `{"items":[]}`, get(t, store).Body.String())
	})

	t.Run("store failure is a generic 500", func(t *testing.T) {
		store := NewMockStore(t)
		store.EXPECT().ListMakes(mock.Anything).Return(nil, errors.New("secret connection string"))

		rec := get(t, store)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.NotContains(t, rec.Body.String(), "secret")
	})
}

func TestPGStoreListMakes(t *testing.T) {
	t.Run("empty database", func(t *testing.T) {
		makes, err := NewPGStore(testdb.New(t)).ListMakes(t.Context())
		require.NoError(t, err)
		assert.NotNil(t, makes)
		assert.Empty(t, makes)
	})

	t.Run("sorted by name, models grouped under their make", func(t *testing.T) {
		pool := testdb.New(t)
		f := testdb.NewFixtures(t, pool) // Toyota / Corolla
		f.AddModel("Toyota", "camry")
		f.AddModel("Audi", "A4")
		f.AddModel("honda", "Civic")

		makes, err := NewPGStore(pool).ListMakes(t.Context())

		require.NoError(t, err)
		var got []string
		for _, m := range makes {
			got = append(got, m.Name)
			for _, model := range m.Models {
				got = append(got, "  "+model.Name)
			}
		}
		assert.Equal(t, []string{"Audi", "  A4", "honda", "  Civic", "Toyota", "  camry", "  Corolla"}, got)
	})

	t.Run("the demo dataset", func(t *testing.T) {
		pool := testdb.New(t)
		require.NoError(t, seed.Load(t.Context(), pool, "../../seed"))

		makes, err := NewPGStore(pool).ListMakes(t.Context())

		require.NoError(t, err)
		assert.Len(t, makes, 7)
		models := 0
		for _, m := range makes {
			assert.NotEmpty(t, m.ID)
			models += len(m.Models)
		}
		assert.Equal(t, 21, models)
	})

	t.Run("database failure", func(t *testing.T) {
		pool := testdb.New(t)
		pool.Close()
		_, err := NewPGStore(pool).ListMakes(t.Context())
		require.Error(t, err)
	})
}
