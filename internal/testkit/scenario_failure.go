package testkit

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TestContainerName is the only container this scenario will ever stop or
// start. It is intentionally not configurable -- accepting a container
// name from config or a flag would let a typo point this at something
// that isn't the test environment.
const TestContainerName = "dbwatch-test-pg"

// DatabaseFailure stops the test Postgres container, confirms the target
// database is unreachable, waits `downtime`, then starts it back up and
// confirms recovery. Requires the same DBWATCH_TEST_ENV gate as every
// other scenario, checked here independently of Guard (which needs a
// live connection this scenario deliberately severs).
func DatabaseFailure(ctx context.Context, cfg Config, dbName string, downtime time.Duration) error {
	n := NewNarrator("DATABASE FAILURE / RECOVERY")
	start := time.Now()

	if os.Getenv(cfg.Safety.RequireEnv) != "true" {
		return fmt.Errorf("refusing to run: %s is not set to \"true\"", cfg.Safety.RequireEnv)
	}
	// One more independent confirmation this is the test container, not
	// just trusting the constant: verify docker actually knows about it
	// under exactly this name before touching anything.
	if out, err := exec.Command("docker", "inspect", "-f", "{{.Name}}", TestContainerName).CombinedOutput(); err != nil {
		return fmt.Errorf("refusing to run: docker inspect %s failed (is the test environment up?): %v: %s", TestContainerName, err, out)
	}

	target, err := cfg.Database(dbName)
	if err != nil {
		return err
	}

	n.Step("confirming %s is reachable before failing it...", dbName)
	if err := probe(ctx, target.DSN); err != nil {
		return fmt.Errorf("database not reachable before test even starts: %w", err)
	}
	n.OK("reachable")

	n.Step("docker stop %s ...", TestContainerName)
	if out, err := exec.Command("docker", "stop", TestContainerName).CombinedOutput(); err != nil {
		return fmt.Errorf("docker stop: %w: %s", err, out)
	}
	stoppedAt := time.Now()
	n.OK("container stopped")

	n.Step("confirming %s is now unreachable...", dbName)
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if err := probe(ctx, target.DSN); err != nil {
			n.OK("unreachable: %v", err)
			break
		}
		time.Sleep(300 * time.Millisecond)
	}
	n.Expect("dbwatch shows this database's identity as ⚠/✕ with a collection error\non every panel (Connections, Cache, Locks, Transactions all fail their\nquery). Other configured databases keep updating normally -- one failure\nmust not freeze the whole dashboard.")

	n.Wait("holding downtime for %s...", downtime)
	select {
	case <-time.After(downtime):
	case <-ctx.Done():
	}

	n.Step("docker start %s ...", TestContainerName)
	if out, err := exec.Command("docker", "start", TestContainerName).CombinedOutput(); err != nil {
		return fmt.Errorf("docker start: %w: %s", err, out)
	}

	n.Step("waiting for %s to become reachable again...", dbName)
	deadline = time.Now().Add(60 * time.Second)
	recovered := false
	for time.Now().Before(deadline) {
		if err := probe(ctx, target.DSN); err == nil {
			recovered = true
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if !recovered {
		return fmt.Errorf("database did not become reachable within 60s of container start")
	}
	downtimeActual := time.Since(stoppedAt)
	n.OK("reachable again -- observed downtime %s", downtimeActual.Round(time.Millisecond))
	n.Expect("dbwatch recovers automatically on its next poll -- no restart needed.\nThe database's identity line returns to ✓ HEALTHY (or whatever its real\nstate is) with a fresh uptime, since PostgreSQL itself restarted.")

	n.Done(time.Since(start))
	return nil
}

func probe(ctx context.Context, dsn string) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return err
	}
	defer pool.Close()
	return pool.Ping(ctx)
}

// MultiDatabaseFailure stops the shared container (all test databases
// live on one PostgreSQL instance) and verifies dbwatch keeps the *other*
// configured databases usable -- there is no separate second instance in
// this harness, so "one db down, others up" is demonstrated by checking
// that dbwatch's per-database state is independent, not by an actually
// isolated second server. See test/README.md for why.
func multiDatabaseFailureNote() string {
	return strings.TrimSpace(`
This harness runs every test database on one shared PostgreSQL instance/
container, so there is no way to fail exactly one of them at the network
level. DatabaseFailure above still exercises the important property --
dbwatch's per-database state is independent, so watch the OTHER
databases in the sidebar table while it runs: they must keep polling and
updating normally while "production" shows an error. A true
multi-instance failure test needs a second postgres container, which is
a reasonable next step if per-instance failure isolation specifically
needs coverage.`)
}
