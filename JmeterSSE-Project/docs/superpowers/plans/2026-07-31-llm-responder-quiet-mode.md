# LLM Responder + Quiet Mode Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** An injected custom message triggers an LLM call (mock or Gemini) whose reply is pushed onto the same client's SSE stream as `event: llm-response`; plus `?quiet=1` streams that carry only conversation events (no automatic ticker).

**Architecture:** Registry channel becomes `chan streamEvent{Type,Data}` so multiple event types flow to a stream. `MessageHandler` fires an async provider call after delivering the custom event. Provider behind a 1-method interface in new `llm.go` (mock + Gemini REST, stdlib only). Quiet mode skips ticker creation in `StreamHandler`.

**Tech Stack:** Go 1.22 stdlib only. Tests via `net/http/httptest` (existing patterns in `stream_handler_test.go`, `message_handler_test.go`).

Spec: `docs/superpowers/specs/2026-07-31-llm-responder-quiet-mode-design.md`

## Global Constraints

- Go 1.22, stdlib only — no SDKs, no new dependencies.
- `LLM_PROVIDER` unset ⇒ server behavior byte-for-byte identical to today (feature fully off). Unknown value ⇒ fatal at startup.
- Channel-delivered events (`custom`, `llm-response`, `llm-error`) NEVER carry an `id:` line — `Last-Event-Id` resume cursor stays owned by automatic `message-N` events only.
- `POST /message` response contract unchanged: `200 {"delivered":true}` means accepted, not LLM-complete.
- Existing `?count=`/`Last-Event-Id` behavior unchanged for non-quiet streams.
- Gemini endpoint (verified from live docs 2026-07-31): `POST https://generativelanguage.googleapis.com/v1beta/models/<model>:generateContent`, header `x-goog-api-key: <key>`, body `{"contents":[{"parts":[{"text":"<prompt>"}]}]}`, reply text at `candidates[0].content.parts[0].text`. Default `LLM_MODEL`: `gemini-3.5-flash`.
- No real network calls in tests — Gemini tested against an `httptest` fake server.
- Environment: no local Go toolchain; run Go via Docker from the worktree root:
  `MSYS_NO_PATHCONV=1 docker run --rm -v "$(pwd):/app" -w /app golang:1.22 go test ./... -v`

---

### Task 1: streamEvent channel

**Files:**
- Modify: `registry.go` (channel type, Send signature)
- Modify: `main.go` (StreamHandler channel case, MessageHandler Send call)
- Modify: `registry_test.go`, `message_handler_test.go`, `stream_handler_test.go` (compile fixes for new types)

**Interfaces:**
- Consumes: existing `ClientRegistry`, `StreamHandler`, `MessageHandler`.
- Produces: `type streamEvent struct { Type string; Data string }`; `Register(id string, client *Client) chan streamEvent`; `Send(id string, ev streamEvent) sendResult`. Buffer size 4. Tasks 2-4 rely on these exact signatures.

- [ ] **Step 1: Update `registry.go`**

Add the type above `registeredClient` and change the channel type and buffer:

```go
type streamEvent struct {
	Type string // "custom", "llm-response", "llm-error"
	Data string
}
```

- `registeredClient.ch` becomes `chan streamEvent`.
- `Register` returns `chan streamEvent`; `make(chan streamEvent, 4)`.
- `Send(id string, ev streamEvent) sendResult` — body unchanged apart from the parameter.

- [ ] **Step 2: Update `main.go` call sites**

`StreamHandler`'s channel case:

```go
		case ev := <-ch:
			err := srw.WriteTypedEvent("", ev.Type, ev.Data)
			if err != nil {
				return
			}
```

`MessageHandler`'s send:

```go
	result := ctx.registry.Send(body.ClientId, streamEvent{Type: "custom", Data: body.Message})
```

- [ ] **Step 3: Fix tests to compile with the new types**

In `registry_test.go`: every `r.Send("abc", "hello")` becomes `r.Send("abc", streamEvent{Type: "custom", Data: "hello"})`; channel receives assert `msg.Data`. The busy test must now fill the buffer: send 4 events expecting `sendOK`, assert the 5th returns `sendBusy`.

In `stream_handler_test.go`: `ctx.registry.Send(clientId, "hi there")` becomes `ctx.registry.Send(clientId, streamEvent{Type: "custom", Data: "hi there"})`.

