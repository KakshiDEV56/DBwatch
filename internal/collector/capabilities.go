package collector

import (
	"context"
	"fmt"
	"slices"

	"github.com/jackc/pgx/v5/pgxpool"
)

// CapabilityStats describes what this server actually exposes to, and
// permits for, the connected role. dbwatch's other panels are only ever as
// complete as these capabilities allow (a role without pg_read_all_stats
// sees only its own backends in pg_stat_activity, pg_stat_statements may
// not be loaded at all, statistics collection can be turned off in
// postgresql.conf) -- this panel exists to make those limits visible
// instead of silently under-reporting.
type CapabilityStats struct {
	IsSuperuser               bool
	HasReadAllStats           bool
	TrackActivities           bool
	TrackCounts               bool
	TrackIOTiming             bool
	PgStatStatementsAvailable bool
	Extensions                []string
}

func (s CapabilityStats) HasExtension(name string) bool {
	return slices.Contains(s.Extensions, name)
}

type CapabilityCollector struct {
	pool *pgxpool.Pool
}

func NewCapabilityCollector(pool *pgxpool.Pool) *CapabilityCollector {
	return &CapabilityCollector{pool: pool}
}

// capabilityQuery reads only system catalogs and current_setting(), never
// pg_has_role() -- so it works back to PostgreSQL 10 (when pg_read_all_stats
// was introduced) without erroring on a role name a given server's version
// might not recognize.
const capabilityQuery = `
SELECT
	COALESCE((SELECT rolsuper FROM pg_roles WHERE rolname = current_user), false),
	EXISTS (
		SELECT 1
		FROM pg_auth_members m
		JOIN pg_roles r ON r.oid = m.roleid
		JOIN pg_roles u ON u.oid = m.member
		WHERE r.rolname = 'pg_read_all_stats' AND u.rolname = current_user
	),
	current_setting('track_activities')::bool,
	current_setting('track_counts')::bool,
	current_setting('track_io_timing')::bool,
	EXISTS (SELECT 1 FROM pg_available_extensions WHERE name = 'pg_stat_statements')
`

const extensionsQuery = `SELECT extname FROM pg_extension ORDER BY extname`

func (c *CapabilityCollector) Collect(ctx context.Context) (CapabilityStats, error) {
	var s CapabilityStats
	err := c.pool.QueryRow(ctx, capabilityQuery).Scan(
		&s.IsSuperuser, &s.HasReadAllStats,
		&s.TrackActivities, &s.TrackCounts, &s.TrackIOTiming,
		&s.PgStatStatementsAvailable,
	)
	if err != nil {
		return s, fmt.Errorf("query server capabilities: %w", err)
	}

	rows, err := c.pool.Query(ctx, extensionsQuery)
	if err != nil {
		return s, fmt.Errorf("query pg_extension: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return s, fmt.Errorf("scan pg_extension row: %w", err)
		}
		s.Extensions = append(s.Extensions, name)
	}
	return s, rows.Err()
}
