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

	"github.com/bluesky2106/keyloop-coding-challenge/backend/internal/api"
	"github.com/bluesky2106/keyloop-coding-challenge/backend/internal/auth"
	"github.com/bluesky2106/keyloop-coding-challenge/backend/internal/config"
	"github.com/bluesky2106/keyloop-coding-challenge/backend/internal/platform"
)

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
	authSvc := auth.NewService(auth.NewPGStore(pool), tokens)

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           api.NewRouter(authSvc, tokens),
		ReadHeaderTimeout: 5 * time.Second,
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