`message_handler_test.go`: the delivered test's channel drain asserts `msg.Data == "hi"` and `msg.Type == "custom"`. The busy test sends 4 pre-fill messages via `ctx.registry.Send(...)` before the POST that must 409.

- [ ] **Step 4: Run tests**

Run: `MSYS_NO_PATHCONV=1 docker run --rm -v "$(pwd):/app" -w /app golang:1.22 go test ./... -v`
Expected: all pass (same count as before; no new tests this task — this is a mechanical type migration protected by the existing suite).

- [ ] **Step 5: Commit**

```bash
git add registry.go main.go registry_test.go message_handler_test.go stream_handler_test.go
git commit -m "refactor: registry channel carries typed streamEvent, buffer 4"
```

---

### Task 2: Quiet mode

**Files:**
- Modify: `main.go` (`StreamHandler`)
- Test: `stream_handler_test.go`

**Interfaces:**
- Consumes: Task 1's `streamEvent` channel.
- Produces: `?quiet=1` behavior; no new Go symbols.

- [ ] **Step 1: Write the failing test**

Append to `stream_handler_test.go`:

```go
func TestStreamHandlerQuietModeSuppressesTicker(t *testing.T) {
	handlerCtx := NewHandlerContext()

	req := httptest.NewRequest(http.MethodGet, "/stream?quiet=1", nil)
	reqCtx, cancel := context.WithCancel(req.Context())
	req = req.WithContext(reqCtx)

	rec := httptest.NewRecorder()
	srw := NewStatusResponseWriter(rec)

	done := make(chan struct{})
	go func() {
		handlerCtx.StreamHandler(srw, req)
		close(done)
	}()

	// Hold the stream open long enough that a ticker (1s) would have fired.
	time.Sleep(1500 * time.Millisecond)
	cancel()
	<-done

	body := rec.Body.String()
	if !strings.Contains(body, "id: hello\n") {
		t.Fatalf("expected hello event, got: %q", body)
	}
	if strings.Contains(body, "id: message-") {
		t.Fatalf("expected no automatic message-N events in quiet mode, got: %q", body)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `MSYS_NO_PATHCONV=1 docker run --rm -v "$(pwd):/app" -w /app golang:1.22 go test ./... -run TestStreamHandlerQuietMode -v`
Expected: FAIL — body contains `id: message-1` (ticker still runs).

- [ ] **Step 3: Implement**

In `StreamHandler`, read the param next to `count` parsing:

```go
	quiet := request.FormValue("quiet") == "1"
```

Skip `X-Expected-Events` when quiet (wrap the existing `if count != math.MaxInt` in `if !quiet && count != math.MaxInt`). After the hello write, branch the loop: when quiet, select only on the channel and context (no ticker, no bound checks — `count` is documented as ignored):

```go
	if quiet {
		for {
			select {
			case ev := <-ch:
				if err := srw.WriteTypedEvent("", ev.Type, ev.Data); err != nil {
					return
				}
			case <-request.Context().Done():
				return
			}
		}
	}
```

placed before the existing pre-loop `if client.LastEventId >= limit` check so bounded-close logic never runs for quiet streams. The existing non-quiet loop stays untouched below it.

- [ ] **Step 4: Run tests**

Run: `MSYS_NO_PATHCONV=1 docker run --rm -v "$(pwd):/app" -w /app golang:1.22 go test ./... -v`
Expected: new test passes, all existing pass.

- [ ] **Step 5: Commit**

```bash
git add main.go stream_handler_test.go
git commit -m "feat: add quiet=1 stream mode without automatic ticker events"
```

---

### Task 3: LLM providers (mock + Gemini)

**Files:**
- Create: `llm.go`
- Test: `llm_test.go`

**Interfaces:**
- Consumes: nothing from other tasks (standalone).
- Produces: `type LLMProvider interface { Complete(ctx context.Context, prompt string) (string, error) }`; `func NewMockProvider(delay time.Duration) LLMProvider`; `func NewGeminiProvider(apiKey, model, baseURL string) LLMProvider` (baseURL parameter exists for tests; production passes `"https://generativelanguage.googleapis.com"`); `func ProviderFromEnv() (LLMProvider, error)` returning `(nil, nil)` when `LLM_PROVIDER` is unset. Task 4 consumes all of these.

- [ ] **Step 1: Write the failing tests**

Create `llm_test.go`:

```go
package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMockProviderReturnsCannedReply(t *testing.T) {
	p := NewMockProvider(10 * time.Millisecond)
	start := time.Now()
	reply, err := p.Complete(context.Background(), "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reply == "" {
		t.Fatal("expected non-empty reply")
	}
	if time.Since(start) < 10*time.Millisecond {
		t.Fatal("expected delay to be applied")
	}
}

