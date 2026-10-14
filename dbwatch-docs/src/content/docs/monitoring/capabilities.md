---
title: Capabilities
description: What this server exposes, and what the connected role is allowed to see. The other panels are only as complete as this list.
---

The capabilities panel is the reason a number elsewhere might be incomplete. It reads the connected role and a handful of server settings. It does not judge the health of the data.

| Field | Why it matters |
| --- | --- |
| Superuser | Sees every backend in `pg_stat_activity`. |
| `pg_read_all_stats` | Same visibility, without superuser. This is the role to grant for a monitoring login. |
| `track_activities` | When off, `pg_stat_activity` does not record current queries. Empty is not the same as idle. |
| `track_counts` | When off, the statistics collector is not maintaining the counters the other panels read. |
| `track_io_timing` | When off, block read and write timings are not collected. |
| `pg_stat_statements` | Available means the extension can be installed. It still has to be listed in `shared_preload_libraries` and the server restarted before [Top queries](/monitoring/queries/) has data. |
| Extensions | The names currently installed in this database. |

## The one warning this panel raises

If the role is not a superuser and does not have `pg_read_all_stats`, the panel is a warning. That is a real limit: Connections, Locks, and Long-running transactions can only see this role's own sessions, so they will under-report.

```sql
GRANT pg_read_all_stats TO dbwatch;
```

`pg_read_all_stats` exists since PostgreSQL 10. Creating `pg_stat_statements` is a separate privilege. A monitoring role often cannot `CREATE EXTENSION`. Create the extension as a superuser once, and let the monitoring role only read it.

A baseline is logged once per connection. The log updates when something changes (an extension installed, a role grant, a reload that flips a `track_*` setting), not on every poll.
