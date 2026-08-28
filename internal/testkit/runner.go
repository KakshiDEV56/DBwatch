package testkit

import (
	"context"
	"fmt"
	"time"
)

// RunResult is one scenario's outcome, used to print the final summary.
type RunResult struct {
	Name     string
	Duration time.Duration
	Err      error
}

// scenario pairs a CLI name (kebab-case, e.g. `dbwatch-test scenario
// long-transaction`) and a display name with a runnable function using
// sensible defaults -- this is what `scenario all` iterates. Each also
// has a dedicated CLI subcommand (see test/cmd/dbwatch-test) exposing the
// same scenario with configurable parameters.
type scenarioDef struct {
	cliName string
	display string
	run     func(ctx context.Context, cfg Config) error
}

func scenarioDefs() []scenarioDef {
	return []scenarioDef{
		{"connections", "connection monitoring", func(ctx context.Context, cfg Config) error {
			return ConnectionPressure(ctx, cfg, "production", 30, 10*time.Second)
		}},
		{"workload", "query workload", func(ctx context.Context, cfg Config) error {
			_, err := Workload(ctx, cfg, "production", 5, 20, 15*time.Second)
			return err
		}},
		{"slow-query", "slow query", func(ctx context.Context, cfg Config) error {
			return SlowQuery(ctx, cfg, "production", 8*time.Second)
		}},
		{"long-transaction", "long transaction", func(ctx context.Context, cfg Config) error {
			return LongTransaction(ctx, cfg, "production", cfg.Thresholds.LongTransaction+15*time.Second)
		}},
		{"idle-transaction", "idle transaction", func(ctx context.Context, cfg Config) error {
			return IdleInTransaction(ctx, cfg, "production", cfg.Thresholds.LongTransaction+15*time.Second)
		}},
		{"locks", "lock contention", func(ctx context.Context, cfg Config) error {
			return LockContention(ctx, cfg, "production", 15*time.Second)
		}},
		{"deadlock", "deadlock", func(ctx context.Context, cfg Config) error {
			return Deadlock(ctx, cfg, "production")
		}},
		{"query-errors", "query errors", func(ctx context.Context, cfg Config) error {
			return QueryErrors(ctx, cfg, "production")
		}},
		{"connection-pressure", "connection pressure", func(ctx context.Context, cfg Config) error {
			return ConnectionPressure(ctx, cfg, "production", 80, 10*time.Second)
		}},
		{"growth", "database growth", func(ctx context.Context, cfg Config) error {
			return Growth(ctx, cfg, "stress", 500, 10*time.Second)
		}},
		{"database-failure", "database failure", func(ctx context.Context, cfg Config) error {
			return DatabaseFailure(ctx, cfg, "production", 10*time.Second)
		}},
	}
}

// RunAll runs every scenario sequentially (never in parallel -- several
// scenarios stop the shared container or push connection limits, so
// concurrent scenarios would corrupt each other's results) and prints a
// running scoreboard as it goes.
func RunAll(ctx context.Context, cfg Config) []RunResult {
	defs := scenarioDefs()
	fmt.Printf("\nDBWATCH TEST SUITE\n\n")

	var results []RunResult
	for _, d := range defs {
		start := time.Now()
		err := d.run(ctx, cfg)
		results = append(results, RunResult{Name: d.display, Duration: time.Since(start), Err: err})
	}

	fmt.Printf("\n")
	passed := 0
	for _, r := range results {
		if r.Err == nil {
			fmt.Printf("✓ %s\n", r.Name)
			passed++
		} else {
			fmt.Printf("✕ %s (%v)\n", r.Name, r.Err)
		}
	}
	fmt.Printf("\n%d/%d scenarios completed\n", passed, len(results))
	return results
}

// ScenarioNames lists every name accepted by `dbwatch-test scenario <name>`.
func ScenarioNames() []string {
	names := make([]string, 0, len(scenarioDefs()))
	for _, d := range scenarioDefs() {
		names = append(names, d.cliName)
	}
	return names
}

// RunNamed runs exactly one scenario by name, as used by
// `dbwatch-test scenario <name>`.
func RunNamed(ctx context.Context, cfg Config, name string) error {
	for _, d := range scenarioDefs() {
		if d.cliName == name {
			return d.run(ctx, cfg)
		}
	}
	return fmt.Errorf("unknown scenario %q (see: dbwatch-test scenario --list)", name)
}
