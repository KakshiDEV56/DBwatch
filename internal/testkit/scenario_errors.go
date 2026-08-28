package testkit

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// QueryErrors runs a fixed set of intentionally-failing statements against
// the test database: syntax error, missing table, missing column, unique
// violation, foreign-key violation, invalid type, and (using the
// restricted role created by the test schema) a permission error.
//
// Every failure is caught and printed here -- none of it is fabricated as
// if it came from dbwatch or from PostgreSQL's server log. dbwatch has no
// mechanism to consume PostgreSQL's server log today (see the
// "DBWATCH FUNCTIONALITY GAP" note this scenario prints), so these
// failures are visible only in this harness's own output and, where
// applicable, in pg_stat_statements -- not as dbwatch events.
func QueryErrors(ctx context.Context, cfg Config, dbName string) error {
	n := NewNarrator("QUERY ERRORS")
	start := time.Now()

	pool, err := Connect(ctx, cfg, dbName)
	if err != nil {
		return err
	}
	defer pool.Close()

	cases := []struct {
		name string
		sql  string
		args []any
	}{
		{"syntax error", "SELEC * FROM users", nil},
		{"missing table", "SELECT * FROM does_not_exist", nil},
		{"missing column", "SELECT nonexistent_column FROM users LIMIT 1", nil},
		{"unique violation", "INSERT INTO users (email) SELECT email FROM users LIMIT 1", nil},
		{"foreign-key violation", "INSERT INTO orders (user_id, total_cents, status) VALUES (999999999, 100, 'pending')", nil},
		{"invalid type", "INSERT INTO users (email) VALUES (12345)", nil},
	}

	for _, c := range cases {
		_, err := pool.Exec(ctx, c.sql, c.args...)
		if err != nil {
			n.OK("%-22s → %v", c.name, err)
		} else {
			n.Fail("%-22s → expected an error, statement succeeded", c.name)
		}
	}

	n.Step("permission error (via restricted role dbwatch_test_restricted)...")
	if err := permissionErrorCase(ctx, cfg, dbName, n); err != nil {
		n.Warn("permission-error case skipped: %v", err)
	}

	n.Expect("None of these are individually-failed-query events in dbwatch today --\nthere is no failed-query panel or server-log ingestion. Syntax/missing-\ntable/missing-column errors never reach pg_stat_statements at all (they\nfail before planning). Constraint violations may nudge pg_stat_database's\nerror-adjacent counters but are not surfaced anywhere in the dashboard.\nSee DBWATCH FUNCTIONALITY GAP: query-level error visibility, in the\nfinal report.")

	n.Done(time.Since(start))
	return nil
}

func permissionErrorCase(ctx context.Context, cfg Config, dbName string, n *Narrator) error {
	target, err := cfg.Database(dbName)
	if err != nil {
		return err
	}
	poolCfg, err := pgxpool.ParseConfig(target.DSN)
	if err != nil {
		return err
	}
	poolCfg.ConnConfig.User = "dbwatch_test_restricted"
	poolCfg.ConnConfig.Password = "dbwatch"

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := Guard(ctx, cfg, pool); err != nil {
		return err
	}

	_, execErr := pool.Exec(ctx, "SELECT * FROM payments LIMIT 1")
	if execErr != nil {
		n.OK("%-22s → %v", "permission error", execErr)
	} else {
		n.Fail("%-22s → expected a permission error, statement succeeded", "permission error")
	}
	return nil
}