func TestGeminiProviderSendsPromptAndParsesReply(t *testing.T) {
	var gotPath, gotKey string
	var gotBody map[string]any
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotKey = r.Header.Get("x-goog-api-key")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"a celebrity"}]}}]}`))
	}))
	defer fake.Close()

	p := NewGeminiProvider("test-key", "gemini-3.5-flash", fake.URL)
	reply, err := p.Complete(context.Background(), "who has a birthday today?")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reply != "a celebrity" {
		t.Fatalf("expected parsed reply, got %q", reply)
	}
	if gotPath != "/v1beta/models/gemini-3.5-flash:generateContent" {
		t.Fatalf("wrong path: %q", gotPath)
	}
	if gotKey != "test-key" {
		t.Fatalf("wrong api key header: %q", gotKey)
	}
	b, _ := json.Marshal(gotBody)
	if want := "who has a birthday today?"; !json.Valid(b) || !containsString(string(b), want) {
		t.Fatalf("prompt not found in request body: %s", b)
	}
}

func TestGeminiProviderMapsHTTPErrorToError(t *testing.T) {
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer fake.Close()

	p := NewGeminiProvider("k", "m", fake.URL)
	_, err := p.Complete(context.Background(), "hi")
	if err == nil {
		t.Fatal("expected error for 429 response")
	}
}
```

Add tiny helper at the bottom of `llm_test.go` (avoids importing strings just for one call, or just use `strings.Contains` — implementer's choice, `strings` is fine):

```go
func containsString(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
}
```

(If using `strings.Contains` directly, add `"strings"` to imports and drop the helper.)

- [ ] **Step 2: Run tests to verify they fail**

Run: `MSYS_NO_PATHCONV=1 docker run --rm -v "$(pwd):/app" -w /app golang:1.22 go test ./... -run 'TestMockProvider|TestGeminiProvider' -v`
Expected: build error — `NewMockProvider`, `NewGeminiProvider` undefined.

- [ ] **Step 3: Implement `llm.go`**

```go
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"
)

type LLMProvider interface {
	Complete(ctx context.Context, prompt string) (string, error)
}

const llmCallTimeout = 30 * time.Second
const defaultGeminiModel = "gemini-3.5-flash"
const geminiBaseURL = "https://generativelanguage.googleapis.com"

type mockProvider struct {
	delay time.Duration
}

func NewMockProvider(delay time.Duration) LLMProvider {
	return &mockProvider{delay: delay}
}

func (m *mockProvider) Complete(ctx context.Context, prompt string) (string, error) {
	select {
	case <-time.After(m.delay):
	case <-ctx.Done():
		return "", ctx.Err()
	}
	return fmt.Sprintf("mock reply to: %s", prompt), nil
}

type geminiProvider struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

func NewGeminiProvider(apiKey, model, baseURL string) LLMProvider {
	return &geminiProvider{
		apiKey:  apiKey,
		model:   model,
		baseURL: baseURL,
		client:  &http.Client{},
	}
}

type geminiRequest struct {
	Contents []geminiContent `json:"contents"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiResponse struct {
	Candidates []struct {
		Content geminiContent `json:"content"`
	} `json:"candidates"`
}

func (g *geminiProvider) Complete(ctx context.Context, prompt string) (string, error) {
	payload, err := json.Marshal(geminiRequest{
		Contents: []geminiContent{{Parts: []geminiPart{{Text: prompt}}}},
	})
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent", g.baseURL, g.model)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", g.apiKey)

	resp, err := g.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("gemini API returned HTTP %d", resp.StatusCode)
	}

	var parsed geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", err
	}
	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini API returned no candidates")
	}
	return parsed.Candidates[0].Content.Parts[0].Text, nil
}

