# SSE Custom Message Injection Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let a client push a human-readable custom message into an already-open `/stream` SSE connection via a new `POST /message` endpoint, targeted by a `clientId` the server hands out in the `hello` event.

**Architecture:** A new `ClientRegistry` (mutex-guarded `map[string]*registeredClient`, each holding a `*Client` and a buffered `chan string`) replaces the existing unguarded `clients` map. `StreamHandler`'s event loop becomes a `select` over a 1s ticker (existing automatic `message-N` events), the registered channel (custom messages), and request-context cancellation. `POST /message` looks a client up by id and sends onto its channel.

**Tech Stack:** Go 1.22 stdlib only (`crypto/rand`, `encoding/hex`, `sync`, `net/http/httptest` for tests) — no new dependencies.

Spec: `docs/superpowers/specs/2026-07-31-sse-custom-message-injection-design.md`

## Global Constraints

- Go 1.22, stdlib only — no new dependencies (matches existing `go.mod`, zero requires).
- Only unit tests via `net/http/httptest` for `ClientRegistry`/`MessageHandler` — no HTTP-level integration test framework, per spec's Testing section (this repo has no existing test infra; full server-boot tests are out of scope).
- The existing `?count=`/`Last-Event-Id` stream-bound behavior for automatic `message-N` events must be unchanged (spec: StreamHandler component).
- `POST /message` is wrapped with the same `authMiddleware` + `loggingMiddleware` as `/stream`/`/status` (spec: MessageHandler component, Error handling).
- Out of scope, do not build: broadcast-to-all-clients, any LLM integration/auto-response logic, message history/replay for a reconnecting client (spec: Out of scope).

---

### Task 1: ClientRegistry core

**Files:**
- Create: `registry.go`
- Test: `registry_test.go`

**Interfaces:**
- Consumes: `Client` struct from `main.go` (existing, will gain a `ClientId` field in Task 3 — this task only needs `RemoteAddr`, `ConnectedAt`, `LastEventId` which already exist).
- Produces: `type sendResult int` with values `sendOK`, `sendUnknownClient`, `sendBusy`; `type ClientRegistry struct{...}`; `func NewClientRegistry() *ClientRegistry`; `func (r *ClientRegistry) Register(id string, client *Client) chan string`; `func (r *ClientRegistry) Unregister(id string)`; `func (r *ClientRegistry) Send(id string, msg string) sendResult`; `func (r *ClientRegistry) Snapshot() map[string]*Client` (keyed by `client.RemoteAddr`, matching the existing `/status` JSON shape); `func newClientID() (string, error)`. Task 3 and Task 4 consume all of these exact names.

- [ ] **Step 1: Write the failing tests**

Create `registry_test.go`:

```go
package main

import "testing"

func TestClientRegistrySendOK(t *testing.T) {
	r := NewClientRegistry()
	ch := r.Register("abc", &Client{RemoteAddr: "1.2.3.4:1"})

	result := r.Send("abc", "hello")
	if result != sendOK {
		t.Fatalf("expected sendOK, got %v", result)
	}

	select {
	case msg := <-ch:
		if msg != "hello" {
			t.Fatalf("expected 'hello', got %q", msg)
		}
	default:
		t.Fatal("expected message on channel, got none")
	}
}

func TestClientRegistrySendUnknownClient(t *testing.T) {
	r := NewClientRegistry()

	result := r.Send("does-not-exist", "hello")
	if result != sendUnknownClient {
		t.Fatalf("expected sendUnknownClient, got %v", result)
	}
}

func TestClientRegistrySendBusy(t *testing.T) {
	r := NewClientRegistry()
	r.Register("abc", &Client{RemoteAddr: "1.2.3.4:1"})

	if result := r.Send("abc", "first"); result != sendOK {
		t.Fatalf("expected sendOK for first send, got %v", result)
	}

	result := r.Send("abc", "second")
	if result != sendBusy {
		t.Fatalf("expected sendBusy for second send, got %v", result)
	}
}

func TestClientRegistryUnregister(t *testing.T) {
	r := NewClientRegistry()
	r.Register("abc", &Client{RemoteAddr: "1.2.3.4:1"})
	r.Unregister("abc")

	result := r.Send("abc", "hello")
	if result != sendUnknownClient {
		t.Fatalf("expected sendUnknownClient after unregister, got %v", result)
	}
}

func TestClientRegistrySnapshot(t *testing.T) {
	r := NewClientRegistry()
	r.Register("abc", &Client{RemoteAddr: "1.2.3.4:1"})
	r.Register("def", &Client{RemoteAddr: "5.6.7.8:2"})

	snap := r.Snapshot()
	if len(snap) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(snap))
	}
	if _, ok := snap["1.2.3.4:1"]; !ok {
		t.Fatal("expected snapshot keyed by RemoteAddr to contain 1.2.3.4:1")
	}
}

func TestNewClientIDUnique(t *testing.T) {
	id1, err := newClientID()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	id2, err := newClientID()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id1 == id2 {
		t.Fatalf("expected different ids, got same: %q", id1)
	}
	if len(id1) != 16 {
		t.Fatalf("expected 16 hex chars (8 bytes), got %d: %q", len(id1), id1)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./... -run TestClientRegistry -v` and `go test ./... -run TestNewClientID -v`
