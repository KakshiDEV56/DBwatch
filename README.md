<p align="center">
  <img src="dbwatch-logo.png" alt="dbwatch logo" width="120" />
</p>

<h1 align="center">dbwatch</h1>

<p align="center">
  A terminal-based observability tool for PostgreSQL that explains <em>why</em>
  a database is becoming unhealthy, not just that it is.
</p>

<p align="center">
  <a href="LICENSE"><img alt="License: MIT" src="https://img.shields.io/badge/License-MIT-yellow.svg"></a>
  <img alt="Go version" src="https://img.shields.io/badge/go-1.25-00ADD8?logo=go&logoColor=white">
  <img alt="Version" src="https://img.shields.io/badge/version-0.0.1-blue">
  <img alt="PostgreSQL" src="https://img.shields.io/badge/postgresql-13%2B-336791?logo=postgresql&logoColor=white">
</p>

## Why

Most teams monitor their application through Grafana, Prometheus, and logs,
while the database itself stays a black box. When something breaks, the
application dashboard shows API latency rising and 5xx errors climbing --
but the actual cause is inside PostgreSQL: connection pool exhaustion, a
long-running transaction blocking others, a query that lost its index, a
falling cache hit ratio.

dbwatch is not another metrics dashboard. It watches PostgreSQL's own
statistics views in real time, in your terminal, and surfaces the specific
condition that's degrading the database -- with enough detail (PIDs,
queries, durations, blocking relationships) to act on immediately.

<p align="center">
  <img src="dbwatch.png" alt="dbwatch running against five databases" width="100%" />
</p>

## Features

- **Multi-database monitoring** -- watches several PostgreSQL databases
  concurrently, switch between them without restarting.
- **Connections** -- active / idle / idle-in-transaction breakdown against
  `max_connections`, with configurable warning/critical thresholds.
- **Cache hit ratio** -- buffer cache health from `pg_stat_database`.
- **Top queries** -- calls, mean time, total time, and rows from
  `pg_stat_statements`, correctly scoped per database.
- **Locks & blocking** -- who is blocking whom, and for how long, via
  `pg_blocking_pids()`.
- **Long-running transactions** -- flags transactions open past a
  threshold, distinguishing `active` from `idle in transaction`.
- **Independent logs/events panel** -- its own scroll position, cursor,
  and search, so a live refresh never disturbs what you're reading.
- **Detail view + clipboard copy** -- inspect any event and copy the full
  query/context via OSC52 (works over SSH, no external clipboard tool).
- **Read/live mode** -- pause polling to read without the screen moving
  under you, resume when ready.
- **Vim-style navigation** -- `h j k l`, `/` search, `Enter` inspect,
  `q` quit, throughout.
- **Automatic recovery** -- a database that's unreachable at startup, or
  goes down mid-session, is retried every poll and picked up the moment
  it's back, with no restart required.
- **A dedicated chaos-testing harness** (`test/`) that generates real
  PostgreSQL failure conditions -- lock contention, deadlocks, long
  transactions, connection pressure, database outages -- to verify
  detection against real server behavior.

## Requirements

- Go 1.25 or later
- Docker and Docker Compose (for the bundled demo/test PostgreSQL
  environments -- not required if you already have a PostgreSQL instance
  to point dbwatch at)
- PostgreSQL 13+ as the monitored target
- The `pg_stat_statements` extension enabled on the target server for the
  Top Queries panel (dbwatch attempts to create it automatically; see
  Configuration below)

## Installation

```bash
git clone <this-repository>
cd dbwatch
go build -o dbwatch ./cmd/dbwatch
```

### Cross-platform

