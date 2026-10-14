---
title: Long-running transactions
description: Transactions open longer than 30 seconds, split into active queries and idle-in-transaction sessions.
---

A transaction is listed when `now() - xact_start` is greater than 30 seconds. That cutoff is fixed in this version. It is not a YAML setting and not a flag. dbwatch's own connection is excluded, and only this database is considered.

Each row has the PID, application name, state, age, and the last query.

| State | What it usually means |
| --- | --- |
| active | A query is actually running. It may be legitimate work, or it may be stuck. |
| idle in transaction | Nothing is running. The client opened a transaction and did not commit or roll back. Vacuum cannot clean dead rows for as long as it stays open, and it can hold locks the whole time. |

Any transaction past 30 seconds sets the panel to warning. The panel is healthy when the list is empty.

Open a log row with Enter to read the full query, and `c` to copy it. The copy uses OSC 52, so it works over SSH without an external clipboard tool.
