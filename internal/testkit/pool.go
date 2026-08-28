package testkit

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Connect opens a pool for a configured database target and runs it
// through Guard before returning -- every caller gets a pool that has
// already been verified safe.
func Connect(ctx context.Context, cfg Config, dbName string) (*pgxpool.Pool, error) {
	target, err := cfg.Database(dbName)
	if err != nil {
		return nil, err
	}
	pool, err := pgxpool.New(ctx, target.DSN)
	if err != nil {
		return nil, fmt.Errorf("connect to %s: %w", dbName, err)
	}
	if err := Guard(ctx, cfg, pool); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}