Expected: FAIL / build error — `NewClientRegistry`, `sendOK`, etc. are undefined.

- [ ] **Step 3: Write the implementation**

Create `registry.go`:

```go
package main

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
)

type sendResult int

const (
	sendOK sendResult = iota
	sendUnknownClient
	sendBusy
)

type registeredClient struct {
	client *Client
	ch     chan string
}

type ClientRegistry struct {
	mu      sync.Mutex
	clients map[string]*registeredClient
}

func NewClientRegistry() *ClientRegistry {
	return &ClientRegistry{clients: make(map[string]*registeredClient)}
}

func (r *ClientRegistry) Register(id string, client *Client) chan string {
	r.mu.Lock()
	defer r.mu.Unlock()
	ch := make(chan string, 1)
	r.clients[id] = &registeredClient{client: client, ch: ch}
	return ch
}

func (r *ClientRegistry) Unregister(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.clients, id)
}

func (r *ClientRegistry) Send(id string, msg string) sendResult {
	r.mu.Lock()
	defer r.mu.Unlock()
	rc, ok := r.clients[id]
	if !ok {
		return sendUnknownClient
	}
	select {
	case rc.ch <- msg:
		return sendOK
	default:
		return sendBusy
	}
}

func (r *ClientRegistry) Snapshot() map[string]*Client {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make(map[string]*Client, len(r.clients))
	for _, rc := range r.clients {
		out[rc.client.RemoteAddr] = rc.client
	}
	return out
}

func newClientID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./... -v`
Expected: PASS for all `TestClientRegistry*` and `TestNewClientIDUnique`.

- [ ] **Step 5: Commit**

```bash
git add registry.go registry_test.go
git commit -m "feat: add ClientRegistry for tracking connected SSE clients"
```

---

### Task 2: WriteTypedEvent on StreamResponseWriter

**Files:**
- Modify: `main.go:36-60` (the `WriteEvent` method)
- Test: `main_test.go` (new)

**Interfaces:**
- Consumes: existing `StreamResponseWriter` struct and `NewStatusResponseWriter` (main.go:25-29, 62-68), unchanged.
- Produces: `func (srw *StreamResponseWriter) WriteTypedEvent(id string, eventType string, data string) error`. `WriteEvent` keeps its existing signature `func (srw *StreamResponseWriter) WriteEvent(id string, data string) error` but becomes a thin wrapper. Task 3 calls `WriteTypedEvent` directly for custom events and keeps calling `WriteEvent` for `hello`/`message-N`.

- [ ] **Step 1: Write the failing tests**

Create `main_test.go`:

