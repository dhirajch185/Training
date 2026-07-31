package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// syncRecorder wraps httptest.ResponseRecorder with a mutex so it can be
// safely written to from a handler goroutine while the test goroutine polls
// its body concurrently (httptest.ResponseRecorder's *bytes.Buffer is not
// goroutine-safe on its own).
type syncRecorder struct {
	mu  sync.Mutex
	rec *httptest.ResponseRecorder
}

func newSyncRecorder() *syncRecorder {
	return &syncRecorder{rec: httptest.NewRecorder()}
}

func (s *syncRecorder) Header() http.Header {
	return s.rec.Header()
}

func (s *syncRecorder) Write(b []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.rec.Write(b)
}

func (s *syncRecorder) WriteHeader(code int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rec.WriteHeader(code)
}

func (s *syncRecorder) Flush() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rec.Flush()
}

func (s *syncRecorder) Body() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.rec.Body.String()
}

// waitForClientId polls the recorder body for the hello event's clientId,
// failing the test if it doesn't show up within 2 seconds.
func waitForClientId(t *testing.T, rec *syncRecorder) string {
	t.Helper()

	var clientId string
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		body := rec.Body()
		if idx := strings.Index(body, "\"clientId\": \""); idx != -1 {
			rest := body[idx+len("\"clientId\": \""):]
			end := strings.Index(rest, "\"")
			if end != -1 {
				clientId = rest[:end]
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
	}

	if clientId == "" {
		t.Fatalf("failed to extract clientId from hello event, body so far: %q", rec.Body())
	}

	return clientId
}

func TestStreamHandlerHelloEventCarriesClientId(t *testing.T) {
	ctx := NewHandlerContext()

	req := httptest.NewRequest(http.MethodGet, "/stream?count=0", nil)
	rec := httptest.NewRecorder()
	srw := NewStatusResponseWriter(rec)

	ctx.StreamHandler(srw, req)

	body := rec.Body.String()
	if !strings.Contains(body, "id: hello") {
		t.Fatalf("expected hello event, got body: %q", body)
	}
	if !strings.Contains(body, "\"clientId\"") {
		t.Fatalf("expected hello event data to contain clientId, got body: %q", body)
	}
}

func TestStreamHandlerCustomMessageHasNoIdLine(t *testing.T) {
	handlerCtx := NewHandlerContext()

	req := httptest.NewRequest(http.MethodGet, "/stream", nil)
	reqCtx, cancel := context.WithCancel(req.Context())
	req = req.WithContext(reqCtx)

	rec := newSyncRecorder()
	srw := NewStatusResponseWriter(rec)

	done := make(chan struct{})
	go func() {
		defer close(done)
		handlerCtx.StreamHandler(srw, req)
	}()

	clientId := waitForClientId(t, rec)

	result := handlerCtx.registry.Send(clientId, streamEvent{Type: "custom", Data: "hi there"})
	if result != sendOK {
		t.Fatalf("expected sendOK, got %v", result)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(rec.Body(), "event: custom\ndata: hi there\n") {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	cancel()
	<-done

	body := rec.Body()
	if !strings.Contains(body, "event: custom\ndata: hi there\n") {
		t.Fatalf("expected custom event with data, got body: %q", body)
	}
	if strings.Contains(body, "id: custom") {
		t.Fatalf("expected no id: line for custom event, got body: %q", body)
	}
}

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

func TestMessageTriggersLLMResponseOnStream(t *testing.T) {
	handlerCtx := NewHandlerContext()
	handlerCtx.llm = NewMockProvider(10 * time.Millisecond)

	req := httptest.NewRequest(http.MethodGet, "/stream?quiet=1", nil)
	reqCtx, cancel := context.WithCancel(req.Context())
	req = req.WithContext(reqCtx)
	rec := newSyncRecorder()
	srw := NewStatusResponseWriter(rec)

	done := make(chan struct{})
	go func() {
		handlerCtx.StreamHandler(srw, req)
		close(done)
	}()

	clientId := waitForClientId(t, rec)

	body, _ := json.Marshal(messageRequest{ClientId: clientId, Message: "ping"})
	postReq := httptest.NewRequest(http.MethodPost, "/message", bytes.NewReader(body))
	postRec := httptest.NewRecorder()
	handlerCtx.MessageHandler(NewStatusResponseWriter(postRec), postReq)
	if postRec.Code != http.StatusOK {
		t.Fatalf("expected 200 from POST, got %d", postRec.Code)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(rec.Body(), "event: llm-response\n") {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	cancel()
	<-done

	streamBody := rec.Body()
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
