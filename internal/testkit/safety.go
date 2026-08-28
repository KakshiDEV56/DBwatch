package testkit

import (
	"context"
	"fmt"
	"os"
	"slices"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Guard is the single choke point every scenario must pass through before
// touching a database. It fails closed: any error here means "refuse to
// run," never "proceed anyway."
//
// Three independent checks, all required:
//  1. The configured environment variable is literally "true" -- a
//     missing or misspelled value refuses, it does not default to safe.
//  2. The connection's target host is in the configured allowlist (catches
//     a DSN accidentally pointed at a real host before we even connect).
//  3. The live `current_database()` the pool actually connected to is in
//     the configured allowlist (catches a DSN that resolves to the right
//     host but the wrong database).
func Guard(ctx context.Context, cfg Config, pool *pgxpool.Pool) error {
	envVar := cfg.Safety.RequireEnv
	if envVar == "" {
		return fmt.Errorf("safety misconfigured: safety.require_env is empty -- refusing to run")
	}
	if os.Getenv(envVar) != "true" {
		return fmt.Errorf("refusing to run: %s is not set to \"true\" (this harness will not touch a database without it)", envVar)
	}

	host := pool.Config().ConnConfig.Host
	if !slices.Contains(cfg.Safety.AllowedHosts, host) {
		return fmt.Errorf("refusing to run: host %q is not in safety.allowed_hosts %v", host, cfg.Safety.AllowedHosts)
	}

	var dbname string
	if err := pool.QueryRow(ctx, "SELECT current_database()").Scan(&dbname); err != nil {
		return fmt.Errorf("refusing to run: could not verify current_database(): %w", err)
	}
	if !slices.Contains(cfg.Safety.AllowedDatabases, dbname) {
		return fmt.Errorf("refusing to run: database %q is not in safety.allowed_databases %v", dbname, cfg.Safety.AllowedDatabases)
	}

	return nil
}