```go
package main

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWriteEventNoType(t *testing.T) {
	rec := httptest.NewRecorder()
	srw := NewStatusResponseWriter(rec)

	if err := srw.WriteEvent("msg-1", "hello"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body := rec.Body.String()
	if strings.Contains(body, "event:") {
		t.Fatalf("expected no event: line, got body: %q", body)
	}
	if !strings.Contains(body, "id: msg-1\n") {
		t.Fatalf("expected id line, got body: %q", body)
	}
	if !strings.Contains(body, "data: hello\n") {
		t.Fatalf("expected data line, got body: %q", body)
	}
}

func TestWriteTypedEventWithType(t *testing.T) {
	rec := httptest.NewRecorder()
	srw := NewStatusResponseWriter(rec)

	if err := srw.WriteTypedEvent("custom-1", "custom", "hi there"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "id: custom-1\n") {
		t.Fatalf("expected id line, got body: %q", body)
	}
	if !strings.Contains(body, "event: custom\n") {
		t.Fatalf("expected event line, got body: %q", body)
	}
	if !strings.Contains(body, "data: hi there\n") {
		t.Fatalf("expected data line, got body: %q", body)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./... -run TestWriteTypedEvent -v` and `go test ./... -run TestWriteEventNoType -v`
Expected: FAIL — `WriteTypedEvent` undefined (or `TestWriteEventNoType` fails if it currently produces different output — either way, confirms the test exercises real behavior before the change).

- [ ] **Step 3: Write the implementation**

In `main.go`, replace the existing `WriteEvent` method (lines 36-60) with:

```go
func (srw *StreamResponseWriter) WriteTypedEvent(id string, eventType string, data string) error {
	_, err := fmt.Fprintf(srw, "id: %s\n", id)
	if err != nil {
		return err
	}

	if eventType != "" {
		_, err = fmt.Fprintf(srw, "event: %s\n", eventType)
		if err != nil {
			return err
		}
	}

	sc := bufio.NewScanner(strings.NewReader(data))
	for sc.Scan() {
		_, err = fmt.Fprintf(srw, "data: %s\n", sc.Text())
		if err != nil {
			return err
		}
	}
	_, err = fmt.Fprintf(srw, "\n")
	if err != nil {
		return err
	}

	err = srw.controller.Flush()
	if err != nil {
		return err
	}

	return nil
}

func (srw *StreamResponseWriter) WriteEvent(id string, data string) error {
	return srw.WriteTypedEvent(id, "", data)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./... -v`
Expected: PASS for `TestWriteEventNoType` and `TestWriteTypedEventWithType`, and all Task 1 tests still pass.

- [ ] **Step 5: Commit**

```bash
git add main.go main_test.go
git commit -m "feat: add WriteTypedEvent for SSE events with an explicit type"
```

---

### Task 3: Wire ClientRegistry into StreamHandler and StatusHandler

**Files:**
- Modify: `main.go:19-23` (`Client` struct)
- Modify: `main.go:70-76` (`HandlerContext` struct + constructor)
- Modify: `main.go:84-143` (`StreamHandler`)
- Modify: `main.go:145-154` (`StatusHandler`)

**Interfaces:**
- Consumes: `NewClientRegistry`, `(*ClientRegistry).Register`, `(*ClientRegistry).Unregister`, `(*ClientRegistry).Snapshot`, `newClientID` (Task 1); `WriteTypedEvent` (Task 2).
- Produces: `Client.ClientId string` field (Task 4/5 reference this in `/status` JSON output); `HandlerContext.registry *ClientRegistry` (Task 4 uses `ctx.registry.Send` directly).

This task has no new automated test of its own — the spec explicitly scopes automated testing to `ClientRegistry` and (in Task 4) `MessageHandler` via `httptest`, not full server-boot HTTP integration tests, since this repo has none today. Verification here is `go build` (compiles) plus a manual smoke test against the real running server, which is the fastest way to catch a bad wire-up in three self-contained handlers only used together in `main()`.

- [ ] **Step 1: Update the `Client` struct**

In `main.go`, replace lines 19-23:

