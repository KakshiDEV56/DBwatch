# syntax=docker/dockerfile:1
#
# dbwatch is a terminal UI, so the container needs an interactive TTY.
# After this image is published, anyone can run it with:
#
#   docker run --rm -it ghcr.io/kakshidev56/dbwatch
#
# Reach PostgreSQL on the laptop (Docker Desktop, or Linux with the
# host-gateway mapping):
#
#   docker run --rm -it \
#     -v dbwatch-config:/home/dbwatch/.config \
#     --add-host=host.docker.internal:host-gateway \
#     ghcr.io/kakshidev56/dbwatch start \
#     -dsn "postgres://user:password@host.docker.internal:5432/app"
#
# On Linux, --network host reaches a server bound to the host's localhost.
# Saved databases persist in the mounted volume at
# /home/dbwatch/.config/dbwatch/databases.yaml.

FROM golang:1.25.9-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/dbwatch \
    ./cmd/dbwatch

FROM alpine:3.22

RUN apk add --no-cache ca-certificates \
    && adduser -D -u 65532 -h /home/dbwatch dbwatch \
    && mkdir -p /home/dbwatch/.config/dbwatch \
    && chown -R dbwatch:dbwatch /home/dbwatch

COPY --from=build /out/dbwatch /usr/local/bin/dbwatch

USER dbwatch
WORKDIR /home/dbwatch

ENV HOME=/home/dbwatch \
    TERM=xterm-256color \
    COLORTERM=truecolor

ENTRYPOINT ["dbwatch"]
CMD ["start"]
