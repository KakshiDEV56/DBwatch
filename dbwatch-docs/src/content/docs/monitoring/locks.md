---
title: Locks and blocking
description: Which session is waiting on a lock, which session holds it, and why a conflict that ends quickly can be missed.
---

The locks panel lists sessions in this database that are waiting on another session, using `pg_blocking_pids()`. Each row is one blocked backend and the backend blocking it: both PIDs, both queries, the application name, and how long each has been in its current query.

An empty panel means nothing is waiting right now.

Any blocked session sets the panel to warning. One query waiting on another is common and often clears on its own, so dbwatch does not call it critical. Critical stays reserved for an unreachable database and for connection usage at or above 90%.

## What a poll can miss

dbwatch only sees locks that are still held when it polls. PostgreSQL's deadlock detector usually aborts one side in under a second. If your interval is 10 seconds, a deadlock that the server already resolved will not show up as a blocked pair.

Deadlocks that did happen are still counted. `pg_stat_database.deadlocks` is a counter, and an increase is written to [Errors and events](/monitoring/errors/) even when the lock itself was gone before the next poll.

A role without `pg_read_all_stats` may not see other sessions' locks. [Capabilities](/monitoring/capabilities/) warns when that is true.