dbwatch runs natively on Linux, macOS, and Windows -- there's no
platform-specific code to maintain: the PostgreSQL driver (`pgx`) speaks
the wire protocol directly rather than wrapping `libpq`, so there's no
cgo anywhere in the dependency tree, and the TUI framework (Bubble Tea /
Lip Gloss) handles per-OS terminal capability detection (colors, mouse,
Windows' virtual terminal sequences) internally. Clipboard copy uses
OSC52 (a terminal escape sequence), so it works wherever the *terminal
emulator* supports it, not conditional on the OS.

Build a release binary for every platform from any single machine:

```bash
make dist
```

produces `dist/dbwatch-{linux,darwin}-{amd64,arm64}` and
`dist/dbwatch-windows-amd64.exe`. Each is a static binary -- no runtime
dependencies to install alongside it.

The `Makefile` and `scripts/*.sh` (demo/test tooling, not the dbwatch
binary itself) are bash and need `make`; on Windows, run them under
WSL2 or Git Bash.

## Quick start: demo environment

A ready-made three-database demo environment is included at the repo
root, seeded with realistic e-commerce data.

```bash
# 1. Start PostgreSQL (three databases: production, staging, analytics;
#    pg_stat_statements preloaded; auto-seeded via scripts/seed.sql)
docker compose up -d
docker compose ps          # wait for "healthy"

# 2. Optional: simulate connection activity (active / idle /
#    idle-in-transaction sessions) so there's something live to watch
./scripts/simulate-load.sh

# 3. Run dbwatch against it
go build -o dbwatch ./cmd/dbwatch
./dbwatch start -config dbwatch.example.yaml -interval 3s
```

Tear down with `docker compose down -v`.

## Configuration

dbwatch reads a YAML config file (default `dbwatch.yaml`, override with
`-config`). Either a single database:

```yaml
database:
  type: postgres
  dsn: postgres://user:password@localhost:5432/app

monitor:
  interval: 10s
```

or several, watched concurrently:

```yaml
databases:
  - name: production
    region: local
    dsn: postgres://user:password@localhost:5432/app_production
  - name: staging
    region: local
    dsn: postgres://user:password@localhost:5432/app_staging

monitor:
  interval: 10s
```

Flags: `-config <path>`, `-dsn <dsn>` (adds/overrides a single database
without editing the file), `-interval <duration>`.

```bash
./dbwatch start -dsn "postgres://user:password@localhost:5432/app" -interval 10s
```

## Keybindings

| Key | Action |
|---|---|
| `h` / `l` | move focus between panels, database table, and logs |
| `j` / `k` | navigate within the focused section |
| `Enter` | inspect a log entry / confirm a database selection |
| `Esc` | back / cancel |
| `c` | copy the selected log entry or detail view to the clipboard |
| `/` | search |
| `n` / `N` | next / previous search match |
| `m` | toggle read mode (pause updates) / live mode |
| `r` | refresh immediately |
| `?` | help |
| `q` | quit |

## Project structure

```
cmd/dbwatch/            CLI entrypoint
internal/
  collector/             PostgreSQL statistics collectors (pg_stat_* queries)
  config/                YAML configuration
  tui/                   Bubble Tea dashboard (model, view, keybindings)
test/
  postgres/              dedicated test PostgreSQL (docker-compose + schema)
  cmd/dbwatch-test/       chaos-testing CLI
internal/testkit/         scenario implementations, safety gate, workload generator
scripts/                 demo-environment seed data and load simulation
```

## Testing / chaos harness

`test/` contains a self-contained harness that generates real PostgreSQL
failure conditions -- lock contention, deadlocks, long-running and
idle-in-transaction sessions, connection pressure, query errors, database
growth, and full container outages -- so dbwatch's detection can be
verified against actual server behavior rather than assumptions.

Every operation is gated by a safety check that refuses to run unless
`DBWATCH_TEST_ENV=true` is set **and** the database it connects to is one
explicitly listed in `test/dbwatch-test.yaml` (checked live via
`current_database()`, not just the DSN string). It cannot be pointed at an
arbitrary database by accident.

```bash
# 1. Start the dedicated test PostgreSQL (5 databases, schema only, on
#    localhost:5433 -- separate from the demo environment's 5432)
make test-up

# 2. Seed realistic data (idempotent -- safe to rerun)
make seed

# 3. Watch it with the real dbwatch TUI
make watch

# 4. In another terminal, run scenarios against it
make test-locks
make test-long-tx
make test-connections
make test-failure

# ...or everything, with a pass/fail summary
make test-all
```

Full scenario list, per-database layout, and safety details are in
[`test/README.md`](test/README.md).

```bash
make clean-test   # tears down only the dedicated test container/volumes
```

## Known limitations

- No storage, growth, WAL, or replication collectors yet -- verified
  present in the underlying database (via the test harness) but not yet
  surfaced in the dashboard.
- No visibility into individual failed queries (syntax errors, constraint
  violations, permission errors) -- `pg_stat_statements` does not record
  statements that fail before planning, and dbwatch does not ingest the
  PostgreSQL server log.
- Detection is poll-interval-bound: an event resolved faster than the
  configured interval (for example, PostgreSQL's own sub-second deadlock
  resolution) may not be observed.

## Third-party libraries

Everything dbwatch's own code imports directly, verified against each
module's actual `LICENSE` file rather than assumed:

| Library | Used for | License |
|---|---|---|
| [`charmbracelet/bubbletea`](https://github.com/charmbracelet/bubbletea) | TUI event loop the whole dashboard runs on | [![MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://github.com/charmbracelet/bubbletea/blob/master/LICENSE) |
| [`charmbracelet/lipgloss`](https://github.com/charmbracelet/lipgloss) | Terminal styling -- borders, color, layout | [![MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://github.com/charmbracelet/lipgloss/blob/master/LICENSE) |
| [`jackc/pgx`](https://github.com/jackc/pgx) | PostgreSQL driver -- speaks the wire protocol directly, no `libpq`/cgo | [![MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://github.com/jackc/pgx/blob/master/LICENSE) |
| [`muesli/termenv`](https://github.com/muesli/termenv) | Terminal color-profile detection + OSC52 clipboard copy | [![MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://github.com/muesli/termenv/blob/master/LICENSE) |
| [`go-yaml/yaml`](https://github.com/go-yaml/yaml) (`gopkg.in/yaml.v3`) | Config file parsing | [![MIT/Apache-2.0](https://img.shields.io/badge/License-MIT%2FApache--2.0-blue.svg)](https://github.com/go-yaml/yaml/blob/v3/LICENSE) |

Each of those pulls in further packages of its own (ANSI parsing,
terminal detection, Unicode width tables, and so on) -- roughly 20 in
total, every one of them MIT, BSD-3-Clause, or Apache-2.0, checked the
same way. That full transitive list isn't hand-copied here, since a
manually maintained copy would drift out of date the moment a dependency
updates -- it's always current straight from the toolchain:

```bash
go list -m all   # every module in the build graph
go mod graph     # who depends on what
```

## License

Distributed under the MIT License. See [`LICENSE`](LICENSE) for the full
text.