```go
type Client struct {
	RemoteAddr  string    `json:"remote"`
	ConnectedAt time.Time `json:"connectedAt"`
	LastEventId int       `json:"lastEventId"`
	ClientId    string    `json:"clientId"`
}
```

- [ ] **Step 2: Replace `HandlerContext`**

Replace lines 70-76:

```go
type HandlerContext struct {
	registry *ClientRegistry
}

func NewHandlerContext() *HandlerContext {
	return &HandlerContext{registry: NewClientRegistry()}
}
```

- [ ] **Step 3: Replace `StreamHandler`**

Replace lines 84-143 (the whole `StreamHandler` method) with:

```go
func (ctx *HandlerContext) StreamHandler(srw *StreamResponseWriter, request *http.Request) {
	clientId, err := newClientID()
	if err != nil {
		srw.WriteHeader(http.StatusInternalServerError)
		return
	}

	client := &Client{
		RemoteAddr:  request.RemoteAddr,
		ConnectedAt: time.Now(),
		LastEventId: 1,
		ClientId:    clientId,
	}
	ch := ctx.registry.Register(clientId, client)

	defer func() {
		ctx.registry.Unregister(clientId)
		fmt.Printf("Client %s (%s) closed connection.\n", client.RemoteAddr, clientId)
	}()

	lastEventId := request.Header.Get("Last-Event-Id")
	_, _ = fmt.Sscanf(lastEventId, "message-%d", &client.LastEventId)

	limit := math.MaxInt
	count := math.MaxInt
	_, _ = fmt.Sscanf(request.FormValue("count"), "%d", &count)
	if count < 0 {
		srw.WriteHeader(http.StatusBadRequest)
		return
	}

	if count <= math.MaxInt-client.LastEventId {
		limit = client.LastEventId + count
	} else {
		limit = math.MaxInt
	}

	fmt.Printf("Starting stream for %s (%s), %d -> %d ...\n", client.RemoteAddr, clientId, client.LastEventId, limit)

	srw.Header().Set("Access-Control-Allow-Origin", "*")
	srw.Header().Set("Access-Control-Allow-Headers", "*")
	srw.Header().Set("Access-Control-Allow-Methods", "GET")
	srw.Header().Set("Content-Type", "text/event-stream")
	srw.Header().Set("Cache-Control", "no-cache")
	srw.Header().Set("Connection", "keep-alive")
	if count != math.MaxInt {
		srw.Header().Set("X-Expected-Events", strconv.Itoa(count))
	}
	srw.WriteHeader(http.StatusOK)

	err = srw.WriteEvent("hello", fmt.Sprintf("{\n  \"message\": \"Hello, %s!\",\n  \"clientId\": \"%s\"\n}", client.RemoteAddr, clientId))
	if err != nil {
		return
	}

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	customSeq := 0
	for {
		select {
		case <-ticker.C:
			if client.LastEventId >= limit {
				return
			}
			random := GenerateRandomString(client.LastEventId, randomStringLength)
			err := srw.WriteEvent(
				fmt.Sprintf("%s%d", messageIdPrefix, client.LastEventId),
				fmt.Sprintf("{\n  \"time\": %d,\n  \"random\": \"%s\"\n}", time.Now().Unix(), random))
			if err != nil {
				return
			}
			client.LastEventId++
		case msg := <-ch:
			customSeq++
			err := srw.WriteTypedEvent(fmt.Sprintf("custom-%d", customSeq), "custom", msg)
			if err != nil {
				return
			}
		case <-request.Context().Done():
			return
		}
	}
}
```

Note the bound check `client.LastEventId >= limit` happens only in the ticker branch (matching the original `for ; client.LastEventId < limit; client.LastEventId++` loop condition, which was likewise only evaluated once per automatic-message iteration) — a custom message arriving via `ch` never advances or is blocked by `limit`, per spec.

**REVISED:** during the final review round, two additional `client.LastEventId >= limit` bound checks were added beyond what's shown above — one immediately before entering the `for`/`select` loop (right after the `hello` event is written), and one immediately after `client.LastEventId++` inside the ticker branch (in addition to the check already shown at the top of that branch). Without them the stream would close one tick late (send one extra `message-N` event past `limit`) whenever `count` was small. See the actual `main.go` for the shipped bound-check placement.

