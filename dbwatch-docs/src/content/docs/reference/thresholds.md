---
title: Thresholds
description: Every cutoff dbwatch applies, and the ones it deliberately does not. Only the polling interval is configurable.
---

These values are compiled in. The only number you can change without editing source is the polling interval, which defaults to 10 seconds. See [Configuration](/configuration/).

## Connections

Utilization is this database's backend count divided by the server-wide `max_connections`.

| Utilization | Status |
| --- | --- |
| below 70% | healthy |
| ≥ 70% | warning |
| ≥ 80% | degraded |
| ≥ 90% | critical |

## Cache hit ratio

`blks_hit / (blks_hit + blks_read)` for this database. No reads yet reports 100%.

| Hit ratio | Status |
| --- | --- |
| ≥ 99% | healthy |
| ≥ 95% and below 99% | warning |
| below 95% | degraded |

A ratio below 85% is still degraded, not critical.

## Locks

| Condition | Status |
| --- | --- |
| no blocked session | healthy |
| one or more blocked sessions | warning |

## Long-running transactions

A transaction is listed when it has been open longer than **30 seconds**. That includes `active` and `idle in transaction`.

| Condition | Status |
| --- | --- |
| none open past 30s | healthy |
| one or more | warning |

## Database size

No warning and no critical. The log records a later sample when the size changes by at least **10 MiB** or **5%**, whichever comes first.

## Capabilities

| Condition | Status |
| --- | --- |
| superuser, or `pg_read_all_stats` | healthy |
| neither | warning |

## Top queries

No time-based alarm. The panel warns when `pg_stat_statements` cannot be created, and shows the five statements with the highest total execution time otherwise.

## Errors and events

The dot matches the newest event. It does not latch on the worst event still in the log.

## Sparklines

Connections and Cache draw a trend after **2** samples, and keep the last **30**. Samples start when dbwatch starts watching that database.
