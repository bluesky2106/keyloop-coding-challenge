package config

import (
	"strings"
	"testing"
	"time"
)

func env(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestLoad(t *testing.T) {
	secret := strings.Repeat("s", 32)
	tests := []struct {
		name    string
		env     map[string]string
		wantErr string
		check   func(t *testing.T, c Config)
	}{
		{
			name: "defaults",
			env:  map[string]string{"DATABASE_URL": "postgres://x", "JWT_SECRET": secret},
			check: func(t *testing.T, c Config) {
				t.Helper()
				if c.Addr != ":8080" || c.JWTTTL != time.Hour {
					t.Fatalf("unexpected defaults: %+v", c)
				}
			},
		},
		{
			name: "overrides",
			env:  map[string]string{"DATABASE_URL": "postgres://x", "JWT_SECRET": secret, "ADDR": ":9000", "JWT_TTL": "15m"},
			check: func(t *testing.T, c Config) {
				t.Helper()
				if c.Addr != ":9000" || c.JWTTTL != 15*time.Minute {
					t.Fatalf("unexpected config: %+v", c)
				}
			},
		},
		{name: "missing database url", env: map[string]string{"JWT_SECRET": secret}, wantErr: "DATABASE_URL"},
		{name: "short secret", env: map[string]string{"DATABASE_URL": "postgres://x", "JWT_SECRET": "short"}, wantErr: "JWT_SECRET"},
		{name: "bad ttl", env: map[string]string{"DATABASE_URL": "postgres://x", "JWT_SECRET": secret, "JWT_TTL": "soon"}, wantErr: "JWT_TTL"},
		{name: "non-positive ttl", env: map[string]string{"DATABASE_URL": "postgres://x", "JWT_SECRET": secret, "JWT_TTL": "-1h"}, wantErr: "JWT_TTL"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := Load(env(tt.env))
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tt.check(t, c)
		})
	}
}
