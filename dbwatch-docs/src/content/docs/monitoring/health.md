---
title: Health and load
description: The summary card at the top of the sidebar. Version, uptime, and per-second rates computed from pg_stat_database counters.
---

The health card is the top of the sidebar: status, server version, database name, uptime, and a load line. Uptime comes from `pg_postmaster_start_time()`. Version comes from `server_version`.

The load numbers are rates. dbwatch subtracts the previous `pg_stat_database` snapshot from the current one and divides by the elapsed time. They are not estimates, and they are not CPU.

| Figure | Meaning |
| --- | --- |
| xact/s | Committed and rolled-back transactions per second |
| tuples/s | Rows returned, fetched, inserted, updated, and deleted per second |
| temp spill | Shown only when queries are writing temp files. Usually `work_mem` is too small for the sort or hash that is running. |
| deadlocks | Cumulative count since statistics were last reset, not a per-second rate |

The first poll has no previous snapshot, so the rates appear from the second poll onward.

## What is deliberately missing

CPU, memory, and disk utilization are not in PostgreSQL's statistics views. Reading them means an agent on the host (`/proc` or the platform's metrics API). dbwatch does not invent those percentages from SQL.

A rising temp-spill rate is the closest SQL-visible sign that work is falling out of memory onto disk. Pair it with [Top queries](/monitoring/queries/) to see which statements are responsible.
