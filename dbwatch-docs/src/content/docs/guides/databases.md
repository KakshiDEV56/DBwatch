---
title: Databases
description: Add and remove databases from inside the app. The list is saved in your user config directory and loaded on the next launch.
---

Every database in the table is polled on the same interval, including the ones whose panels are not on screen. Selecting a row only changes which panels and logs you are looking at.

## Add

Press `a`. The prompt is the same one as the welcome screen. Paste a `postgres://` URL and press Enter.

- An empty or malformed URL stays on the prompt with an error.
- A valid URL is parsed, named after the database in the path, and saved.
- If the server is down, the database is still added and shown as offline until a later poll succeeds.

`Esc` closes the prompt when at least one database is already configured. On the welcome screen, with nothing to go back to, `Esc` quits.

## Remove

Select the row and press `d`. dbwatch asks for confirmation. `y` or Enter removes it. `n` or `Esc` cancels.

Removal drops it from the saved list and closes dbwatch's pool. It does not drop the PostgreSQL database, and it does not revoke anything.

If you remove the last database, the welcome screen comes back.

## Which role to use

Grant the monitoring role `pg_read_all_stats` so Connections, Locks, and Long-running transactions can see every session:

```sql
GRANT pg_read_all_stats TO dbwatch;
```

Superuser also works, and is more privilege than monitoring needs. Create `pg_stat_statements` once as a role that is allowed to, if the monitoring role cannot.

The saved file contains the connection strings, including passwords, for databases you add in the app. Flags do not write into that file. See [Configuration](/configuration/).
