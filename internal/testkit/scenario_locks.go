package testkit

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// LockContention holds a row lock on transaction A and forces transaction
// B to block on the same row, for `duration`, then releases A. Targets
// accounts.id=1, matching the example throughout the spec.
func LockContention(ctx context.Context, cfg Config, dbName string, duration time.Duration) error {
	n := NewNarrator("LOCK CONTENTION")
	start := time.Now()

	pool, err := Connect(ctx, cfg, dbName)
	if err != nil {
		return err
	}
	defer pool.Close()

	connA, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer connA.Release()
	connB, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer connB.Release()

	n.Step("creating transaction A...")
	txA, err := connA.Begin(ctx)
	if err != nil {
		return err
	}
	var pidA int32
	if err := txA.QueryRow(ctx, "SELECT pg_backend_pid()").Scan(&pidA); err != nil {
		return err
	}
	n.OK("pid %d", pidA)

	n.Step("acquiring lock on accounts.id=1...")
	if _, err := txA.Exec(ctx, "UPDATE accounts SET balance_cents = balance_cents + 100 WHERE id = 1"); err != nil {
		_ = txA.Rollback(ctx)
		return err
	}
	n.OK("lock acquired")

	n.Step("creating transaction B...")
	txB, err := connB.Begin(ctx)
	if err != nil {
		_ = txA.Rollback(ctx)
		return err
	}
	var pidB int32
	if err := txB.QueryRow(ctx, "SELECT pg_backend_pid()").Scan(&pidB); err != nil {
		_ = txA.Rollback(ctx)
		return err
	}
	n.OK("pid %d", pidB)

	n.Step("attempting conflicting update from B (this will block)...")
	blockDone := make(chan error, 1)
	go func() {
		_, err := txB.Exec(ctx, "UPDATE accounts SET balance_cents = balance_cents - 100 WHERE id = 1")
		blockDone <- err
	}()

	time.Sleep(500 * time.Millisecond)
	blocking, err := confirmBlocked(ctx, pool, pidB)
	if err != nil {
		n.Warn("could not confirm block via pg_blocking_pids: %v", err)
	} else if blocking {
		n.OK("transaction B is now blocked by pid %d", pidA)
	} else {
		n.Warn("pg_blocking_pids did not report pid %d as blocked -- update may have completed already", pidB)
	}

	n.Expect("Locks & Blocking shows \"pid %d blocked by pid %d\", with blocked-waiting\nduration climbing toward %s.", pidB, pidA, duration)

	n.Wait("holding for %s before releasing A...", duration)
	select {
	case <-time.After(duration):
	case <-ctx.Done():
	}

	n.Step("cleaning up...")
	if err := txA.Rollback(ctx); err != nil {
		n.Warn("rollback A: %v", err)
	} else {
		n.OK("transaction A rolled back -- lock released")
	}

	select {
	case err := <-blockDone:
		if err != nil {
			n.Warn("transaction B's update returned: %v", err)
		} else {
			n.OK("transaction B's update completed")
		}
	case <-time.After(10 * time.Second):
		n.Fail("transaction B did not unblock within 10s of A releasing")
	}
	if err := txB.Rollback(ctx); err != nil {
		n.Warn("rollback B: %v", err)
	} else {
		n.OK("transaction B rolled back")
	}
	n.OK("locks released")
	n.Expect("Locks & Blocking warning disappears within one poll interval.")

	n.Done(time.Since(start))
	return nil
}

func confirmBlocked(ctx context.Context, pool *pgxpool.Pool, pid int32) (bool, error) {
	var n int
	err := pool.QueryRow(ctx, "SELECT cardinality(pg_blocking_pids($1))", pid).Scan(&n)
	return n > 0, err
}

