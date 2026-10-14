---
title: Changelog
description: Notable changes to dbwatch. The project follows Keep a Changelog. Before 1.0.0, minor versions may include breaking changes.
---

Versions before 1.0.0 use minor bumps (`0.x`) for changes that may break a previous release. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## 0.0.2

29 August 2026. [Compare with 0.0.1](https://github.com/KakshiDEV56/DBwatch/compare/v0.0.1...v0.0.2).

### Added

- A welcome screen on first run, and again after the last database is removed. Paste a connection string there. No YAML file is required to start.
- Databases are added with `a` and removed with `d` after a yes/no confirmation. The list is stored in the per-user config directory and loaded on every launch.
- Mouse support. The wheel and a click use the same movement as the keyboard, on the sidebar, the database table, and the logs.
- Search on `/`, with a match count across database names and the current panel's log.
- An in-app guide on `?` that documents every panel, not only the keys.
- Real per-second load on the health card: transactions, tuples, temp-file spill, and the deadlock counter, differenced between polls.
- Denser cards. Connections and Cache show the active / idle / idle-in-transaction and hit / miss split. Locks and long transactions show the rows, not only a count.
- Sparklines on Connections and Cache, from the last 30 samples, drawn only after real samples exist.
- `make dist` cross-compiles static binaries for Linux, macOS, and Windows (amd64 and arm64, except Windows arm64) with cgo disabled.
- Third-party licenses checked against each module's `LICENSE` file.

### Changed

- Gruvbox-inspired theme. Color lives on the status symbol and the value. Borders stay neutral.
- Lock contention and a moderate cache drop are warning or degraded. Critical is near connection exhaustion, or a database that cannot be reached.
- The header is one line: version, database count, warning and critical count, live mode, clock.
- The module path is `github.com/KakshiDEV56/DBwatch`, so `go install` resolves.

### Fixed

- A database that is down at startup is retried, instead of being skipped for the rest of the session.
- Box titles and card text no longer wrap past their height on a narrow terminal.
- A startup race where Top queries could poll before `pg_stat_statements` existed, and report a one-time "relation does not exist".

## 0.0.1

28 August 2026. [Release](https://github.com/KakshiDEV56/DBwatch/releases/tag/v0.0.1).

### Added

- First release: a PostgreSQL observability UI built with Bubble Tea.
- Connections, cache hit ratio, top queries, locks, and long-running transactions, each scoped to the selected database.
- A log with its own cursor, search, and OSC 52 copy.
- Vim keys, and read mode versus live mode.
- A seeded demo Postgres under `docker-compose.yml`.
- A chaos harness under `test/` that creates lock contention, deadlocks, long and idle transactions, connection pressure, query errors, growth, and outages. It refuses to run unless `DBWATCH_TEST_ENV=true` is set and the database is one listed in `test/dbwatch-test.yaml`.
