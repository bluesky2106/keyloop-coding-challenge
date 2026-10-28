// Package testdb creates isolated PostgreSQL databases for tests.
package testdb

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bluesky2106/keyloop-coding-challenge/backend/internal/platform"
)

// New returns a pool to a fresh database with all migrations applied. The database is
// dropped when the test ends. It needs DATABASE_URL (any database on a server the user
// may create databases on); start one with `make db-up`.
func New(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool, cleanup, err := Setup(true)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	return pool
}

// NewEmpty is like New but leaves the database without any schema.
func NewEmpty(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool, cleanup, err := Setup(false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	return pool
}

// Setup creates a fresh database, optionally migrated, and returns a pool and a cleanup
// that drops it. It exists for TestMain, which has no *testing.T.
func Setup(migrate bool) (*pgxpool.Pool, func(), error) {
	ctx := context.Background()

	adminURL := os.Getenv("DATABASE_URL")
	if adminURL == "" {
		return nil, nil, fmt.Errorf("DATABASE_URL is not set: run `make db-up` and export the variables from .env")
	}
	admin, err := pgxpool.New(ctx, adminURL)
	if err != nil {
		return nil, nil, fmt.Errorf("connect admin: %w", err)
	}

	suffix := make([]byte, 6)
	if _, err := rand.Read(suffix); err != nil {
		admin.Close()
		return nil, nil, fmt.Errorf("random name: %w", err)
	}
	name := "test_" + hex.EncodeToString(suffix)
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		admin.Close()
		return nil, nil, fmt.Errorf("create database: %w", err)
	}
	drop := func() {
		_, _ = admin.Exec(ctx, "DROP DATABASE "+name+" WITH (FORCE)")
		admin.Close()
	}

	u, err := url.Parse(adminURL)
	if err != nil {
		drop()
		return nil, nil, fmt.Errorf("parse DATABASE_URL: %w", err)
	}
	u.Path = "/" + name
	testURL := u.String()

	if migrate {
		if err := platform.Migrate(testURL); err != nil {
			drop()
			return nil, nil, fmt.Errorf("migrate: %w", err)
		}
	}
	pool, err := platform.Connect(ctx, testURL)
	if err != nil {
		drop()
		return nil, nil, fmt.Errorf("connect test database: %w", err)
	}
	return pool, func() { pool.Close(); drop() }, nil
}
