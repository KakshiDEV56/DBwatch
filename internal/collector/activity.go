package collector

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ActivityStats is a cumulative snapshot from pg_stat_database -- counters
// since the stats were last reset, not point-in-time rates. The caller
// (see tui.DBState) diffs consecutive snapshots to derive per-second
// rates, the way any counter-based metrics system works.
//
// This is deliberately NOT a CPU-usage metric. PostgreSQL's own
// statistics views expose transaction/tuple/temp-file activity inside the
// database, but nothing about the host's CPU, memory, or disk hardware --
// that's OS-level data no SQL query can produce. Getting real CPU% needs
// either an agent running on the database host (reading /proc or
// equivalent) or a cloud provider's own monitoring API. What's here is
// the honest, SQL-visible proxy for "how much work is this database
// doing": transaction rate, tuple throughput, and temp-file spillage.
type ActivityStats struct {
	XactCommit   int64
	XactRollback int64
	TupReturned  int64
	TupFetched   int64
	TupInserted  int64
	TupUpdated   int64
	TupDeleted   int64
	TempFiles    int64
	TempBytes    int64
	Deadlocks    int64
}

type ActivityCollector struct {
	pool *pgxpool.Pool
}

func NewActivityCollector(pool *pgxpool.Pool) *ActivityCollector {
	return &ActivityCollector{pool: pool}
}

const activityQuery = `
SELECT
	xact_commit, xact_rollback,
	tup_returned, tup_fetched, tup_inserted, tup_updated, tup_deleted,
	temp_files, temp_bytes, deadlocks
FROM pg_stat_database
WHERE datname = current_database()
`

func (c *ActivityCollector) Collect(ctx context.Context) (ActivityStats, error) {
	var s ActivityStats
	err := c.pool.QueryRow(ctx, activityQuery).Scan(
		&s.XactCommit, &s.XactRollback,
		&s.TupReturned, &s.TupFetched, &s.TupInserted, &s.TupUpdated, &s.TupDeleted,
		&s.TempFiles, &s.TempBytes, &s.Deadlocks,
	)
	if err != nil {
		return s, fmt.Errorf("query pg_stat_database activity: %w", err)
	}
	return s, nil
}
