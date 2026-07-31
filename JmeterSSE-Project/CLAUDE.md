# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

`sse-server` is a small Go HTTP server that exposes a Server-Sent Events
stream (`/stream`) for testing SSE client libraries, plus a `/status`
endpoint listing connected clients, and a `/message` endpoint for injecting
custom messages into open streams. Single package `main`, five files, tests,
no external dependencies (stdlib only, `go.mod` has zero requires).

## Commands

```
go build              # build binary (also: make build)
go build && ./sse-server   # run locally, listens on :8080
GOOS=darwin GOARCH=arm64 go build   # make macos
GOOS=linux GOARCH=amd64 go build    # make linux
docker build -t sse-server .
docker run -p 8080:8080 sse-server
```

Tests: `go test ./...` runs unit tests for handlers and registry.

Env vars: `PORT` (default 8080), `AUTH_TOKEN` (enables bearer-token auth),
`AUTH_TOKEN_FILE` (path to token file, takes precedence over `AUTH_TOKEN`
if readable).

## Architecture

- **main.go** — `HandlerContext` holds a reference to `ClientRegistry`.
  `StreamHandler` drives the SSE loop: reads `Last-Event-Id` header to
  resume a client's sequence position, honors `?count=` to cap the number
  of events before closing, writes one event/sec via
  `StreamResponseWriter.WriteTypedEvent` (sets `id:`/`event:`/`data:` lines
  and flushes through `http.ResponseController`). The event loop is a
  `select` over a 1-second ticker (for `message-N` events), the
  `ClientRegistry` channel (for custom messages), and request-context
  cancellation. `StatusHandler` dumps registry snapshot as JSON.
  `MessageHandler` is the new third route, accepting POST requests to
  inject custom messages via `registry.Send()`.
- **registry.go** — `ClientRegistry` is the single source of truth for
  connected clients (replaces the old unguarded `clients` map), keyed by a
  `crypto/rand`-generated `clientId` (not `GenerateRandomString`, which is
  deterministic by design and unsuitable as a unique id). `Register`
  creates a buffered channel (size 1) for custom messages and stores the
  client. `Send` delivers a custom message onto a client's channel if
  connected and not already backed up, used by `POST /message`. `Unregister`
  removes a client when the stream closes. `Snapshot` returns a copy of all
  connected clients for `/status`.
- **middlewares.go** — `HttpMiddleware` is `func(HandlerFunc) HandlerFunc`;
  `AdaptHandler` composes middleware around a `HandlerFunc` and returns a
  stdlib `http.HandlerFunc`. Two middlewares: `NewLoggingMiddleware` (writes
  one log line per request after it completes) and `NewAuthMiddleware`
  (rejects with 401 if `AuthValidator` fails — CORS headers are duplicated
  here for the 401 path since the normal response path sets them separately
  in `StreamHandler`).
- **auth.go** — `AuthValidator func(*http.Request) bool`. Two
  implementations: `NewAllowAllAuthValidator` (default, no `AUTH_TOKEN` set)
  and `NewTokenAuthValidator` (exact `Authorization: Bearer <token>` match).
- **randstr.go** — `GenerateRandomString(seq, length)` is deterministic per
  `seq` (seeds `math/rand` with `seq` as the source), so the same message
  number always produces the same random string — this is intentional, it
  lets SSE clients/tests verify resumed streams (via `Last-Event-Id`)
  reproduce identical content.
- CORS is wide open by design (`Access-Control-Allow-Origin: *`,
  `Access-Control-Allow-Headers: *`) — this is a test/demo server, not
  production-hardened.
- Dockerfile builds a static binary (`CGO_ENABLED=0`) into a `scratch` image
  running as non-root `scratchuser`.
