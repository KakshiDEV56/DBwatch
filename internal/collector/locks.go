package collector

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// BlockedLock describes one blocked/blocking backend pair.
type BlockedLock struct {
	BlockedPID       int32
	BlockedQuery     string
	BlockedApp       string
	BlockedDuration  time.Duration
	BlockingPID      int32
	BlockingQuery    string
	BlockingApp      string
	BlockingDuration time.Duration
}

type LocksCollector struct {
	pool *pgxpool.Pool
}

func NewLocksCollector(pool *pgxpool.Pool) *LocksCollector {
	return &LocksCollector{pool: pool}
}

const locksQuery = `
SELECT
	blocked.pid,
	blocked.query,
	coalesce(blocked.application_name, ''),
	now() - blocked.query_start,
	blocking.pid,
	blocking.query,
	coalesce(blocking.application_name, ''),
	now() - blocking.query_start
FROM pg_stat_activity blocked
JOIN LATERAL unnest(pg_blocking_pids(blocked.pid)) AS bp(pid) ON true
JOIN pg_stat_activity blocking ON blocking.pid = bp.pid
WHERE cardinality(pg_blocking_pids(blocked.pid)) > 0
	AND blocked.datname = current_database()
`

func (c *LocksCollector) Collect(ctx context.Context) ([]BlockedLock, error) {
	rows, err := c.pool.Query(ctx, locksQuery)
	if err != nil {
		return nil, fmt.Errorf("query blocked locks: %w", err)
	}
	defer rows.Close()

	var locks []BlockedLock
	for rows.Next() {
		var l BlockedLock
		if err := rows.Scan(
			&l.BlockedPID, &l.BlockedQuery, &l.BlockedApp, &l.BlockedDuration,
			&l.BlockingPID, &l.BlockingQuery, &l.BlockingApp, &l.BlockingDuration,
		); err != nil {
			return nil, fmt.Errorf("scan blocked lock row: %w", err)
		}
		locks = append(locks, l)
	}
	return locks, rows.Err()
}
