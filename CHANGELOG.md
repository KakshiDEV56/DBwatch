# Changelog

All notable changes to dbwatch are documented here. Format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this
project uses [semantic versioning](https://semver.org/) once it reaches
1.0.0 -- before that, minor versions (0.x) may include breaking changes.

## [0.0.2] - 2026-08-29

### Added

- Zero-config onboarding: a welcome screen on first run (or after
  removing the last configured database) with a connection-string input
  right there -- no YAML file required before dbwatch will even start.
- Databases are now added (`a`) and removed (`d`, with a yes/no
  confirmation) at runtime, and persisted to a per-user config
  directory (`~/.config/dbwatch/databases.yaml` on Linux, the platform
  equivalent elsewhere) that loads automatically on every launch.
- Mouse support: wheel-scroll and click-to-select across the sidebar
  panels, the database table, and the logs panel, sharing the same
  move logic as keyboard navigation.
- Centered search overlay (`/`) with live match counts across database
  names and the current panel's logs.
- Full scrollable in-app guide (`?`) documenting every panel, metric,
  and threshold dbwatch shows -- not just a keybinding list.
- Real per-second load metrics in the Health card -- transactions/sec,
  tuples/sec, temp-file spill rate, cumulative deadlocks -- via a new
  activity collector, diffed between consecutive polls rather than
  estimated.
- Denser metric cards: Connections and Cache show their real
  active/idle/idle-in-tx and hit/miss breakdowns; Locks & Blocking and
  Long-Running Transactions show actual PID pairs / rows, not just a
  count.
- Bounded rolling history (30 samples) powering real sparklines on
  Connections and Cache -- rendered only once genuine samples exist.
- `make dist` cross-compiles static release binaries for
  linux/darwin/windows (amd64 + arm64) from a single machine, with no
  cgo dependency anywhere in the tree.
- Third-party library attribution in the README, verified against each
  dependency's actual `LICENSE` file rather than assumed.
- This changelog.

### Changed

- Full Gruvbox-inspired visual redesign: a centralized `Theme` type
  everything derives from, neutral borders by default with color living
  in status symbols and values instead of bright per-panel outlines,
  and corrected severity semantics (lock contention and moderate cache
  degradation are now warning/degraded, not critical -- critical is
  reserved for genuinely severe states like near connection exhaustion
  or an unreachable database).
- Header collapsed from two lines to one dense info line (version,
  database count, aggregate warning/critical count, live mode, clock).
- Module path renamed to `github.com/KakshiDEV56/DBwatch` so
  `go install`/`go get` resolve correctly.
- Installation docs restructured around three paths: download a
  prebuilt binary, `go install`, or build from source.

### Fixed

- A database unreachable when dbwatch starts is now retried
  automatically instead of being permanently skipped for the rest of
  the session.
- Several width/truncation bugs where box titles or card content could
  silently wrap past their allocated box height on narrower terminals,
  breaking alignment with everything below them.
- A startup-tick race where the Top Queries panel's first poll could
  run concurrently with `pg_stat_statements` being enabled, producing a
  spurious one-time "relation does not exist" error.

## [0.0.1] - 2026-08-28

### Added

- Initial release: a PostgreSQL observability TUI built with Bubble Tea.
- Multi-database monitoring dashboard: Connections, Cache Hit Ratio,
  Top Queries (`pg_stat_statements`), Locks & Blocking
  (`pg_blocking_pids()`), and Long-Running Transactions, each scoped
  correctly per database.
- Independent, scrollable logs/events panel with its own cursor,
  search, and clipboard copy via OSC52 (no external clipboard tool,
  works over SSH).
- Vim-style navigation (`h j k l`) and read/live mode throughout.
- A seeded three-database demo environment (`docker-compose.yml`,
  `scripts/`) for a quick first look.
- A dedicated chaos-testing harness (`test/`) that generates real
  PostgreSQL failure conditions -- lock contention, deadlocks, long and
  idle-in-transaction sessions, connection pressure, query errors,
  database growth, full outages -- gated by an explicit safety check,
  to verify detection against real server behavior rather than
  assumptions.

[0.0.2]: https://github.com/KakshiDEV56/DBwatch/compare/v0.0.1...v0.0.2
[0.0.1]: https://github.com/KakshiDEV56/DBwatch/releases/tag/v0.0.1
