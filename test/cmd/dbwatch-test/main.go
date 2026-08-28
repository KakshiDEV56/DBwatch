// Command dbwatch-test is dbwatch's chaos/workload test harness. It
// generates realistic PostgreSQL activity and failure conditions against
// a dedicated local test environment (test/postgres/) so that dbwatch's
// detection can be verified against real server behavior.
//
// Every subcommand refuses to run unless DBWATCH_TEST_ENV=true and the
// target database is one explicitly listed in test/dbwatch-test.yaml --
// see internal/testkit/safety.go.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"dbwatch/internal/testkit"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := dispatch(ctx, os.Args[1], os.Args[2:]); err != nil {
		fmt.Fprintln(os.Stderr, "dbwatch-test:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `Usage: dbwatch-test <command> [flags]

Commands:
  seed                    populate the schema with test data
  workload                run a continuous mixed query load
  slow-query              run a single slow query
  long-tx                 hold a long-running (active) transaction
  idle-tx                 hold an idle-in-transaction session
  locks                   demonstrate lock contention + recovery
  deadlock                trigger a real deadlock
  errors                  run a battery of intentionally-failing queries
  connections             push connection utilization up, then release
  growth                  generate database growth
  database-failure        stop/start the test container, verify detection + recovery
  scenario <name>         run one named scenario (see: scenario --list)
  scenario all            run every scenario in sequence with a summary

All commands accept -config (default test/dbwatch-test.yaml) and -db
(default: production, except growth which defaults to stress).

Every command requires DBWATCH_TEST_ENV=true in the environment -- see
test/README.md.`)
}

func dispatch(ctx context.Context, cmd string, args []string) error {
	switch cmd {
	case "seed":
		return cmdSeed(ctx, args)
	case "workload":
		return cmdWorkload(ctx, args)
	case "slow-query":
		return cmdSlowQuery(ctx, args)
	case "long-tx":
		return cmdLongTx(ctx, args)
	case "idle-tx":
		return cmdIdleTx(ctx, args)
	case "locks":
		return cmdLocks(ctx, args)
	case "deadlock":
		return cmdDeadlock(ctx, args)
	case "errors":
		return cmdErrors(ctx, args)
	case "connections":
		return cmdConnections(ctx, args)
	case "growth":
		return cmdGrowth(ctx, args)
	case "database-failure":
		return cmdDatabaseFailure(ctx, args)
	case "scenario":
		return cmdScenario(ctx, args)
	case "-h", "--help", "help":
		usage()
		return nil
	default:
		usage()
		return fmt.Errorf("unknown command %q", cmd)
	}
}

// commonFlags wires -config and -db onto a FlagSet and returns pointers,
// leaving fs.Parse to the caller.
func commonFlags(fs *flag.FlagSet, defaultDB string) (*string, *string) {
	config := fs.String("config", "test/dbwatch-test.yaml", "path to test config")
	db := fs.String("db", defaultDB, "database name from test config")
	return config, db
}

func cmdSeed(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("seed", flag.ExitOnError)
	config, db := commonFlags(fs, "production")
	users := fs.Int("users", 0, "user rows (0 = default)")
	products := fs.Int("products", 0, "product rows (0 = default)")
	orders := fs.Int("orders", 0, "order rows (0 = default)")
	orderItems := fs.Int("order-items", 0, "order_item rows (0 = default)")
	payments := fs.Int("payments", 0, "payment rows (0 = default)")
	sessions := fs.Int("sessions", 0, "session rows (0 = default)")
	events := fs.Int("events", 0, "event rows (0 = default)")
	auditLogs := fs.Int("audit-logs", 0, "audit_log rows (0 = default)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := testkit.Load(*config)
	if err != nil {
		return err
	}
	counts := testkit.DefaultSeedCounts()
	overrideIfSet(&counts.Users, *users)
	overrideIfSet(&counts.Products, *products)
	overrideIfSet(&counts.Orders, *orders)
	overrideIfSet(&counts.OrderItems, *orderItems)
	overrideIfSet(&counts.Payments, *payments)
	overrideIfSet(&counts.Sessions, *sessions)
	overrideIfSet(&counts.Events, *events)
	overrideIfSet(&counts.AuditLogs, *auditLogs)

	pool, err := testkit.Connect(ctx, cfg, *db)
	if err != nil {
		return err
	}
	defer pool.Close()

	n := testkit.NewNarrator("SEED")
	start := time.Now()
	if err := testkit.Seed(ctx, pool, n, counts); err != nil {
		return err
	}
	n.Done(time.Since(start))
	return nil
}

func overrideIfSet(target *int, flagVal int) {
	if flagVal > 0 {
		*target = flagVal
	}
}

func cmdWorkload(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("workload", flag.ExitOnError)
	config, db := commonFlags(fs, "production")
	workers := fs.Int("workers", 10, "concurrent workers")
	qps := fs.Int("qps", 50, "target queries/sec")
	duration := fs.Duration("duration", 5*time.Minute, "how long to run")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg, err := testkit.Load(*config)
	if err != nil {
		return err
	}
	report, err := testkit.Workload(ctx, cfg, *db, *workers, *qps, *duration)
	if err != nil {
		return err
	}
	fmt.Printf("\nworkers:      %d\ntarget QPS:   %d\nactual QPS:   %.1f\nduration:     %s\nsuccessful:   %d\nfailed:       %d\n",
		report.Workers, report.TargetQPS, report.ActualQPS, report.Duration, report.Successful, report.Failed)
	return nil
}

func cmdSlowQuery(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("slow-query", flag.ExitOnError)
	config, db := commonFlags(fs, "production")
	duration := fs.Duration("duration", 10*time.Second, "how long the query sleeps")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg, err := testkit.Load(*config)
	if err != nil {
		return err
	}
	return testkit.SlowQuery(ctx, cfg, *db, *duration)
}

func cmdLongTx(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("long-tx", flag.ExitOnError)
	config, db := commonFlags(fs, "production")
	duration := fs.Duration("duration", 60*time.Second, "how long to hold the transaction")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg, err := testkit.Load(*config)
	if err != nil {
		return err
	}
	return testkit.LongTransaction(ctx, cfg, *db, *duration)
}

func cmdIdleTx(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("idle-tx", flag.ExitOnError)
	config, db := commonFlags(fs, "production")
	duration := fs.Duration("duration", 60*time.Second, "how long to stay idle in transaction")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg, err := testkit.Load(*config)
	if err != nil {
		return err
	}
	return testkit.IdleInTransaction(ctx, cfg, *db, *duration)
}

func cmdLocks(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("locks", flag.ExitOnError)
	config, db := commonFlags(fs, "production")
	duration := fs.Duration("duration", 20*time.Second, "how long to hold the blocking lock")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg, err := testkit.Load(*config)
	if err != nil {
		return err
	}
	return testkit.LockContention(ctx, cfg, *db, *duration)
}

func cmdDeadlock(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("deadlock", flag.ExitOnError)
	config, db := commonFlags(fs, "production")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg, err := testkit.Load(*config)
	if err != nil {
		return err
	}
	return testkit.Deadlock(ctx, cfg, *db)
}

func cmdErrors(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("errors", flag.ExitOnError)
	config, db := commonFlags(fs, "production")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg, err := testkit.Load(*config)
	if err != nil {
		return err
	}
	return testkit.QueryErrors(ctx, cfg, *db)
}

func cmdConnections(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("connections", flag.ExitOnError)
	config, db := commonFlags(fs, "production")
	workers := fs.Int("workers", 50, "connections to hold open")
	duration := fs.Duration("duration", 30*time.Second, "how long to hold them")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg, err := testkit.Load(*config)
	if err != nil {
		return err
	}
	return testkit.ConnectionPressure(ctx, cfg, *db, *workers, *duration)
}

func cmdGrowth(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("growth", flag.ExitOnError)
	config, db := commonFlags(fs, "stress")
	rate := fs.Int("rate", 500, "rows inserted per second")
	duration := fs.Duration("duration", 60*time.Second, "how long to run")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg, err := testkit.Load(*config)
	if err != nil {
		return err
	}
	return testkit.Growth(ctx, cfg, *db, *rate, *duration)
}

func cmdDatabaseFailure(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("database-failure", flag.ExitOnError)
	config, db := commonFlags(fs, "production")
	downtime := fs.Duration("downtime", 15*time.Second, "how long to keep the container stopped")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg, err := testkit.Load(*config)
	if err != nil {
		return err
	}
	return testkit.DatabaseFailure(ctx, cfg, *db, *downtime)
}

func cmdScenario(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("scenario", flag.ExitOnError)
	config := fs.String("config", "test/dbwatch-test.yaml", "path to test config")
	list := fs.Bool("list", false, "list available scenario names")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *list {
		fmt.Println(strings.Join(testkit.ScenarioNames(), "\n"))
		return nil
	}

	rest := fs.Args()
	if len(rest) == 0 {
		return fmt.Errorf("usage: dbwatch-test scenario <name|all> (see: scenario --list)")
	}
	cfg, err := testkit.Load(*config)
	if err != nil {
		return err
	}

	if rest[0] == "all" {
		results := testkit.RunAll(ctx, cfg)
		for _, r := range results {
			if r.Err != nil {
				return fmt.Errorf("%d scenario(s) failed", countFailed(results))
			}
		}
		return nil
	}
	return testkit.RunNamed(ctx, cfg, rest[0])
}

func countFailed(results []testkit.RunResult) int {
	n := 0
	for _, r := range results {
		if r.Err != nil {
			n++
		}
	}
	return n
}
