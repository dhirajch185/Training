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

type streamEvent struct {
	Type string // "custom", "llm-response", "llm-error"
	Data string
}

type registeredClient struct {
	client *Client
	ch     chan streamEvent
}

type ClientRegistry struct {
	mu      sync.Mutex
	clients map[string]*registeredClient
}

func NewClientRegistry() *ClientRegistry {
	return &ClientRegistry{clients: make(map[string]*registeredClient)}
}

func (r *ClientRegistry) Register(id string, client *Client) chan streamEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	ch := make(chan streamEvent, 4)
	r.clients[id] = &registeredClient{client: client, ch: ch}
	return ch
}

func (r *ClientRegistry) Unregister(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.clients, id)
}

func (r *ClientRegistry) Send(id string, ev streamEvent) sendResult {
	r.mu.Lock()
	defer r.mu.Unlock()
	rc, ok := r.clients[id]
	if !ok {
		return sendUnknownClient
	}
	select {
	case rc.ch <- ev:
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
