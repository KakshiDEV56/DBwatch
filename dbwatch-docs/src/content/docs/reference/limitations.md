---
title: Limitations
description: What dbwatch cannot see, because PostgreSQL does not expose it or because dbwatch only polls.
---

## Polling

dbwatch samples on an interval. A lock, a transaction, or a connection spike that starts and ends between two polls is invisible as a live row. Deadlocks are the exception: the server counts them, and an increase is logged even if the deadlock itself lasted milliseconds.

The default interval is 10 seconds. A shorter `-interval` sees more, and costs more queries.

## Failed statements

`pg_stat_statements` does not store statements that fail before planning. Syntax errors, many permission errors, and some constraint failures never become a top-query row. dbwatch does not tail the PostgreSQL log, so those errors are not in the event panel either.

## Host metrics

CPU, memory, and disk fullness are not available through SQL. The health card's rates (transactions per second, tuples per second, temp-file spill, deadlocks) are the load signal dbwatch can actually measure. Temp spill is a hint that `work_mem` is too small. It is not a memory percentage.

Database size has the same gap. `pg_database_size()` is the database, not the disk it sits on, so there is no "disk almost full" status.

## Visibility of other sessions

Without superuser or `pg_read_all_stats`, `pg_stat_activity` only returns the connected role's own backends. Connections, locks, and long transactions then describe dbwatch itself more than the application. The Capabilities panel warns when this is the case.

## Not built yet

There is no replication lag panel, no WAL panel, and no per-table growth panel beyond the database-level size. The test harness can create those conditions. The dashboard does not show them yet.

## Clipboard and terminals

Copy uses OSC 52. Terminals and multiplexers that disable OSC 52 will ignore it. There is no fallback to `pbcopy` or `xclip`.
