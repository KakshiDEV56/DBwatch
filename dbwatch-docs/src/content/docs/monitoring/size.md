---
title: Database size
description: On-disk size of this database, split into tables and indexes, plus a growth rate measured during this session.
---

Database size is read from the catalogs. Nothing here is estimated.

| Figure | Source |
| --- | --- |
| Current size | `pg_database_size()`. Relations and catalogs. WAL is not included. |
| Tables | Sum of `pg_table_size()` over user tables: heap, TOAST, and the free space map. Indexes are not included. System schemas (`pg_catalog`, `information_schema`, `pg_toast`) are skipped, because their bytes are already inside the database size. |
| Indexes | Sum of `pg_indexes_size()` over those same user tables. |
| Growth | The change between the previous reading and this one, divided by the time between them. |

Growth is a signed rate. `VACUUM FULL`, `TRUNCATE`, and `DROP` make a database smaller, so a negative rate is kept. It is not clamped to zero.

## No capacity alarm

dbwatch cannot see the disk the data directory lives on. Inventing "80% of disk" from `pg_database_size()` would be a guess, so this panel stays informational even when the database is large. A real disk-full warning has to come from the host or the volume.

The event log records a baseline, then another line when the size later changes by at least 10 MiB or 5%, whichever comes first. Autovacuum-scale noise stays out of the log.
