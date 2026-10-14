---
title: Docker
description: Pull the published image and run the terminal UI, including how to reach PostgreSQL on the host and how to keep the saved database list.
---

The image is `ghcr.io/kakshidev56/dbwatch`. A push to `main` publishes `latest` and a short commit SHA. A `v*` tag publishes semver tags (`v0.0.2`, `0.0.2`, `0.0`). Both `linux/amd64` and `linux/arm64` are in the same manifest, so Docker pulls the architecture that matches the machine.

```bash
docker pull ghcr.io/kakshidev56/dbwatch:latest
```

:::note[If the pull is denied]
GitHub creates a new package as private. Open the package on GitHub, set its visibility to public, and link it to the `KakshiDEV56/DBwatch` repository. After that, other machines can pull without logging in.
:::

## Run it

dbwatch draws a full-screen terminal UI. The container needs an interactive TTY.

```bash
docker run --rm -it ghcr.io/kakshidev56/dbwatch
```

The image's entrypoint is `dbwatch` and the default command is `start`, so that opens the welcome screen. Pass flags after the image name:

```bash
docker run --rm -it ghcr.io/kakshidev56/dbwatch start -interval 5s
```

The process runs as user `dbwatch` (uid 65532), with `TERM=xterm-256color`.

## Reach Postgres on the host

`localhost` inside the container is not your laptop.

Docker Desktop, and Linux with the host-gateway mapping:

```bash
docker run --rm -it \
  -v dbwatch-config:/home/dbwatch/.config \
  --add-host=host.docker.internal:host-gateway \
  ghcr.io/kakshidev56/dbwatch start \
  -dsn "postgres://user:password@host.docker.internal:5432/app"
```

On Linux, host networking is the other option. Then `localhost` in the URL is the host:

```bash
docker run --rm -it --network host \
  -v dbwatch-config:/home/dbwatch/.config \
  ghcr.io/kakshidev56/dbwatch start \
  -dsn "postgres://user:password@localhost:5432/app"
```

The named volume `dbwatch-config` is the per-user config directory. Databases you add with `a` persist there. Without the volume, the list disappears when the container exits.

## This is not the demo database

The image is only the dbwatch binary. It does not contain PostgreSQL.

The `docker-compose.yml` at the root of the git repository starts a separate Postgres 16 container for the [demo environment](/guides/demo/). You run dbwatch against that server. You do not run dbwatch via that compose file.
