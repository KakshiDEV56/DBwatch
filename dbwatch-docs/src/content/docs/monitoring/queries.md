---
title: Top queries
description: The five statements with the highest total execution time in this database, from pg_stat_statements.
---

Top queries reads `pg_stat_statements`, filtered to the current database. The extension is shared by every database on the server, so dbwatch joins it to `pg_database` and drops statements that ran elsewhere.

The panel shows five statements, ordered by total execution time:

| Column | Meaning |
| --- | --- |
| calls | How many times this statement shape has run |
| mean | Average time per call |
| total | Cumulative time across every call |
| rows | Rows returned or affected, summed across calls |

`pg_stat_statements` records mean, min, max, and standard deviation. It does not record a true P95 or P99. dbwatch shows the mean it can defend, and does not invent a percentile.

The panel's own status stays informational when data is flowing. It becomes a warning when the extension could not be enabled, and offline when the query itself fails.

## Enable pg_stat_statements

dbwatch runs `CREATE EXTENSION IF NOT EXISTS pg_stat_statements` when it connects. That statement only succeeds if the module is already loaded.

On the server:

```text
shared_preload_libraries = 'pg_stat_statements'
pg_stat_statements.track = all
```

`shared_preload_libraries` is read at server start. Changing it requires a restart. After the restart, either let dbwatch create the extension, or create it yourself:

```sql
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;
```

The role dbwatch uses needs permission to create the extension, or the extension must already exist. If creation fails, the panel shows the error, including the reminder about `shared_preload_libraries`. The rest of the dashboard keeps working.

Statements that fail before planning (syntax errors, some permission errors) are not stored in `pg_stat_statements`. dbwatch does not read the PostgreSQL server log, so those failures do not appear here. See [Limitations](/reference/limitations/).