// Deadlock makes transaction A lock row 1 then wait on row 2, while
// transaction B locks row 2 then waits on row 1 -- a classic deadlock.
// PostgreSQL's own deadlock detector (deadlock_timeout, ~1s by default)
// aborts one side automatically; nothing here can hang indefinitely
// because Postgres itself guarantees resolution, and a per-statement
// lock_timeout is set as a defensive second layer regardless.
func Deadlock(ctx context.Context, cfg Config, dbName string) error {
	n := NewNarrator("DEADLOCK")
	start := time.Now()

	pool, err := Connect(ctx, cfg, dbName)
	if err != nil {
		return err
	}
	defer pool.Close()

	connA, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer connA.Release()
	connB, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer connB.Release()

	txA, err := connA.Begin(ctx)
	if err != nil {
		return err
	}
	txB, err := connB.Begin(ctx)
	if err != nil {
		_ = txA.Rollback(ctx)
		return err
	}
	_, _ = txA.Exec(ctx, "SET LOCAL lock_timeout = '10s'")
	_, _ = txB.Exec(ctx, "SET LOCAL lock_timeout = '10s'")

	var pidA, pidB int32
	_ = txA.QueryRow(ctx, "SELECT pg_backend_pid()").Scan(&pidA)
	_ = txB.QueryRow(ctx, "SELECT pg_backend_pid()").Scan(&pidB)
	n.OK("transaction A pid %d, transaction B pid %d", pidA, pidB)

	n.Step("A locks accounts.id=1...")
	if _, err := txA.Exec(ctx, "UPDATE accounts SET balance_cents = balance_cents + 1 WHERE id = 1"); err != nil {
		_ = txA.Rollback(ctx)
		_ = txB.Rollback(ctx)
		return err
	}
	n.Step("B locks accounts.id=2...")
	if _, err := txB.Exec(ctx, "UPDATE accounts SET balance_cents = balance_cents + 1 WHERE id = 2"); err != nil {
		_ = txA.Rollback(ctx)
		_ = txB.Rollback(ctx)
		return err
	}

	n.Step("A now waits for row 2 (held by B), B now waits for row 1 (held by A)...")
	barrier := make(chan struct{})
	resA := make(chan error, 1)
	resB := make(chan error, 1)
	go func() {
		<-barrier
		_, err := txA.Exec(ctx, "UPDATE accounts SET balance_cents = balance_cents + 1 WHERE id = 2")
		resA <- err
	}()
	go func() {
		<-barrier
		_, err := txB.Exec(ctx, "UPDATE accounts SET balance_cents = balance_cents + 1 WHERE id = 1")
		resB <- err
	}()
	close(barrier)

	n.Wait("waiting for PostgreSQL's deadlock detector (deadlock_timeout)...")
	errA, errB := <-resA, <-resB

	switch {
	case errA != nil && errB == nil:
		n.OK("PostgreSQL detected the deadlock and aborted pid %d:\n  %s", pidA, errA)
		n.OK("pid %d (the other side) proceeded", pidB)
	case errB != nil && errA == nil:
		n.OK("PostgreSQL detected the deadlock and aborted pid %d:\n  %s", pidB, errB)
		n.OK("pid %d (the other side) proceeded", pidA)
	case errA != nil && errB != nil:
		n.Warn("both sides errored (unexpected but not unsafe): A=%v B=%v", errA, errB)
	default:
		n.Fail("neither side errored -- deadlock did not occur as expected")
	}

	n.Step("rolling back both sides regardless of outcome...")
	_ = txA.Rollback(ctx)
	_ = txB.Rollback(ctx)
	n.OK("cleaned up -- accounts table unchanged")

	n.Expect("PostgreSQL resolves a deadlock in well under a second (deadlock_timeout,\ndefault 1s). dbwatch polls on an interval, so it will very likely NOT\ncatch this window live -- this is an honest limitation of polling-based\nlock monitoring, not a bug. It should show the transient lock contention\nonly if a poll happens to land inside that ~1s window.")

	n.Done(time.Since(start))
	if errA == nil && errB == nil {
		return fmt.Errorf("expected a deadlock, neither transaction was aborted")
	}
	return nil
}
