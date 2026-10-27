package main

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bluesky2106/keyloop-coding-challenge/backend/internal/testdb"
)

var quiet = slog.New(slog.NewTextHandler(io.Discard, nil))

func envOf(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

func validEnv(databaseURL, addr string) map[string]string {
	return map[string]string{
		"DATABASE_URL": databaseURL,
		"JWT_SECRET":   strings.Repeat("s", 32),
		"ADDR":         addr,
	}
}

func TestRunRejectsInvalidConfig(t *testing.T) {
	err := run(t.Context(), envOf(nil), quiet)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DATABASE_URL")
}

func TestRunFailsWhenTheDatabaseIsUnreachable(t *testing.T) {
	env := validEnv("postgres://u@127.0.0.1:1/db?sslmode=disable&connect_timeout=1", "127.0.0.1:0")
	err := run(t.Context(), envOf(env), quiet)
	require.Error(t, err)
}

func TestRunFailsWhenThePortIsTaken(t *testing.T) {
	pool := testdb.New(t)
	taken, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = taken.Close() })

	env := validEnv(pool.Config().ConnString(), taken.Addr().String())
	err = run(t.Context(), envOf(env), quiet)

	require.Error(t, err, "a server that cannot listen must not look like a clean exit")
}

func TestRunServesThenShutsDownGracefully(t *testing.T) {
	pool := testdb.New(t)
	probe, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := probe.Addr().String()
	require.NoError(t, probe.Close())

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- run(ctx, envOf(validEnv(pool.Config().ConnString(), addr)), quiet) }()

	require.Eventually(t, func() bool {
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+addr+"/healthz", nil)
		if reqErr != nil {
			return false
		}
		resp, doErr := http.DefaultClient.Do(req)
		if doErr != nil {
			return false
		}
		_ = resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 10*time.Second, 50*time.Millisecond, "server never became healthy")

	cancel()
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(15 * time.Second):
		t.Fatal("server did not shut down")
	}
}
