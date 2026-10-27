// Package testdb creates an isolated, fully migrated PostgreSQL database per test.
package testdb

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/url"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bluesky2106/keyloop-coding-challenge/backend/internal/platform"
)

// New returns a pool to a fresh database with all migrations applied. The database is
// dropped when the test ends. It needs DATABASE_URL (any database on a server the user
// may create databases on); start one with `make up`.
func New(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()

	adminURL := os.Getenv("DATABASE_URL")
	if adminURL == "" {
		t.Fatal("DATABASE_URL is not set: run `make up` and export the variables from .env")
	}
	admin, err := pgxpool.New(ctx, adminURL)
	if err != nil {
		t.Fatalf("connect admin: %v", err)
	}

	suffix := make([]byte, 6)
	if _, err := rand.Read(suffix); err != nil {
		t.Fatalf("random name: %v", err)
	}
	name := "test_" + hex.EncodeToString(suffix)
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		admin.Close()
		t.Fatalf("create database: %v", err)
	}

	u, err := url.Parse(adminURL)
	if err != nil {
		t.Fatalf("parse DATABASE_URL: %v", err)
	}
	u.Path = "/" + name
	testURL := u.String()

	if err := platform.Migrate(testURL); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := platform.Connect(ctx, testURL)
	if err != nil {
		t.Fatalf("connect test database: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
		if _, err := admin.Exec(ctx, "DROP DATABASE "+name+" WITH (FORCE)"); err != nil {
			t.Errorf("drop database: %v", err)
		}
		admin.Close()
	})
	return pool
}