// ProviderFromEnv returns (nil, nil) when LLM_PROVIDER is unset (feature off).
func ProviderFromEnv() (LLMProvider, error) {
	switch os.Getenv("LLM_PROVIDER") {
	case "":
		return nil, nil
	case "mock":
		delay := 500 * time.Millisecond
		if ms := os.Getenv("MOCK_DELAY_MS"); ms != "" {
			if n, err := strconv.Atoi(ms); err == nil && n >= 0 {
				delay = time.Duration(n) * time.Millisecond
			}
		}
		return NewMockProvider(delay), nil
	case "gemini":
		key := os.Getenv("GEMINI_API_KEY")
		if key == "" {
			return nil, fmt.Errorf("LLM_PROVIDER=gemini requires GEMINI_API_KEY")
		}
		model := os.Getenv("LLM_MODEL")
		if model == "" {
			model = defaultGeminiModel
		}
		return NewGeminiProvider(key, model, geminiBaseURL), nil
	default:
		return nil, fmt.Errorf("unknown LLM_PROVIDER %q (want mock or gemini)", os.Getenv("LLM_PROVIDER"))
	}
}
```

- [ ] **Step 4: Run tests**

Run: `MSYS_NO_PATHCONV=1 docker run --rm -v "$(pwd):/app" -w /app golang:1.22 go test ./... -v`
Expected: all pass including the three new provider tests.

- [ ] **Step 5: Commit**

```bash
git add llm.go llm_test.go
git commit -m "feat: add LLMProvider interface with mock and Gemini implementations"
```

---

### Task 4: Wire the responder

**Files:**
- Modify: `main.go` (`HandlerContext`, `MessageHandler`, `main()`)
- Test: `message_handler_test.go` or `stream_handler_test.go` (end-to-end mock test)

**Interfaces:**
- Consumes: `streamEvent` + `Send` (Task 1); `LLMProvider`, `ProviderFromEnv`, `NewMockProvider`, `llmCallTimeout` (Task 3).
- Produces: `HandlerContext.llm LLMProvider` field (nil = off). No other new symbols.

- [ ] **Step 1: Write the failing test**

Append to `stream_handler_test.go` (reuses that file's live-handler pattern — read the existing `TestStreamHandlerCustomMessageHasNoIdLine` first and mirror its structure):

```go
func TestMessageTriggersLLMResponseOnStream(t *testing.T) {
	handlerCtx := NewHandlerContext()
	handlerCtx.llm = NewMockProvider(10 * time.Millisecond)

	req := httptest.NewRequest(http.MethodGet, "/stream?quiet=1", nil)
	reqCtx, cancel := context.WithCancel(req.Context())
	req = req.WithContext(reqCtx)
	rec := httptest.NewRecorder()
	srw := NewStatusResponseWriter(rec)

	done := make(chan struct{})
	go func() {
		handlerCtx.StreamHandler(srw, req)
		close(done)
	}()

	clientId := waitForClientId(t, rec) // extract from hello event; helper mirrors existing test's extraction, factor it out if not already shared

	body, _ := json.Marshal(messageRequest{ClientId: clientId, Message: "ping"})
	postReq := httptest.NewRequest(http.MethodPost, "/message", bytes.NewReader(body))
	postRec := httptest.NewRecorder()
	handlerCtx.MessageHandler(NewStatusResponseWriter(postRec), postReq)
	if postRec.Code != http.StatusOK {
		t.Fatalf("expected 200 from POST, got %d", postRec.Code)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(rec.Body.String(), "event: llm-response\n") {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	cancel()
	<-done

	streamBody := rec.Body.String()
	if !strings.Contains(streamBody, "event: custom\ndata: ping\n") {
		t.Fatalf("expected custom event on stream, got: %q", streamBody)
	}
	if !strings.Contains(streamBody, "event: llm-response\n") {
		t.Fatalf("expected llm-response event on stream, got: %q", streamBody)
	}
	if !strings.Contains(streamBody, "mock reply to: ping") {
		t.Fatalf("expected mock reply text on stream, got: %q", streamBody)
	}
}
```

(`waitForClientId` — if the existing custom-message test extracts the clientId inline, extract that logic into this shared helper as part of this step so both tests use it.)

- [ ] **Step 2: Run test to verify it fails**

Run: `MSYS_NO_PATHCONV=1 docker run --rm -v "$(pwd):/app" -w /app golang:1.22 go test ./... -run TestMessageTriggersLLM -v`
Expected: build error — `HandlerContext` has no field `llm`.

- [ ] **Step 3: Implement**

`HandlerContext` gains the field (nil = feature off):

```go
type HandlerContext struct {
	registry *ClientRegistry
	llm      LLMProvider
}
```

In `MessageHandler`, after the `sendOK` case is decided (only on successful delivery), fire the async call:

```go
	if result == sendOK && ctx.llm != nil {
		go ctx.respondWithLLM(body.ClientId, body.Message)
	}
```

New method on `HandlerContext`:

```go
func (ctx *HandlerContext) respondWithLLM(clientId string, prompt string) {
	callCtx, cancel := context.WithTimeout(context.Background(), llmCallTimeout)
	defer cancel()

	reply, err := ctx.llm.Complete(callCtx, prompt)
	ev := streamEvent{Type: "llm-response", Data: reply}
	if err != nil {
		ev = streamEvent{Type: "llm-error", Data: err.Error()}
	}
	if res := ctx.registry.Send(clientId, ev); res != sendOK {
		fmt.Printf("LLM %s for %s dropped (%v)\n", ev.Type, clientId, res)
	}
}
```

(`context` import added to main.go.) In `main()`, after the auth setup:

```go
	llm, err := ProviderFromEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "LLM configuration error: %v\n", err)
		os.Exit(1)
	}
	ctx := NewHandlerContext()
	ctx.llm = llm
	if llm != nil {
		fmt.Printf("LLM responder enabled (%s)\n", os.Getenv("LLM_PROVIDER"))
	}
