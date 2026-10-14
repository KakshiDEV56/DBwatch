---
title: Cache hit ratio
description: The share of reads PostgreSQL served from shared buffers instead of disk, from pg_stat_database.
---

Cache hit ratio is `blks_hit / (blks_hit + blks_read)` for this database, from `pg_stat_database`. Those counters are cumulative since stats were last reset, not a window of the last few seconds.

A ratio near 100% means the working set fits in shared buffers. Reads that miss go to disk (or to the operating system's cache, which Postgres still counts as a miss).

If the database has had no reads yet, dbwatch reports 100%. There is nothing to have missed. A new database that has had a little traffic and a cold cache can sit well below the thresholds without that meaning the working set is too large. Give it real traffic before treating a low ratio as a sizing problem.

## Thresholds

| Hit ratio | Status |
| --- | --- |
| 99% and above | healthy |
| 95% up to 99% | warning |
| below 95% | degraded |

A low hit ratio stays degraded. It does not become critical. Red is reserved for a database that is unreachable or about to exhaust `max_connections`.

The panel draws a sparkline once two samples exist in this session.

## What dbwatch will not infer

A low ratio does not, by itself, name the missing index or the query that caused it. Use [Top queries](/monitoring/queries/) for the statements taking the most time, and the host's own monitoring for whether the disk is actually saturated. PostgreSQL's statistics views do not expose CPU or disk hardware.
