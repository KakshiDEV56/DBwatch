---
title: Connections
description: Active, idle, and idle-in-transaction sessions on this database, compared with the server's max_connections.
---

The Connections panel counts backends in `pg_stat_activity` for the current database only. dbwatch's own connection is excluded.

| State | Meaning |
| --- | --- |
| active | Running a query right now |
| idle | Connected, waiting for the next command |
| idle in transaction | Inside `BEGIN` … `COMMIT`, and not running anything. This holds locks and blocks autovacuum until the client commits or rolls back |

The total is compared with `max_connections`. That setting is server-wide: every database on the instance shares it. The counts above are only this database, so a quiet database can look fine while the instance is close to the ceiling because of other databases. Watch the utilization percentage, not just the count for the database you have selected.

## Thresholds

| Utilization | Status |
| --- | --- |
| below 70% | healthy |
| 70% up to 80% | warning |
| 80% up to 90% | degraded |
| 90% and above | critical |

Critical is reserved for this case and for a database that cannot be reached. A lock or a slow transaction does not turn the screen red.

After two polls, the panel draws a sparkline of utilization for this session, up to 30 samples.

## If the numbers look too low

A role that is not a superuser and does not have `pg_read_all_stats` only sees its own backends. Connections, locks, and transactions will all under-count. The [Capabilities](/monitoring/capabilities/) panel warns when that is the case.
