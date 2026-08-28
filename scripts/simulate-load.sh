#!/usr/bin/env bash
# Opens a handful of connections against dbwatch-pg in various states
# (active, idle, idle-in-transaction) so dbwatch has something to show.
# Each connection lives inside the container via `docker exec -d`, so it
# survives independently of this script.
set -euo pipefail

CONTAINER="${CONTAINER:-dbwatch-pg}"
DB="${DB:-dbwatch_demo}"
DURATION="${DURATION:-1800}" # seconds each simulated connection stays open

psql_bg() {
  docker exec -d "$CONTAINER" sh -c "$1"
}

echo "Opening connections against $CONTAINER/$DB for ${DURATION}s..."

# Active: long-running query.
for i in 1 2; do
  psql_bg "echo 'select pg_sleep($DURATION);' | psql -U postgres -d $DB"
done

# Idle: ran a query, then sits waiting on stdin (connection stays open, no
# query in flight).
for i in 1 2 3; do
  psql_bg "(echo 'select 1;'; sleep $DURATION) | psql -U postgres -d $DB"
done

# Idle in transaction: opened a transaction, ran one statement, never
# committed.
for i in 1 2; do
  psql_bg "(echo 'begin; select count(*) from orders;'; sleep $DURATION) | psql -U postgres -d $DB"
done

echo "Done. ~7 simulated connections will stay open for ${DURATION}s."
