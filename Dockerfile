# syntax=docker/dockerfile:1
# bun is the package manager; vue-tsc needs a real node binary to run.
FROM oven/bun:1 AS web-base
COPY --from=node:22-slim /usr/local/bin/node /usr/local/bin/node
WORKDIR /web

FROM web-base AS web
COPY web/package.json web/bun.lock ./
RUN --mount=type=cache,target=/root/.bun/install/cache bun install --frozen-lockfile
COPY web/ ./
RUN bun run build

FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
COPY --from=web /web/dist ./web/dist
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/app ./cmd/app

FROM alpine:3.21
RUN apk add --no-cache tzdata ca-certificates
ENV TZ_NAME=Asia/Tashkent TZ=Asia/Tashkent
COPY --from=build /out/app /app
EXPOSE 8080
ENTRYPOINT ["/bin/sh", "-c", "/app migrate up && exec /app serve"]
