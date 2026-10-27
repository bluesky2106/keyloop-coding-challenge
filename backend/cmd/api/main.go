// Command api starts the inventory dashboard HTTP server.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bluesky2106/keyloop-coding-challenge/backend/internal/actions"
	"github.com/bluesky2106/keyloop-coding-challenge/backend/internal/api"
	"github.com/bluesky2106/keyloop-coding-challenge/backend/internal/auth"
	"github.com/bluesky2106/keyloop-coding-challenge/backend/internal/catalog"
	"github.com/bluesky2106/keyloop-coding-challenge/backend/internal/config"
	"github.com/bluesky2106/keyloop-coding-challenge/backend/internal/inventory"
	"github.com/bluesky2106/keyloop-coding-challenge/backend/internal/platform"
)

// agingThresholdDays is the single source of the aging rule: a vehicle is aging when it
// has been in stock for more than this many days. Making it per-dealership later means
// changing where this value comes from (see docs/system-design.md, section 7).
const agingThresholdDays = 90

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, os.Getenv, logger); err != nil {
		logger.Error("server failed", "err", err)
		os.Exit(1)
	}
}

// run starts the server and blocks until ctx is cancelled (graceful shutdown) or the
// server fails.
func run(ctx context.Context, getenv func(string) string, logger *slog.Logger) error {
	cfg, err := config.Load(getenv)
	if err != nil {
		return err
	}

	if err := platform.Migrate(cfg.DatabaseURL); err != nil {
		return err
	}
	pool, err := platform.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	tokens := auth.NewTokens(cfg.JWTSecret, cfg.JWTTTL)
	handler := api.NewRouter(api.Deps{
		Tokens:    tokens,
		Auth:      auth.NewService(auth.NewPGStore(pool), tokens),
		Catalog:   catalog.NewPGStore(pool),
		Inventory: inventory.NewService(inventory.NewPGStore(pool), agingThresholdDays),
		Actions:   actions.NewService(actions.NewPGStore(pool), agingThresholdDays),
	})

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serveErr := make(chan error, 1)
	go func() {
		logger.Info("listening", "addr", cfg.Addr)
		serveErr <- srv.ListenAndServe()
	}()

	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	if err := <-serveErr; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
