---
title: Demo environment
description: A Postgres 16 container with seeded data, so you can click around dbwatch before pointing it at a real server.
---

The repository includes a demo Postgres, separate from the dbwatch container image. The image on GitHub Container Registry is only the terminal UI. This compose file is only the database.

From a clone of the repository:

```bash
docker compose up -d
docker compose ps
```

Wait until the container is healthy. It listens on `localhost:5432`, database `dbwatch_demo`, user `postgres`, password `dbwatch`. `pg_stat_statements` is preloaded.

```text
postgres://postgres:dbwatch@localhost:5432/dbwatch_demo
```

Paste that into `dbwatch start`, or pass it for one run:

```bash
dbwatch start -dsn "postgres://postgres:dbwatch@localhost:5432/dbwatch_demo" -interval 5s
```

`scripts/simulate-load.sh` opens a mix of active, idle, and idle-in-transaction sessions against that server so the Connections panel has something to show. It is optional.

Tear the demo down, including its volume:

```bash
docker compose down -v
```

## The test harness is a different server

`make test-up` starts another Postgres, on port **5433**, used by the chaos scenarios (locks, deadlocks, long transactions, connection pressure, outages). It is not the demo above, and `make clean-test` only removes that test container. The scenario list and the safety check that refuses to run against an unlisted database are in `test/README.md` in the repository.
