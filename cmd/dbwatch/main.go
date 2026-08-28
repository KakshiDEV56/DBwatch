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

	"github.com/KakshiDEV56/DBwatch/internal/config"
	"github.com/KakshiDEV56/DBwatch/internal/store"
	"github.com/KakshiDEV56/DBwatch/internal/tui"
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

With no arguments, dbwatch loads whatever databases you've previously
added (press 'a' inside the app to add one -- no config file needed) from
its own per-user config directory. Run with zero databases configured and
it shows a welcome screen to add your first one.

Flags:
  -config string    path to an additional config file (optional)
  -dsn string        Postgres DSN, adds a database for just this run
  -interval duration polling interval (default 10s)

-config/-dsn databases are layered on top of your saved list for this
run; they are not saved unless added again from within the app.`)
}

func runStart(args []string) error {
	fs := flag.NewFlagSet("start", flag.ExitOnError)
	configPath := fs.String("config", "", "path to an additional config file")
	dsnFlag := fs.String("dsn", "", "Postgres DSN for this run")
	intervalFlag := fs.Duration("interval", 0, "polling interval")
	if err := fs.Parse(args); err != nil {
		return err
	}

	saved, err := store.Load()
	if err != nil {
		return fmt.Errorf("load saved databases: %w", err)
	}
	targets := append([]config.Database{}, saved...)

	interval := 10 * time.Second
	if *configPath != "" {
		cfg, err := config.Load(*configPath)
		if err != nil {
			return err
		}
		targets = append(targets, cfg.Databases...)
		if cfg.Monitor.Interval > 0 {
			interval = cfg.Monitor.Interval
		}
	}
	if *dsnFlag != "" {
		targets = append(targets, config.Database{Name: "default", Region: "local", DSN: *dsnFlag})
	}
	if dsn := os.Getenv("DBWATCH_DSN"); dsn != "" && len(targets) == 0 {
		targets = append(targets, config.Database{Name: "default", Region: "local", DSN: dsn})
	}
	if *intervalFlag > 0 {
		interval = *intervalFlag
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	dbs := make([]*tui.DBState, 0, len(targets))
	for _, target := range targets {
		dbs = append(dbs, tui.ConnectDatabase(ctx, target))
	}
	defer func() {
		for _, db := range dbs {
			if db.Pool != nil {
				db.Pool.Close()
			}
		}
	}()

	// Zero databases is not an error -- dbwatch shows its welcome screen
	// and lets the user add one right there, rather than requiring a
	// config file to exist before it will even start.
	return tui.Run(ctx, dbs, interval)
}
