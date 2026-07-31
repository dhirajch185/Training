# LLM responder + quiet stream mode — design

## Purpose

Let an injected custom message trigger an LLM call whose reply is pushed
back onto the same client's SSE stream, so JMeter can measure end-to-end
message→response latency (POST timestamp → `llm-response` event arrival).
Also add a quiet stream mode so a stream can carry only the conversation
(no automatic 1/sec `message-N` ticker events).

Free-tier reality drives the provider design: hosted free tiers are
rate-limited, so load runs use a mock provider and real-latency runs use a
real provider (Gemini) at low thread counts.

## Trigger & flow

- `LLM_PROVIDER` env var: `mock` | `gemini` | unset. Unset = feature
  entirely off, server behaves exactly as today.
- When enabled: each accepted `POST /message` still delivers the
  `event: custom` to the target stream exactly as today, and additionally
  fires the LLM call in a new goroutine. When the reply arrives it is
  pushed to the same client's stream as `event: llm-response`.
- The POST response is unchanged (`200 {"delivered": true}` means the
  message was accepted, not that the LLM finished). JMeter measures POST
  time → `llm-response` arrival on the SSE sampler.

## Channel change (the only structural change)

Registry channel `chan string` becomes `chan streamEvent`:

```go
type streamEvent struct {
    Type string // "custom", "llm-response", "llm-error"
    Data string
}
```

- Custom message → `{Type: "custom", Data: msg}`.
- LLM reply → `{Type: "llm-response", Data: text}`.
- LLM failure → `{Type: "llm-error", Data: short reason}`.
- Buffer size grows 1 → 4 (an LLM reply can arrive while a custom message
  sits undrained). `Send` keeps its non-blocking three-way result
  (`sendOK`/`sendBusy`/`sendUnknownClient`); `sendBusy` now means "buffer
  of 4 full".
- `StreamHandler`'s channel case becomes
  `WriteTypedEvent("", ev.Type, ev.Data)` — still no `id:` line for any
  channel-delivered event, so the `Last-Event-Id` resume cursor remains
  owned exclusively by automatic `message-N` events.

## Provider interface

```go
type LLMProvider interface {
    Complete(ctx context.Context, prompt string) (string, error)
}
```

New file `llm.go`. Two implementations, stdlib only (`net/http`,
`encoding/json`) — no SDK dependency:

- **`mockProvider`** — returns a canned reply after a configurable delay.
  Env: `MOCK_DELAY_MS` (default 500). Used for high-concurrency load runs.
- **`geminiProvider`** — REST `generateContent` call, non-streaming.
  Env: `GEMINI_API_KEY` (required when `LLM_PROVIDER=gemini`; fatal error
  at startup if missing), `LLM_MODEL` (default set at implementation time
  from current Gemini docs — model names and endpoint verified against the
  live docs during implementation, not hardcoded from memory). 30-second
  context timeout per call.

Provider selection happens once in `main()`; `MessageHandler` (or a small
responder helper it calls) receives the chosen provider. Unknown
`LLM_PROVIDER` value → fatal error at startup (fail loud, not silently
off).

## Error handling

- LLM error or timeout → `event: llm-error` with a short reason string on
  the target stream. JMeter asserts on absence of `llm-error` /presence of
  `llm-response`.
- Client disconnected before the reply arrives → the non-blocking send
  drops the event; log one line to stdout. No retry, no queueing.
- Gemini 429/5xx → just an `llm-error`. Mitigation is test design (mock
  for volume, Gemini at 1-2 threads), not retry logic.

## Quiet mode

- `GET /stream?quiet=1` — `hello` event still sent (clientId discovery
  unchanged), but the 1-second ticker never starts: the select loop has
  only the channel case and request-context cancellation.
- The stream then carries exclusively injected `custom` events and
  `llm-response`/`llm-error` events.
- `count` is meaningless with `quiet=1` (there is nothing to count) —
  ignored, documented as such. `X-Expected-Events` is not sent for quiet
  streams.
- Default (no `quiet` param, or any value other than `1`) is byte-for-byte
  today's behavior.

## Testing

- Mock-provider end-to-end path: POST `/message` against a live
  `StreamHandler` recorder (existing stream_handler_test.go pattern) with
  `LLM_PROVIDER=mock` wiring → assert `event: llm-response` appears on the
  stream after the `event: custom`.
- Gemini provider: one unit test of request building + response parsing
  against an `httptest` fake Gemini server (assert prompt lands in the
  request JSON, reply text is extracted from the response JSON, non-200
  maps to error). No real API calls in tests.
- Quiet mode: request `/stream?quiet=1` with a cancellable context, hold
  it open briefly, assert no `message-N` event appears and `hello` does.
- Real-key Gemini smoke test stays manual (curl/JMeter).

## Out of scope

- Streaming/chunked LLM replies (single full-reply event only; revisit if
  time-to-first-token measurement is ever needed).
- Retry/backoff on provider errors, response caching, rate-limit tracking.
- Conversation history/context — each message is a standalone prompt.
- Any additional providers beyond mock + Gemini (interface makes them
  cheap to add later).
- JMeter script changes (separate follow-up once server ships).
