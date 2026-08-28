// Command dbwatch monitors PostgreSQL databases and renders an
// observability dashboard in the terminal.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"dbwatch/internal/collector"
	"dbwatch/internal/config"
	"dbwatch/internal/tui"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "start":
		if err := runStart(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "dbwatch:", err)
			os.Exit(1)
		}
	case "-h", "--help", "help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "dbwatch: unknown command %q\n\n", os.Args[1])
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `Usage: dbwatch start [flags]

Flags:
  -config string    path to config file (default "dbwatch.yaml")
  -dsn string        Postgres DSN, adds/overrides the "default" database entry
  -interval duration polling interval, overrides config file (default 10s)

Config supports either a single "database:" block or a "databases:" list —
see dbwatch.example.yaml.`)
}

func runStart(args []string) error {
	fs := flag.NewFlagSet("start", flag.ExitOnError)
	configPath := fs.String("config", "dbwatch.yaml", "path to config file")
	dsnFlag := fs.String("dsn", "", "Postgres DSN")
	intervalFlag := fs.Duration("interval", 0, "polling interval")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}

	if *dsnFlag != "" {
		cfg.Databases = append(cfg.Databases, config.Database{Name: "default", Region: "local", DSN: *dsnFlag})
	}
	if dsn := os.Getenv("DBWATCH_DSN"); dsn != "" && len(cfg.Databases) == 0 {
		cfg.Databases = append(cfg.Databases, config.Database{Name: "default", Region: "local", DSN: dsn})
	}
	if len(cfg.Databases) == 0 {
		return fmt.Errorf("no database configured (use -dsn, DBWATCH_DSN, or database(s) in %s)", *configPath)
	}

	interval := cfg.Monitor.Interval
	if *intervalFlag > 0 {
		interval = *intervalFlag
	}
	if interval <= 0 {
		interval = 10 * time.Second
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	dbs := make([]*tui.DBState, 0, len(cfg.Databases))
	for _, target := range cfg.Databases {
		db := tui.NewDBState(target)
		dbs = append(dbs, db)

		// pgxpool.New only parses the DSN — it does not connect eagerly, so
		// this failing means a malformed config, not an unreachable server.
		// Reachability (and recovery from unreachability) is handled by
		// DBState.Bootstrap, retried every tick from the TUI's poll loop —
		// a database that's down now, or down at launch, is picked up
		// automatically once it responds.
		pool, err := pgxpool.New(ctx, target.DSN)
		if err != nil {
			db.ConnectErr = fmt.Errorf("connect: %w", err)
			continue
		}

		db.Pool = pool
		db.Connections = collector.NewConnectionsCollector(pool)
		db.Cache = collector.NewCacheCollector(pool)
		db.Locks = collector.NewLocksCollector(pool)
		db.Transactions = collector.NewTransactionsCollector(pool)
		db.Queries = collector.NewQueriesCollector(pool, 5)
	}
	defer func() {
		for _, db := range dbs {
			if db.Pool != nil {
				db.Pool.Close()
			}
		}
	}()

	return tui.Run(ctx, dbs, interval)
}
