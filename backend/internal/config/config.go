// Package config loads application configuration from environment variables.
//
// All configuration is read from the process environment. In local
// development, main.go loads a .env file (if present) into the process
// environment before Load() is called; in Docker/production the environment
// is provided directly by the container runtime, so no .env file is needed
// there.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all runtime configuration for the backend service.
type Config struct {
	// AppEnv is one of "development", "staging", "production".
	AppEnv string
	// AppPort is the TCP port the HTTP server listens on.
	AppPort string

	// DatabaseURL is a full PostgreSQL connection string, e.g.
	// postgres://user:password@host:5432/dbname?sslmode=disable
	DatabaseURL string

	// JWTSecret signs and verifies access/refresh tokens (Phase 2).
	JWTSecret string
	// AccessTokenTTL / RefreshTokenTTL control JWT lifetimes (Phase 2).
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration

	// LogLevel is one of "debug", "info", "warn", "error".
	LogLevel string

	// AllowedOrigins is the CORS allowlist for the web dashboard.
	AllowedOrigins []string

	// AutoMigrate runs pending database migrations automatically on startup
	// when true. Useful for local development and containerized deploys;
	// disable in environments where migrations are applied out-of-band.
	AutoMigrate bool
}

// Load reads configuration from environment variables, applying sane
// defaults for local development. It returns an error if a required
// production-critical value is missing.
func Load() (*Config, error) {
	cfg := &Config{
		AppEnv:          getEnv("APP_ENV", "development"),
		AppPort:         getEnv("APP_PORT", "8080"),
		DatabaseURL:     getEnv("DATABASE_URL", ""),
		JWTSecret:       getEnv("JWT_SECRET", ""),
		AccessTokenTTL:  getEnvDuration("ACCESS_TOKEN_TTL", 15*time.Minute),
		RefreshTokenTTL: getEnvDuration("REFRESH_TOKEN_TTL", 7*24*time.Hour),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
		AllowedOrigins:  getEnvList("ALLOWED_ORIGINS", []string{"http://localhost:3000"}),
		AutoMigrate:     getEnvBool("AUTO_MIGRATE", true),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("config: DATABASE_URL is required")
	}
	if cfg.AppEnv == "production" && cfg.JWTSecret == "" {
		return nil, fmt.Errorf("config: JWT_SECRET is required in production")
	}
	if cfg.JWTSecret == "" {
		// Non-production convenience default. Never used when AppEnv is
		// "production" because of the check above.
		cfg.JWTSecret = "dev-only-insecure-secret-change-me"
	}

	return cfg, nil
}

// IsProduction reports whether the service is running in production mode.
func (c *Config) IsProduction() bool {
	return c.AppEnv == "production"
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}

func getEnvList(key string, fallback []string) []string {
	v, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(v) == "" {
		return fallback
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return fallback
	}
	return out
}
