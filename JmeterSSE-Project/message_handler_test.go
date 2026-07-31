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
		if msg.Data != "hi" {
			t.Fatalf("expected 'hi', got %q", msg.Data)
		}
		if msg.Type != "custom" {
			t.Fatalf("expected type 'custom', got %q", msg.Type)
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

	for i := 0; i < 4; i++ {
		if result := ctx.registry.Send("abc", streamEvent{Type: "custom", Data: "fill"}); result != sendOK {
			t.Fatalf("expected sendOK filling buffer slot %d, got %v", i, result)
		}
	}

	body, _ := json.Marshal(messageRequest{ClientId: "abc", Message: "overflow"})
	req := httptest.NewRequest(http.MethodPost, "/message", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	srw := NewStatusResponseWriter(rec)
	ctx.MessageHandler(srw, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rec.Code)
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

func TestMessageHandlerEmptyMessage(t *testing.T) {
	ctx := NewHandlerContext()
	ctx.registry.Register("abc", &Client{RemoteAddr: "1.2.3.4:1"})

	body, _ := json.Marshal(messageRequest{ClientId: "abc", Message: ""})
	req := httptest.NewRequest(http.MethodPost, "/message", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	srw := NewStatusResponseWriter(rec)

	ctx.MessageHandler(srw, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestMessageHandlerOptionsPreflight(t *testing.T) {
	ctx := NewHandlerContext()

	req := httptest.NewRequest(http.MethodOptions, "/message", nil)
	rec := httptest.NewRecorder()
	srw := NewStatusResponseWriter(rec)

	ctx.MessageHandler(srw, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("expected Access-Control-Allow-Origin: *, got %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); got != "*" {
		t.Fatalf("expected Access-Control-Allow-Headers: *, got %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got != "POST, OPTIONS" {
		t.Fatalf("expected Access-Control-Allow-Methods: POST, OPTIONS, got %q", got)
	}
}
