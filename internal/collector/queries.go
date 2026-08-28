package collector

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// QueryStat is one row from pg_stat_statements.
type QueryStat struct {
	Query       string
	Calls       int64
	TotalExecMs float64
	MeanExecMs  float64
	Rows        int64
}

type QueriesCollector struct {
	pool  *pgxpool.Pool
	limit int
}

func NewQueriesCollector(pool *pgxpool.Pool, limit int) *QueriesCollector {
	return &QueriesCollector{pool: pool, limit: limit}
}

// EnsureExtension creates pg_stat_statements if it isn't already present.
// It requires the module to already be loaded via the server's
// shared_preload_libraries setting (a config change + restart) — CREATE
// EXTENSION alone cannot do that part, so a clear error is returned if the
// module isn't loaded rather than failing silently later.
func (c *QueriesCollector) EnsureExtension(ctx context.Context) error {
	_, err := c.pool.Exec(ctx, `CREATE EXTENSION IF NOT EXISTS pg_stat_statements`)
	if err != nil {
		return fmt.Errorf("enable pg_stat_statements (requires shared_preload_libraries='pg_stat_statements' on the server, then a restart): %w", err)
	}
	return nil
}

// pg_stat_statements is shared across every database in the cluster
// (keyed by dbid), so it must be joined against pg_database and filtered
// to the current one — otherwise queries run against unrelated databases
// on the same server would show up here.
const queriesQuery = `
SELECT s.query, s.calls, s.total_exec_time, s.mean_exec_time, s.rows
FROM pg_stat_statements s
JOIN pg_database d ON d.oid = s.dbid
WHERE d.datname = current_database()
ORDER BY s.total_exec_time DESC
LIMIT $1
`

func (c *QueriesCollector) Collect(ctx context.Context) ([]QueryStat, error) {
	rows, err := c.pool.Query(ctx, queriesQuery, c.limit)
	if err != nil {
		return nil, fmt.Errorf("query pg_stat_statements: %w", err)
	}
	defer rows.Close()

	var stats []QueryStat
	for rows.Next() {
		var s QueryStat
		if err := rows.Scan(&s.Query, &s.Calls, &s.TotalExecMs, &s.MeanExecMs, &s.Rows); err != nil {
			return nil, fmt.Errorf("scan pg_stat_statements row: %w", err)
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}
