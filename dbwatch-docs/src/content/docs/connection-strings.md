---
title: Connection strings
description: The postgres:// URL dbwatch accepts, how the database name is chosen, and how to reach a server from Docker or over TLS.
---

dbwatch takes a standard PostgreSQL connection URI. `postgres://` and `postgresql://` both work. The driver is [pgx](https://github.com/jackc/pgx), which speaks the Postgres wire protocol directly.

```text
postgres://user:password@host:5432/database
```

| Part | Meaning |
| --- | --- |
| `user` | Role dbwatch connects as |
| `password` | That role's password. URL-encode anything that is not unreserved (`@`, `:`, `/`, `#`, `?`, and spaces). |
| `host` | Hostname or IP of the PostgreSQL server |
| `5432` | Port. Omit the colon and port to use 5432. |
| `database` | Database name. This becomes the label in the sidebar when you add the URL from the welcome screen. |

A keyword DSN (`host=localhost user=postgres`) is not what the welcome screen asks for. Paste a `postgres://` URL there.

## Examples

Local server, default port, database `app`:

```text
postgres://postgres:secret@localhost:5432/app
```

TLS required (typical for a hosted Postgres):

```text
postgres://user:secret@db.example.com:5432/app?sslmode=require
```

Password contains an `@`:

```text
postgres://user:p%40ssword@localhost:5432/app
```

`%40` is `@`. If the password is left unencoded, the host is parsed wrong and the URL is rejected.

## What dbwatch does with the URL

When you submit the welcome screen (or press `a` later):

1. The string is trimmed. An empty string is rejected with "enter a postgres:// connection string".
2. pgx parses it. A malformed URL is shown as an error and is not saved.
3. The sidebar name is the database path (`app` in the examples above). If the path is missing, the name is `db1`, `db2`, and so on.
4. The region is stored as `local`.
5. A URL that parses but whose server is down is still saved. The row shows as offline and is retried every poll until it connects.

Adding a database never writes to PostgreSQL except for the monitoring queries, and an attempt to `CREATE EXTENSION pg_stat_statements` if that extension is not already installed. Removing a database only stops watching it.

## From a container

`localhost` inside a container is the container, not your laptop. To watch Postgres running on the host:

```bash
docker run --rm -it \
  --add-host=host.docker.internal:host-gateway \
  ghcr.io/kakshidev56/dbwatch start \
  -dsn "postgres://user:password@host.docker.internal:5432/app"
```

On Linux, `--network host` can be used instead, and then `localhost` in the URL is the host's localhost. More of that is on the [Docker](/docker/) page.

:::caution
A connection string with a password will be stored in your per-user config file (`~/.config/dbwatch/databases.yaml` on Linux) if you add it from the welcome screen. `-dsn` and `DBWATCH_DSN` are not written to that file. See [Configuration](/configuration/).
:::

## Useful query parameters

These are pgx / libpq parameters, not dbwatch settings. Common ones:

| Parameter | Use |
| --- | --- |
| `sslmode` | `disable` on a trusted local socket path, `require` or `verify-full` for a remote server |
| `sslrootcert` | CA file when the server certificate is not in the default trust store |
| `connect_timeout` | Seconds to wait for the TCP handshake |
| `application_name` | Shows up in `pg_stat_activity` so you can see dbwatch's own sessions |

dbwatch's own polling interval is not a URL parameter. Set it with [`-interval`](/configuration/#flags) or the YAML `monitor.interval` field.
