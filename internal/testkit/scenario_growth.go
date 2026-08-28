package testkit

import (
	"context"
	"fmt"
	"time"
)

// Growth inserts rows into growth_test at a steady rate and reports the
// resulting database-size delta. This targets dbwatch_test_stress by
// convention so it never inflates the more "realistic-looking" databases.
func Growth(ctx context.Context, cfg Config, dbName string, rowsPerSecond int, duration time.Duration) error {
	n := NewNarrator("DATABASE GROWTH")
	start := time.Now()

	pool, err := Connect(ctx, cfg, dbName)
	if err != nil {
		return err
	}
	defer pool.Close()

	var before int64
	if err := pool.QueryRow(ctx, "SELECT pg_database_size(current_database())").Scan(&before); err != nil {
		return err
	}
	n.OK("current database size: %s", humanBytes(before))
	n.Step("inserting ~%d rows/sec into growth_test for %s...", rowsPerSecond, duration)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	deadline := time.Now().Add(duration)
	inserted := int64(0)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			deadline = time.Now()
		case <-ticker.C:
			tag, err := pool.Exec(ctx, `
				INSERT INTO growth_test (payload)
				SELECT repeat('x', 200)
				FROM generate_series(1, $1)`, rowsPerSecond)
			if err != nil {
				return fmt.Errorf("insert batch: %w", err)
			}
			inserted += tag.RowsAffected()
		}
	}

	var after int64
	if err := pool.QueryRow(ctx, "SELECT pg_database_size(current_database())").Scan(&after); err != nil {
		return err
	}
	elapsed := time.Since(start)
	rate := float64(after-before) / elapsed.Seconds()

	n.OK("inserted %d rows", inserted)
	n.OK("database size: %s → %s (+%s, %s/s)", humanBytes(before), humanBytes(after), humanBytes(after-before), humanBytes(int64(rate)))
	n.Expect("dbwatch has no storage/growth collector yet -- this size change is real\n(verified here via pg_database_size) but will NOT appear anywhere in the\ncurrent dashboard. See DBWATCH FUNCTIONALITY GAP: storage monitoring.")

	n.Done(elapsed)
	return nil
}

func humanBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}
