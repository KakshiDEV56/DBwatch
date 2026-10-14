---
title: Dashboard
description: The header, the panel sidebar, the database table, and the log. Every database is polled on the same interval, including the ones you are not looking at.
---

dbwatch is one screen. The header is a single line: version, how many databases are configured, how many are warning or critical, whether you are in live or read mode, and the clock.

Under that, three regions:

- **Panels**, on the left. Eight sections for the database that is selected. Move with `j` and `k`, or click a card.
- **Database table**. Every database you have added. The status next to a name is that database's live state, even while another database's panels are on screen. Enter or a click selects it.
- **Logs**, along the bottom. The history for the panel that is focused. It has its own cursor. A refresh does not pull you back to the latest row unless you are already following the end.

`h` and `l` (or Tab and Shift+Tab) move focus between panels, the database table, and the logs. The full key list is on [Keyboard and mouse](/guides/keyboard/).

## Status

Every panel, and every database row, uses the same six levels. Nothing in the UI invents its own color.

| Symbol | Label | Meaning |
| --- | --- | --- |
| `●` | info | No judgement yet, or the panel is informational |
| `✓` | healthy | Inside the normal range |
| `⚠` | warning | Worth looking at |
| `▲` | degraded | Clearly worse, not an outage |
| `✕` | critical | Near a hard limit. Today this is connection usage at or above 90% |
| `○` | offline | dbwatch cannot query that database |

The dot on a database row is the worst status across its panels. Offline wins over everything else, because the server is not answering.

## Polling

Live mode is the default. Every database is queried on the interval (10 seconds unless you changed it). `r` polls immediately. `m` switches to read mode, which stops polling so the numbers stay put while you read them. `m` again resumes.

A database that is down at startup is not dropped. The same retry runs every tick, and the row comes back to life when the server does.

## What each panel is

| Panel | Source | Alarm |
| --- | --- | --- |
| [Connections](/monitoring/connections/) | `pg_stat_activity`, `max_connections` | 70% / 80% / 90% |
| [Cache hit ratio](/monitoring/cache/) | `pg_stat_database` | below 99% / below 95% |
| [Top queries](/monitoring/queries/) | `pg_stat_statements` | warning only if the extension cannot be enabled |
| [Locks and blocking](/monitoring/locks/) | `pg_blocking_pids()` | warning while any session is blocked |
| [Long-running transactions](/monitoring/transactions/) | `pg_stat_activity` | warning while any transaction is older than 30s |
| [Database size](/monitoring/size/) | `pg_database_size()` | none. Size is reported, not judged |
| [Capabilities](/monitoring/capabilities/) | roles and settings | warning if the role cannot see other sessions |
| [Errors and events](/monitoring/errors/) | other panels' warnings, plus deadlocks | the latest event |

Sparklines on Connections and Cache appear after two samples in this session. They are not backfilled from before dbwatch started.
