package testkit

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// maxSafeConnectionFraction caps how far this scenario will push toward
// max_connections -- it must threaten the threshold, never the ceiling
// itself, so a real client is never locked out.
const maxSafeConnectionFraction = 0.8

// ConnectionPressure opens `workers` concurrent connections and holds
// each one active (via pg_sleep) for `duration`, to push connection
// utilization toward -- but never past -- a safe fraction of
// max_connections.
func ConnectionPressure(ctx context.Context, cfg Config, dbName string, workers int, duration time.Duration) error {
	n := NewNarrator("CONNECTION PRESSURE")
	start := time.Now()

	guardPool, err := Connect(ctx, cfg, dbName)
	if err != nil {
		return err
	}

	var maxConn int
	if err := guardPool.QueryRow(ctx, "SELECT current_setting('max_connections')::int").Scan(&maxConn); err != nil {
		guardPool.Close()
		return err
	}
	safeCap := int(float64(maxConn) * maxSafeConnectionFraction)
	if workers > safeCap {
		n.Warn("requested %d workers exceeds safe cap (%.0f%% of max_connections=%d) -- clamping to %d",
			workers, maxSafeConnectionFraction*100, maxConn, safeCap)
		workers = safeCap
	}
	guardPool.Close()

	target, err := cfg.Database(dbName)
	if err != nil {
		return err
	}
	poolCfg, err := pgxpool.ParseConfig(target.DSN)
	if err != nil {
		return err
	}
	poolCfg.MaxConns = int32(workers) + 2
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := Guard(ctx, cfg, pool); err != nil {
		return err
	}

	n.Step("opening %d connections (max_connections=%d, safe cap %.0f%%)...", workers, maxConn, maxSafeConnectionFraction*100)
	n.Expect("Connections utilization climbs toward ~%.0f%%. dbwatch should transition\n✓ healthy → ⚠ warning (≥70%%) → ▲ critical (≥90%%) as configured thresholds\nare crossed, then back to ✓ healthy once these connections close.",
		float64(workers)/float64(maxConn)*100)

	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			conn, err := pool.Acquire(ctx)
			if err != nil {
				errs <- err
				return
			}
			defer conn.Release()
			if _, err := conn.Exec(ctx, "SELECT pg_sleep($1)", duration.Seconds()); err != nil {
				errs <- err
			}
		}()
	}

	n.Wait("holding %d connections active for %s...", workers, duration)
	wg.Wait()
	close(errs)

	failed := 0
	for err := range errs {
		failed++
		if failed <= 3 {
			n.Warn("worker error: %v", err)
		}
	}
	if failed > 0 {
		n.Warn("%d/%d workers reported an error", failed, workers)
	} else {
		n.OK("all %d workers completed cleanly", workers)
	}
	n.Expect("Connections utilization drops back down within one poll interval.")

	n.Done(time.Since(start))
	if failed == workers {
		return fmt.Errorf("all %d workers failed", workers)
	}
	return nil
}
