package testkit

import (
	"context"
	"time"
)

// SlowQuery runs a single pg_sleep query so dbwatch's query and
// connection panels have a real slow statement to observe. It reports its
// own PID before running so you can correlate it against dbwatch's PID
// display live.
func SlowQuery(ctx context.Context, cfg Config, dbName string, duration time.Duration) error {
	n := NewNarrator("SLOW QUERY")
	start := time.Now()

	pool, err := Connect(ctx, cfg, dbName)
	if err != nil {
		return err
	}
	defer pool.Close()

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	var pid int32
	if err := conn.QueryRow(ctx, "SELECT pg_backend_pid()").Scan(&pid); err != nil {
		return err
	}
	n.OK("connected as pid %d on %s", pid, dbName)

	n.Step("running SELECT pg_sleep(%s)...", duration)
	n.Expect("pid %d appears as an active connection for ~%s; once it completes, it will\nshow up in Top Queries (pg_stat_statements) with mean/total time around %s.\nIf %s exceeds the long-transaction threshold, it will also appear under\nLong-Running Transactions while in flight.", pid, duration, duration, duration)

	if _, err := conn.Exec(ctx, "SELECT pg_sleep($1)", duration.Seconds()); err != nil {
		return err
	}
	n.OK("query completed")
	n.Done(time.Since(start))
	return nil
}

// LongTransaction holds an open transaction "active" (a query genuinely
// in flight, via pg_sleep) for the given duration -- distinct from
// IdleInTransaction, which holds a transaction open with nothing
// executing. PostgreSQL reports these as different pg_stat_activity
// states, and dbwatch should show the right one.
func LongTransaction(ctx context.Context, cfg Config, dbName string, duration time.Duration) error {
	n := NewNarrator("LONG-RUNNING TRANSACTION")
	start := time.Now()

	pool, err := Connect(ctx, cfg, dbName)
	if err != nil {
		return err
	}
	defer pool.Close()

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	var pid int32
	if err := tx.QueryRow(ctx, "SELECT pg_backend_pid()").Scan(&pid); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}
	n.OK("BEGIN on pid %d", pid)

	n.Step("holding transaction open (state=active) for %s via pg_sleep...", duration)
	n.Expect("Long-Running Transactions shows pid %d, state \"active\", duration climbing\npast the %s threshold. Locks & Blocking stays clear -- this scenario\nholds no row lock other transactions would want.", pid, cfg.Thresholds.LongTransaction)

	if _, err := tx.Exec(ctx, "SELECT pg_sleep($1)", duration.Seconds()); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}

	n.Step("rolling back (read-only transaction, nothing to commit)...")
	if err := tx.Rollback(ctx); err != nil {
		return err
	}
	n.OK("transaction ended -- dbwatch should show recovery within one poll interval")
	n.Done(time.Since(start))
	return nil
}

// IdleInTransaction opens a transaction, runs one real query, then goes
// quiet without sending anything further -- PostgreSQL reports this as
// state "idle in transaction", not "active".
func IdleInTransaction(ctx context.Context, cfg Config, dbName string, duration time.Duration) error {
	n := NewNarrator("IDLE IN TRANSACTION")
	start := time.Now()

	pool, err := Connect(ctx, cfg, dbName)
	if err != nil {
		return err
	}
	defer pool.Close()

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	var pid int32
	var count int64
	if err := tx.QueryRow(ctx, "SELECT pg_backend_pid()").Scan(&pid); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}
	// best-effort -- table may not exist on every database, that's fine.
	_ = tx.QueryRow(ctx, "SELECT count(*) FROM users").Scan(&count)
	n.OK("BEGIN on pid %d, ran one query (count=%d)", pid, count)

	n.Step("going idle for %s -- no further statements sent...", duration)
	n.Expect("Long-Running Transactions shows pid %d, state \"idle in transaction\"\n(not \"active\") once past the %s threshold -- this is the distinction\ndbwatch needs to get right.", pid, cfg.Thresholds.LongTransaction)

	select {
	case <-time.After(duration):
	case <-ctx.Done():
	}

	n.Step("rolling back...")
	if err := tx.Rollback(ctx); err != nil {
		return err
	}
	n.OK("transaction ended")
	n.Done(time.Since(start))
	return nil
}
