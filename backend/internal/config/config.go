// Package config loads and validates process configuration from the environment.
package config

import (
	"errors"
	"fmt"
	"time"
)

const minJWTSecretLen = 32

// Config is the validated runtime configuration.
type Config struct {
	Addr        string
	DatabaseURL string
	JWTSecret   []byte
	JWTTTL      time.Duration
}

// Load reads configuration through getenv (os.Getenv in production) and fails fast
// on anything missing or unsafe.
func Load(getenv func(string) string) (Config, error) {
	cfg := Config{
		Addr:        getenv("ADDR"),
		DatabaseURL: getenv("DATABASE_URL"),
		JWTSecret:   []byte(getenv("JWT_SECRET")),
		JWTTTL:      time.Hour,
	}
	if cfg.Addr == "" {
		cfg.Addr = ":8080"
	}

	var errs []error
	if cfg.DatabaseURL == "" {
		errs = append(errs, errors.New("DATABASE_URL is required"))
	}
	if len(cfg.JWTSecret) < minJWTSecretLen {
		errs = append(errs, fmt.Errorf("JWT_SECRET is required and must be at least %d bytes", minJWTSecretLen))
	}
	if v := getenv("JWT_TTL"); v != "" {
		ttl, err := time.ParseDuration(v)
		if err != nil || ttl <= 0 {
			errs = append(errs, fmt.Errorf("JWT_TTL must be a positive duration such as 1h, got %q", v))
		} else {
			cfg.JWTTTL = ttl
		}
	}
	if err := errors.Join(errs...); err != nil {
		return Config{}, fmt.Errorf("invalid configuration: %w", err)
	}
	return cfg, nil
}
