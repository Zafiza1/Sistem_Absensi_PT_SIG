// Package database wires up the PostgreSQL connection pool and the schema
// migration runner used by both cmd/server and cmd/migrate.
package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPool creates a PostgreSQL connection pool and verifies connectivity
// with a bounded retry loop. Retrying matters because in docker-compose the
// backend container frequently starts before Postgres has finished its own
// startup sequence, even with a healthcheck-based `depends_on`.
func NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("database: invalid DATABASE_URL: %w", err)
	}
	poolCfg.MaxConns = 20
	poolCfg.MinConns = 2
	poolCfg.MaxConnLifetime = time.Hour
	poolCfg.HealthCheckPeriod = time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("database: failed to create pool: %w", err)
	}

	const (
		maxAttempts = 10
		retryDelay  = 2 * time.Second
	)

	var pingErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		pingErr = pool.Ping(ctx)
		if pingErr == nil {
			return pool, nil
		}
		if attempt == maxAttempts {
			break
		}
		select {
		case <-ctx.Done():
			pool.Close()
			return nil, ctx.Err()
		case <-time.After(retryDelay):
		}
	}

	pool.Close()
	return nil, fmt.Errorf("database: could not reach postgres after %d attempts: %w", maxAttempts, pingErr)
}
