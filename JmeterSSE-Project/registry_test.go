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
