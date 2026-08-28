# dbwatch test / chaos environment

A dedicated, disposable PostgreSQL environment for verifying that dbwatch's
TUI actually detects real database conditions -- not just that some SQL ran.

This is separate from the demo environment at the repo root
(`docker-compose.yml`, `dbwatch.example.yaml`). Nothing here touches that.

## Safety

Every `dbwatch-test` command refuses to run unless **all** of:

1. `DBWATCH_TEST_ENV=true` is set in the environment.
2. The target host is in `safety.allowed_hosts` in `dbwatch-test.yaml`
   (`localhost` / `127.0.0.1` by default).
3. The database it actually connected to (`current_database()`, checked
   live, not just the DSN string) is in `safety.allowed_databases`.

Fail any check and the command exits before touching anything. There is no
way to point this harness at an arbitrary database by mistake -- both the
DSN's target and what the server reports back are checked. `database-failure`
additionally only ever stops/starts one hard-coded container name
(`dbwatch-test-pg`), never anything read from config or a flag.

## Quick start

```bash
# 1. Start the test PostgreSQL container (5 databases, schema only, on
#    localhost:5433 -- separate from the demo env's 5432)
make test-up

# 2. Seed realistic data (defaults: 20k users, 100k orders, 300k events,
#    etc. -- small enough to seed in seconds; pass -users/-orders/... to
#    dbwatch-test seed for a bigger run)
export DBWATCH_TEST_ENV=true
make seed

# 3. Watch it with the real dbwatch TUI
make watch

# 4. In another terminal, run scenarios against it
make test-locks
make test-long-tx
make test-connections
# ...or everything:
make test-all
```

## Individual scenarios

Each of these is also a `dbwatch-test` subcommand with its own flags
(`-db`, `-duration`, `-workers`, ...) if you want more control than the
`make test-*` defaults give you:

| Command | What it does |
|---|---|
| `dbwatch-test workload` | continuous mixed read/write/join/aggregation load |
| `dbwatch-test slow-query` | one `pg_sleep` query |
| `dbwatch-test long-tx` | holds a transaction **active** (a query genuinely in flight) |
| `dbwatch-test idle-tx` | holds a transaction **idle** (nothing in flight) -- distinct state |
| `dbwatch-test locks` | transaction A locks a row, B blocks on it, then releases |
| `dbwatch-test deadlock` | A and B lock opposite rows in reverse order; PostgreSQL aborts one |
| `dbwatch-test errors` | syntax/missing-table/missing-column/constraint/permission errors |
| `dbwatch-test connections` | opens many connections toward (never past) a safe % of `max_connections` |
| `dbwatch-test growth` | inserts rows at a steady rate, reports the real size delta |
| `dbwatch-test database-failure` | stops the container, confirms detection, restarts it, confirms recovery |
| `dbwatch-test scenario <name>` | run one scenario by its `make test-*` name |
| `dbwatch-test scenario all` | run every scenario in sequence with a pass/fail summary |

Every scenario prints what it's doing as it goes (not a black box), and an
"Expected in dbwatch" note telling you what to look for in the TUI while it
runs.

## Databases

| dbwatch-test name | Real database | Purpose |
|---|---|---|
| `production` | `dbwatch_test_production` | full schema, most scenarios target this by default |
| `staging` | `dbwatch_test_staging` | smaller subset, stays quiet -- a healthy comparison point |
| `analytics` | `dbwatch_test_analytics` | events/audit_logs only |
| `development` | `dbwatch_test_development` | minimal, mostly idle |
| `stress` | `dbwatch_test_stress` | isolated target for `growth`/high-volume connection tests, so `production`'s numbers stay readable |

## Cleanup

```bash
make clean-test
```

Only ever touches `test/postgres/docker-compose.yml`'s container and
volumes -- it cannot affect the demo environment or anything else on your
machine.
