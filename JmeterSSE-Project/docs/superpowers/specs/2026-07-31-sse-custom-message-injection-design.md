# SSE custom message injection — design

## Purpose

Allow a client (initially a JMeter test) to push a human-readable custom
message into an already-open `/stream` connection, so the delivery latency
of that message can later be measured once an LLM is wired up to respond to
it. SSE is server→client only, so this requires a side channel: a new
`POST /message` endpoint that targets one specific open stream connection.

## Architecture / data flow

```
client A: GET /stream
  -> server generates clientId, registers a channel for it
  -> "hello" event includes clientId in its data
  -> loop: select on 1s ticker (auto message-N events)
                  the registered channel (custom events)
                  request context Done() (disconnect/cleanup)

client A (same or different sampler): POST /message
  body: {"clientId": "...", "message": "..."}
  -> look up channel by clientId
  -> found, accepted:        200 {"delivered": true}
  -> found, buffer full:     409 {"delivered": false, "reason": "busy"}
  -> not found/disconnected: 404 {"delivered": false, "reason": "unknown_client"}

shortly after, on the /stream side:
  id: custom-1
  event: custom
  data: <the message text>
```

Automatic `message-N` events keep flowing on their existing 1s cadence;
custom messages interleave with them, they don't replace or pause them.

## Components

- **`ClientRegistry`** (new): single source of truth for connected clients,
  replacing the existing unguarded `clients map[string]*Client` in
  `main.go` entirely (not alongside it). Holds
  `map[string]*registeredClient` guarded by one `sync.Mutex`, where
  `registeredClient` bundles `*Client` (existing `/status` bookkeeping
  struct) and `ch chan string` (for custom-message delivery). Methods:
  `Register(id string, client *Client) chan string`,
  `Send(id, msg string) sendResult` (see Concurrency below for the
  three-way result), `Unregister(id string)`, `Snapshot() map[string]*Client`
  (returns a copy for `StatusHandler` to marshal, taking the lock
  internally — `StatusHandler` and `StreamHandler` never touch the map
  directly, so there is no unguarded access path left anywhere).
- **`StreamHandler`** (main.go, modified): generates a `clientId` via
  `crypto/rand` (e.g. 8 random bytes, hex-encoded) — deliberately NOT
  `GenerateRandomString`, which is seeded and deterministic by design (used
  for reproducible per-message content, `randstr.go`); reusing it for an
  identifier that must be unique across concurrently-opened connections
  would let two connections opened in the same instant collide. Includes
  the generated id in the `hello` event's JSON data, registers via
  `ClientRegistry.Register`, and replaces the current
  `for ; client.LastEventId < limit; client.LastEventId++` loop with:
  ```go
  ticker := time.NewTicker(1 * time.Second)
  defer ticker.Stop()
  defer registry.Unregister(clientId)
  customSeq := 0
  for {
      select {
      case <-ticker.C:
          if client.LastEventId >= limit {
              return // existing count/limit bound, preserved
          }
          // existing message-N emission, unchanged
          client.LastEventId++
      case msg := <-ch:
          customSeq++
          srw.WriteTypedEvent(fmt.Sprintf("custom-%d", customSeq), "custom", msg)
      case <-request.Context().Done():
          return
      }
  }
  ```
  The `count`/`Last-Event-Id` bound continues to apply only to automatic
  `message-N` events, exactly as today — custom messages are never counted
  against `limit`, and this loop must not close the stream while custom
  messages could still arrive within `limit`'s window (i.e. checking the
  bound only in the ticker branch, not as a separate loop condition, is
  intentional).
- **`WriteTypedEvent`** (main.go, new method on `StreamResponseWriter`):
  extends `WriteEvent` to also write an `event: <type>` line before `data:`
  lines. `WriteEvent` itself stays as-is (used for `hello`/`message-N`,
  which have no explicit type today) or becomes a thin wrapper calling
  `WriteTypedEvent(id, "", data)` that skips the `event:` line when type is
  empty — implementation detail, not user-visible either way.
- **`MessageHandler`** (main.go, new): `POST /message`, decodes JSON body
  `{clientId, message}`, calls `registry.Send`, maps the result to a status
  code per Error handling below. Registered in `main()` wrapped with the
  same `authMiddleware` + `loggingMiddleware` as the existing routes.

## Concurrency

`ClientRegistry`'s mutex guards every map access — register, send lookup,
unregister, and the `/status` snapshot — so there is no unguarded path left
(closing the gap in the current code where `ctx.clients` is touched from
`StreamHandler` and `StatusHandler` with no lock at all).

`Send` uses a buffered channel (size 1) per client with a non-blocking
`select`/`default`, and returns one of three results so the two failure
modes aren't conflated:
- `sendOK` — channel accepted the message.
- `sendUnknownClient` — no registered client for that id (never connected,
  or already disconnected/unregistered).
- `sendBusy` — client is registered but its buffer is still full (previous
  custom message not yet drained by the 1s-cadence loop).

A full buffer is never blocked on — the POST caller always gets an
immediate response either way.

## Error handling

- Unknown/disconnected `clientId` on POST → `404 {"delivered": false,
  "reason": "unknown_client"}`.
- Registered but not yet drained (`sendBusy`) → `409 {"delivered": false,
  "reason": "busy"}` — distinct from 404 so a caller can tell "bad id" from
  "retry shortly" instead of getting the same response for both.
- Malformed JSON body on POST → `400`.
- `AUTH_TOKEN` set and missing/wrong bearer token on POST → `401`, same as
  `/stream`/`/status` today.

## Testing

Add `main_test.go` covering `ClientRegistry` directly: register → send →
receive on the channel (`sendOK`); send to an unknown id → `sendUnknownClient`;
send twice without draining → second call `sendBusy`. No HTTP-level
integration tests — this repo has no existing test infrastructure, and
the registry logic is the only new piece with real correctness risk
(concurrent map access, channel delivery, the three-way result). Manual
verification against the running server (via `curl`/JMeter) covers the
HTTP wiring.

## Out of scope

- Any actual LLM integration or auto-response logic — this design only
  gets a message onto an open stream; what (if anything) responds to it is
  future work.
- Broadcast-to-all-clients — deliberately not built, per targeted-delivery
  decision above.
- Message history/replay for a client that reconnects mid-conversation —
  not requested, would need a persistence layer this design doesn't have.
