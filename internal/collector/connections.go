// Package collector implements PostgreSQL statistics collectors, each
// pulling a focused slice of state from pg_stat_* / pg_settings.
package collector

import (
	"context"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ConnectionStats is a snapshot of PostgreSQL connection usage.
type ConnectionStats struct {
	Active            int
	Idle              int
	IdleInTransaction int
	Total             int
	MaxConnections    int
}

// UtilizationPercent returns Total/MaxConnections as a 0-100 percentage.
func (s ConnectionStats) UtilizationPercent() float64 {
	if s.MaxConnections == 0 {
		return 0
	}
	return float64(s.Total) / float64(s.MaxConnections) * 100
}

// ConnectionsCollector reads connection counts from pg_stat_activity and
// max_connections from pg_settings.
type ConnectionsCollector struct {
	pool *pgxpool.Pool
}

func NewConnectionsCollector(pool *pgxpool.Pool) *ConnectionsCollector {
	return &ConnectionsCollector{pool: pool}
}

const connectionsQuery = `
SELECT
	count(*) FILTER (WHERE state = 'active')              AS active,
	count(*) FILTER (WHERE state = 'idle')                AS idle,
	count(*) FILTER (WHERE state = 'idle in transaction')  AS idle_in_transaction,
	count(*)                                               AS total
FROM pg_stat_activity
WHERE pid <> pg_backend_pid()
	AND datname = current_database()
`

const maxConnectionsQuery = `SELECT setting FROM pg_settings WHERE name = 'max_connections'`

func (c *ConnectionsCollector) Collect(ctx context.Context) (ConnectionStats, error) {
	var stats ConnectionStats

	row := c.pool.QueryRow(ctx, connectionsQuery)
	if err := row.Scan(&stats.Active, &stats.Idle, &stats.IdleInTransaction, &stats.Total); err != nil {
		return stats, fmt.Errorf("query pg_stat_activity: %w", err)
	}

	var maxConnRaw string
	if err := c.pool.QueryRow(ctx, maxConnectionsQuery).Scan(&maxConnRaw); err != nil {
		return stats, fmt.Errorf("query max_connections: %w", err)
	}
	maxConn, err := strconv.Atoi(maxConnRaw)
	if err != nil {
		return stats, fmt.Errorf("parse max_connections %q: %w", maxConnRaw, err)
	}
	stats.MaxConnections = maxConn

	return stats, nil
}
