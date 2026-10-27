package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func newTestService(t *testing.T, store Store) *Service {
	t.Helper()
	return NewService(store, NewTokens(testSecret, time.Hour))
}

func hashOf(t *testing.T, password string) string {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	require.NoError(t, err)
	return string(h)
}

func TestServiceLogin(t *testing.T) {
	errDB := errors.New("connection reset")
	user := User{ID: "u1", DealershipID: "d1", PasswordHash: hashOf(t, "secret")}

	tests := []struct {
		name     string
		store    func(*MockStore)
		password string
		wantErr  error
	}{
		{
			name: "valid credentials",
			store: func(s *MockStore) {
				s.EXPECT().FindUserByUsername(mock.Anything, "alice").Return(user, nil)
			},
			password: "secret",
		},
		{
			name: "wrong password",
			store: func(s *MockStore) {
				s.EXPECT().FindUserByUsername(mock.Anything, "alice").Return(user, nil)
			},
			password: "wrong",
			wantErr:  ErrInvalidCredentials,
		},
		{
			name: "unknown user gets the same error as a wrong password",
			store: func(s *MockStore) {
				s.EXPECT().FindUserByUsername(mock.Anything, "alice").Return(User{}, ErrUserNotFound)
			},
			password: "secret",
			wantErr:  ErrInvalidCredentials,
		},
		{
			name: "store failure is not reported as bad credentials",
			store: func(s *MockStore) {
				s.EXPECT().FindUserByUsername(mock.Anything, "alice").Return(User{}, errDB)
			},
			password: "secret",
			wantErr:  errDB,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewMockStore(t)
			tt.store(store)

			res, err := newTestService(t, store).Login(t.Context(), "alice", tt.password)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Empty(t, res.AccessToken)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, 3600, res.ExpiresIn)
			claims, err := NewTokens(testSecret, time.Hour).Parse(res.AccessToken)
			require.NoError(t, err)
			assert.Equal(t, Claims{UserID: "u1", DealershipID: "d1"}, claims)
		})
	}
}

func TestServiceMe(t *testing.T) {
	claims := Claims{UserID: "u1", DealershipID: "d1"}
	want := Me{UserID: "u1", Username: "alice", DealershipID: "d1", DealershipName: "Acme"}

	store := NewMockStore(t)
	store.EXPECT().FindProfile(mock.Anything, claims).Return(want, nil).Once()
	got, err := newTestService(t, store).Me(t.Context(), claims)
	require.NoError(t, err)
	assert.Equal(t, want, got)

	store.EXPECT().FindProfile(mock.Anything, claims).Return(Me{}, ErrUserNotFound).Once()
	_, err = newTestService(t, store).Me(t.Context(), claims)
	require.ErrorIs(t, err, ErrUserNotFound)
}
