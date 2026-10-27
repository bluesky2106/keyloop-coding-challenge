// Command seed loads the demo dataset from CSV files. It is safe to run repeatedly.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/bluesky2106/keyloop-coding-challenge/backend/internal/platform"
	"github.com/bluesky2106/keyloop-coding-challenge/backend/internal/seed"
)

func main() {
	if err := run(); err != nil {
		slog.Error("seed failed", "err", err)
		os.Exit(1)
	}
}

func run() error {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	dir := os.Getenv("SEED_DIR")
	if dir == "" {
		dir = "seed"
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	if err := platform.Migrate(databaseURL); err != nil {
		return err
	}
	pool, err := platform.Connect(ctx, databaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := seed.Load(ctx, pool, dir); err != nil {
		return err
	}
	slog.Info("seed complete", "dir", dir)
	return nil
}