**REVISED:** custom events ship with NO `id:` (empty string) instead of `custom-N` — the `srw.WriteTypedEvent(fmt.Sprintf("custom-%d", customSeq), "custom", msg)` call shown above, and the `customSeq` counter it depends on, do not exist in the shipped code. During review it was found that giving a custom event an `id: custom-N` clobbers the SSE `Last-Event-Id` resume cursor (a reconnecting client would resume from `custom-N` instead of its real `message-N` position, corrupting `Last-Event-Id`-based resume). The shipped call site is `srw.WriteTypedEvent("", "custom", msg)` — empty id, no `customSeq` variable at all. See the actual `main.go` for the shipped behavior.

- [ ] **Step 4: Replace `StatusHandler`**

Replace lines 145-154 (now shifted — locate by name, not line number, since Step 3 changed the file's line count):

```go
func (ctx *HandlerContext) StatusHandler(srw *StreamResponseWriter, request *http.Request) {
	body, err := json.Marshal(ctx.registry.Snapshot())
	if err != nil {
		srw.WriteHeader(http.StatusInternalServerError)
		return
	}

	srw.Header().Set("Content-Type", "application/json")
	_, _ = srw.Write(body)
}
```

- [ ] **Step 5: Build and run the full test suite**

Run: `go build ./...`
Expected: no errors.

Run: `go test ./... -v`
Expected: PASS — all Task 1 and Task 2 tests unaffected by this change.

- [ ] **Step 6: Manual smoke test**

```bash
go build && ./sse-server
```

In another terminal:

```bash
curl -N "http://localhost:8080/stream?count=3"
```

Expected: `hello` event's `data:` is now JSON containing both `message` and `clientId`, e.g.:
```
id: hello
data: {
data:   "message": "Hello, 127.0.0.1:54321!",
data:   "clientId": "a1b2c3d4e5f6a7b8"
data: }
```
followed by exactly 3 `message-N` events, then the connection closes (confirms `?count=` bound still works).

While a separate long-running stream is open (e.g. `curl -N http://localhost:8080/stream` in another terminal, no `count`), run:

```bash
curl http://localhost:8080/status
```

Expected: JSON keyed by remote address (e.g. `"127.0.0.1:54322"`), each entry now also containing a `"clientId"` field.

- [ ] **Step 7: Commit**

```bash
git add main.go
git commit -m "feat: wire ClientRegistry into StreamHandler and StatusHandler"
```

---

### Task 4: MessageHandler (POST /message)

**Files:**
- Modify: `main.go` (add types + handler near `StatusHandler`, wire route in `main()`)
- Test: `message_handler_test.go` (new)

**Interfaces:**
- Consumes: `HandlerContext.registry` (Task 3); `sendOK`, `sendUnknownClient`, `sendBusy` (Task 1).
- Produces: `type messageRequest struct{ ClientId string; Message string }`; `type messageResponse struct{ Delivered bool; Reason string }`; `func (ctx *HandlerContext) MessageHandler(srw *StreamResponseWriter, request *http.Request)`. Task 5's README examples reference this endpoint's exact request/response shape.

- [ ] **Step 1: Write the failing tests**

Create `message_handler_test.go`:

```go
package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMessageHandlerDelivered(t *testing.T) {
	ctx := NewHandlerContext()
	ch := ctx.registry.Register("abc", &Client{RemoteAddr: "1.2.3.4:1"})

	body, _ := json.Marshal(messageRequest{ClientId: "abc", Message: "hi"})
	req := httptest.NewRequest(http.MethodPost, "/message", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	srw := NewStatusResponseWriter(rec)

	ctx.MessageHandler(srw, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp messageResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unexpected error decoding response: %v", err)
	}
	if !resp.Delivered {
		t.Fatalf("expected delivered=true, got %+v", resp)
	}

	select {
	case msg := <-ch:
		if msg != "hi" {
			t.Fatalf("expected 'hi', got %q", msg)
		}
	default:
		t.Fatal("expected message on channel")
	}
}

func TestMessageHandlerUnknownClient(t *testing.T) {
	ctx := NewHandlerContext()

	body, _ := json.Marshal(messageRequest{ClientId: "does-not-exist", Message: "hi"})
	req := httptest.NewRequest(http.MethodPost, "/message", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	srw := NewStatusResponseWriter(rec)

	ctx.MessageHandler(srw, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestMessageHandlerBusy(t *testing.T) {
	ctx := NewHandlerContext()
	ctx.registry.Register("abc", &Client{RemoteAddr: "1.2.3.4:1"})

	body, _ := json.Marshal(messageRequest{ClientId: "abc", Message: "first"})
	req := httptest.NewRequest(http.MethodPost, "/message", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	srw := NewStatusResponseWriter(rec)
	ctx.MessageHandler(srw, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected first send to return 200, got %d", rec.Code)
	}

	body2, _ := json.Marshal(messageRequest{ClientId: "abc", Message: "second"})
	req2 := httptest.NewRequest(http.MethodPost, "/message", bytes.NewReader(body2))
	rec2 := httptest.NewRecorder()
	srw2 := NewStatusResponseWriter(rec2)
	ctx.MessageHandler(srw2, req2)

	if rec2.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rec2.Code)
	}
}

func TestMessageHandlerMalformedJSON(t *testing.T) {
	ctx := NewHandlerContext()

	req := httptest.NewRequest(http.MethodPost, "/message", bytes.NewReader([]byte("not json")))
	rec := httptest.NewRecorder()
	srw := NewStatusResponseWriter(rec)

	ctx.MessageHandler(srw, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestMessageHandlerWrongMethod(t *testing.T) {
	ctx := NewHandlerContext()

	req := httptest.NewRequest(http.MethodGet, "/message", nil)
	rec := httptest.NewRecorder()
	srw := NewStatusResponseWriter(rec)

	ctx.MessageHandler(srw, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./... -run TestMessageHandler -v`
Expected: FAIL / build error — `messageRequest`, `messageResponse`, `MessageHandler` undefined.

- [ ] **Step 3: Write the implementation**

In `main.go`, add after `StatusHandler`:

```go
type messageRequest struct {
	ClientId string `json:"clientId"`
	Message  string `json:"message"`
}

type messageResponse struct {
	Delivered bool   `json:"delivered"`
	Reason    string `json:"reason,omitempty"`
}

func (ctx *HandlerContext) MessageHandler(srw *StreamResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		srw.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var body messageRequest
	if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
		srw.WriteHeader(http.StatusBadRequest)
		return
	}

	result := ctx.registry.Send(body.ClientId, body.Message)

	srw.Header().Set("Content-Type", "application/json")

	var resp messageResponse
	switch result {
	case sendOK:
		resp = messageResponse{Delivered: true}
		srw.WriteHeader(http.StatusOK)
	case sendBusy:
		resp = messageResponse{Delivered: false, Reason: "busy"}
		srw.WriteHeader(http.StatusConflict)
	default:
		resp = messageResponse{Delivered: false, Reason: "unknown_client"}
		srw.WriteHeader(http.StatusNotFound)
	}

	_ = json.NewEncoder(srw).Encode(resp)
}
```

In `main()`, add the route next to the existing two:

```go
http.Handle("/status", adapt(ctx.StatusHandler))
http.Handle("/stream", adapt(ctx.StreamHandler))
http.Handle("/message", adapt(ctx.MessageHandler))
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./... -v`
Expected: PASS for all `TestMessageHandler*` tests and everything from Tasks 1-3.

- [ ] **Step 5: Manual smoke test against the real server**

```bash
go build && ./sse-server
```

In one terminal, open a stream and note the `clientId` from the `hello` event:

```bash
curl -N http://localhost:8080/stream
```

In another terminal, using the `clientId` just printed:

```bash
curl -X POST http://localhost:8080/message \
  -H "Content-Type: application/json" \
  -d '{"clientId":"<paste-id-here>","message":"Today is Monday, time is 4 pm"}'
```

Expected: `{"delivered":true}` from the POST, and shortly after, on the first terminal's stream:
```
id: custom-1
event: custom
data: Today is Monday, time is 4 pm
```
interleaved with the ongoing `message-N` events. Then POST again with a made-up clientId (e.g. `"nope"`) and confirm `404 {"delivered":false,"reason":"unknown_client"}`.

- [ ] **Step 6: Commit**

```bash
git add main.go message_handler_test.go
git commit -m "feat: add POST /message endpoint to inject custom SSE messages"
```

---

### Task 5: Documentation updates

**Files:**
- Modify: `README.md`
- Modify: `CLAUDE.md`

**Interfaces:**
- Consumes: the final `hello` event JSON shape and `POST /message` request/response shapes from Tasks 3-4. No new interfaces produced — this task is documentation only.

- [ ] **Step 1: Update the `/stream` section of README.md**

Find the block documenting the `hello` event (currently shows `id: hello` / `data: Hello, 192.168.65.1:35326!`) and replace it with the new JSON shape:

```
id: hello
data: {
data:   "message": "Hello, 192.168.65.1:35326!",
data:   "clientId": "a1b2c3d4e5f6a7b8"
data: }
```

Also update the `Last-Event-ID` and `count` example blocks later in the same file wherever they show the old plain-text `hello` line, to the same JSON shape (with a placeholder clientId consistent with that example's remote address).

- [ ] **Step 2: Add a new `## POST /message` section to README.md**

Add after the existing `#### count parameter` section and before `## Authentication`:

```markdown
### `POST /message`

The `POST /message` endpoint injects a custom message into one specific,
already-open `/stream` connection, identified by the `clientId` returned in
that connection's `hello` event. The message appears on the target stream
as a `custom` event, interleaved with the automatic `message-N` events —
it does not pause or replace them.

Request body:

```json
{
  "clientId": "a1b2c3d4e5f6a7b8",
  "message": "Today is Monday, time is 4 pm"
}
```

Responses:

- `200 {"delivered": true}` — message accepted, will appear on the target
  stream shortly.
- `404 {"delivered": false, "reason": "unknown_client"}` — no connection is
  registered under that `clientId` (never connected, or already
  disconnected).
- `409 {"delivered": false, "reason": "busy"}` — the client is connected but
  its previous custom message hasn't been delivered yet; retry.
- `400` — malformed request body.

On the `/stream` side, the injected message arrives as:

```
id: custom-1
event: custom
data: Today is Monday, time is 4 pm
```
```

- [ ] **Step 3: Update the Authentication section note**

In the `## Authentication` section, change the line describing which requests need the bearer token from applying to "all requests" — confirm it already says "all requests" (it does), so `POST /message` is already covered by that existing wording; no edit needed here beyond a quick read-check.

- [ ] **Step 4: Update CLAUDE.md's architecture notes**

In the `## Architecture` section's `main.go` bullet, add a mention of `ClientRegistry`, and add a new bullet for `registry.go`. Read the current file first to match its exact bullet style, then add a bullet reading approximately:

```markdown
- **registry.go** — `ClientRegistry` is the single source of truth for
  connected clients (replaces the old unguarded `clients` map), keyed by a
  `crypto/rand`-generated `clientId` (not `GenerateRandomString`, which is
  deterministic by design and unsuitable as a unique id). `Send` delivers a
  custom message onto a client's buffered channel (size 1) if connected and
  not already backed up, used by `POST /message`.
```

and update the existing `main.go` bullet to mention that `StreamHandler`'s loop is now a `select` over a ticker, the registry channel, and request-context cancellation, and that `POST /message` (`MessageHandler`) is the new third route.

- [ ] **Step 5: Commit**

```bash
git add README.md CLAUDE.md
git commit -m "docs: document POST /message and the new hello event shape"
```
