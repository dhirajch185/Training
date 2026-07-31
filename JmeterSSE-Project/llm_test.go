package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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
	if want := "who has a birthday today?"; !json.Valid(b) || !strings.Contains(string(b), want) {
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
