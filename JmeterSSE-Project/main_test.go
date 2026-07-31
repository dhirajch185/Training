package main

import (
	"bufio"
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

func TestWriteTypedEventNoID(t *testing.T) {
	rec := httptest.NewRecorder()
	srw := NewStatusResponseWriter(rec)

	if err := srw.WriteTypedEvent("", "custom", "hi"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body := rec.Body.String()
	if strings.Contains(body, "id:") {
		t.Fatalf("expected no id: line, got body: %q", body)
	}
	if !strings.Contains(body, "event: custom\n") {
		t.Fatalf("expected event line, got body: %q", body)
	}
	if !strings.Contains(body, "data: hi\n") {
		t.Fatalf("expected data line, got body: %q", body)
	}
}

func TestWriteTypedEventOversizedLineReturnsError(t *testing.T) {
	rec := httptest.NewRecorder()
	srw := NewStatusResponseWriter(rec)

	oversized := strings.Repeat("a", bufio.MaxScanTokenSize+1)
	err := srw.WriteTypedEvent("", "custom", oversized)
	if err == nil {
		t.Fatal("expected non-nil error for oversized data line, got nil")
	}
}
