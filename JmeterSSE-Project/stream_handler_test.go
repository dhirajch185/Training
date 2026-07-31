package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

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

	rec := httptest.NewRecorder()
	srw := NewStatusResponseWriter(rec)

	done := make(chan struct{})
	go func() {
		defer close(done)
		handlerCtx.StreamHandler(srw, req)
	}()

	var clientId string
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		body := rec.Body.String()
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
		t.Fatalf("failed to extract clientId from hello event, body so far: %q", rec.Body.String())
	}

	result := handlerCtx.registry.Send(clientId, "hi there")
	if result != sendOK {
		t.Fatalf("expected sendOK, got %v", result)
	}

	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(rec.Body.String(), "event: custom\ndata: hi there\n") {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	cancel()
	<-done

	body := rec.Body.String()
	if !strings.Contains(body, "event: custom\ndata: hi there\n") {
		t.Fatalf("expected custom event with data, got body: %q", body)
	}
	if strings.Contains(body, "id: custom") {
		t.Fatalf("expected no id: line for custom event, got body: %q", body)
	}
}
