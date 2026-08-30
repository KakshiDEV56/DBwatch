package collector

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SizeStats is real, queryable disk usage for the current database --
// nothing here is estimated or sampled.
type SizeStats struct {
	DatabaseBytes int64 // pg_database_size(current_database()) -- every relation plus catalogs, excludes WAL
	TablesBytes   int64 // sum of pg_table_size() over user tables: heap + toast + free space map, excludes indexes
	IndexesBytes  int64 // sum of pg_indexes_size() over user tables
}

type SizeCollector struct {
	pool *pgxpool.Pool
}

func NewSizeCollector(pool *pgxpool.Pool) *SizeCollector {
	return &SizeCollector{pool: pool}
}

// sizeQuery sums over user tables only (excluding pg_catalog,
// information_schema, and pg_toast, whose sizes are already folded into
// DatabaseBytes via pg_database_size) so TablesBytes+IndexesBytes reads as
// "your data" rather than the whole cluster-internal footprint.
const sizeQuery = `
SELECT
	pg_database_size(current_database()),
	COALESCE(SUM(pg_table_size(c.oid)), 0),
	COALESCE(SUM(pg_indexes_size(c.oid)), 0)
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE c.relkind = 'r'
  AND n.nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
`

func (c *SizeCollector) Collect(ctx context.Context) (SizeStats, error) {
	var s SizeStats
	err := c.pool.QueryRow(ctx, sizeQuery).Scan(&s.DatabaseBytes, &s.TablesBytes, &s.IndexesBytes)
	if err != nil {
		return s, fmt.Errorf("query database size: %w", err)
	}
	return s, nil
}