```

- [ ] **Step 4: Run all tests**

Run: `MSYS_NO_PATHCONV=1 docker run --rm -v "$(pwd):/app" -w /app golang:1.22 go test ./... -v`
Expected: all pass. Existing MessageHandler tests unaffected (nil provider = no goroutine fired).

- [ ] **Step 5: Manual smoke test (mock)**

```bash
docker compose build
LLM_PROVIDER=mock docker compose up   # or add environment: [LLM_PROVIDER=mock] temporarily
```

Actually for compose, run instead: `docker run --rm -p 8081:8080 -e LLM_PROVIDER=mock sse-server`.
Terminal A: `curl -N "http://localhost:8081/stream?quiet=1"` → note clientId.
Terminal B: POST a message; within ~500ms terminal A shows `event: custom` then `event: llm-response` / `data: mock reply to: ...` and nothing else. Note in the report this was run (it can be, no live external API needed).

- [ ] **Step 6: Commit**

```bash
git add main.go stream_handler_test.go message_handler_test.go
git commit -m "feat: wire async LLM responder into MessageHandler"
```

---

### Task 5: Documentation

**Files:**
- Modify: `README.md`, `CLAUDE.md`, `compose.yaml`

**Interfaces:** none produced; documents Tasks 1-4 behavior.

- [ ] **Step 1: README.md**

- New `### LLM responder` section after `### POST /message`: env vars table (`LLM_PROVIDER` mock|gemini, `GEMINI_API_KEY`, `LLM_MODEL` default `gemini-3.5-flash`, `MOCK_DELAY_MS` default 500), the flow (message → `event: custom` → async LLM → `event: llm-response` or `event: llm-error`), a mock curl example, and the free-tier note: use `mock` for load runs, `gemini` at 1-2 threads only (rate limits).
- New `#### quiet parameter` under `/stream` (after the `count` section): `?quiet=1` = hello + injected/LLM events only, no `message-N` ticker, `count` ignored.
- [ ] **Step 2: CLAUDE.md** — add `llm.go` bullet (provider interface, mock + Gemini REST, `ProviderFromEnv`); mention `quiet=1` and `streamEvent` channel in the `main.go`/`registry.go` bullets; file count five → six.
- [ ] **Step 3: compose.yaml** — add commented-out env lines showing how to enable:

```yaml
    # environment:
    #   - LLM_PROVIDER=mock        # or gemini (needs GEMINI_API_KEY)
    #   - MOCK_DELAY_MS=500
```

- [ ] **Step 4: Commit**

```bash
git add README.md CLAUDE.md compose.yaml
git commit -m "docs: document LLM responder, quiet mode, and compose env hooks"
```
