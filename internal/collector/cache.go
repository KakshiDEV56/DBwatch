package collector

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// CacheStats is the buffer cache hit/read counts for the current database.
type CacheStats struct {
	BlocksHit  int64
	BlocksRead int64
}

// HitRatio returns the cache hit percentage. An idle database with no reads
// yet reports 100%, since there's nothing to have missed.
func (s CacheStats) HitRatio() float64 {
	total := s.BlocksHit + s.BlocksRead
	if total == 0 {
		return 100
	}
	return float64(s.BlocksHit) / float64(total) * 100
}

type CacheCollector struct {
	pool *pgxpool.Pool
}

func NewCacheCollector(pool *pgxpool.Pool) *CacheCollector {
	return &CacheCollector{pool: pool}
}

const cacheQuery = `
SELECT blks_hit, blks_read
FROM pg_stat_database
WHERE datname = current_database()
`

func (c *CacheCollector) Collect(ctx context.Context) (CacheStats, error) {
	var s CacheStats
	err := c.pool.QueryRow(ctx, cacheQuery).Scan(&s.BlocksHit, &s.BlocksRead)
	if err != nil {
		return s, fmt.Errorf("query pg_stat_database: %w", err)
	}
	return s, nil
}
