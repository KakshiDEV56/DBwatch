---
title: Configuration
description: The saved database list, the optional YAML file, and the flags and environment variable that add databases for a single run.
---

Most people never write a config file. Databases added with the welcome screen or the `a` key are stored for you and loaded on every launch.

Flags and `DBWATCH_DSN` exist for scripts, CI, and containers, where an interactive welcome screen is the wrong interface.

## Saved databases

dbwatch uses the OS user-config directory:

| OS | File |
| --- | --- |
| Linux | `~/.config/dbwatch/databases.yaml` |
| macOS | `~/Library/Application Support/dbwatch/databases.yaml` |
| Windows | `%AppData%\dbwatch\databases.yaml` |

The file is a YAML list of databases. dbwatch writes it atomically (a temp file, then a rename). Deleting the file is the same as starting over: the next launch shows the welcome screen.

Inside a container the same path is `/home/dbwatch/.config/dbwatch/databases.yaml`. Mount a volume there if the list should survive `docker run`. See [Docker](/docker/).

## Flags

```bash
dbwatch start [flags]
```

| Flag | Default | Effect |
| --- | --- | --- |
| `-config` | none | Load an extra YAML file for this run. Those databases are not written to the saved list. |
| `-dsn` | none | Add one database for this run. Not written to the saved list. |
| `-interval` | `10s` | How often every database is polled. A value greater than zero replaces the interval from `-config`. |

The saved list is always loaded first. `-config` databases are appended. `-dsn` is appended after that. Nothing from those two flags is persisted unless you add the same URL again from inside the app.

```bash
dbwatch start -dsn "postgres://user:password@localhost:5432/app" -interval 10s
```

:::note
`DBWATCH_DSN` is only used when the run would otherwise have zero databases: no saved list, no `-config` entries, and no `-dsn`. If you already have a saved database, the environment variable is ignored. `-dsn` does not have that restriction.
:::

## YAML file

`-config` accepts either one database or several. A missing region becomes `local`. A legacy single-database file with no name becomes `default`.

One database:

```yaml
database:
  type: postgres
  dsn: postgres://user:password@localhost:5432/app

monitor:
  interval: 10s
```

Several:

```yaml
databases:
  - name: production
    region: local
    dsn: postgres://user:password@localhost:5432/app_production
  - name: staging
    region: local
    dsn: postgres://user:password@localhost:5432/app_staging

monitor:
  interval: 10s
```

`interval` is a Go duration: `10s`, `500ms`, `1m`. The default, when the field is omitted or the file is missing, is 10 seconds. `type` is accepted and stored. PostgreSQL is the only database dbwatch monitors.

The polling interval is the only monitor setting. The 30 second long-transaction cutoff, the connection percentages, and the cache percentages are fixed. They are listed in [Thresholds](/reference/thresholds/).

## What is not configurable

- Panel thresholds. Changing them would mean editing the source.
- Which panels appear. All eight are always shown.
- A historical retention window beyond the current session. Sparklines keep the last 30 samples collected since dbwatch started watching that database.
