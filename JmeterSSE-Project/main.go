package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const defaultListenPort = "8080"
const messageIdPrefix = "message-"
const randomStringLength = 16

type Client struct {
	RemoteAddr  string    `json:"remote"`
	ConnectedAt time.Time `json:"connectedAt"`
	LastEventId int       `json:"lastEventId"`
	ClientId    string    `json:"clientId"`
}

type StreamResponseWriter struct {
	http.ResponseWriter
	controller *http.ResponseController
	statusCode int
}

func (srw *StreamResponseWriter) WriteHeader(statusCode int) {
	srw.statusCode = statusCode
	srw.ResponseWriter.WriteHeader(statusCode)
}

func (srw *StreamResponseWriter) WriteTypedEvent(id string, eventType string, data string) error {
	if id != "" {
		_, err := fmt.Fprintf(srw, "id: %s\n", id)
		if err != nil {
			return err
		}
	}

	if eventType != "" {
		_, err := fmt.Fprintf(srw, "event: %s\n", eventType)
		if err != nil {
			return err
		}
	}

	sc := bufio.NewScanner(strings.NewReader(data))
	for sc.Scan() {
		_, err := fmt.Fprintf(srw, "data: %s\n", sc.Text())
		if err != nil {
			return err
		}
	}
	if err := sc.Err(); err != nil {
		return err
	}
	_, err := fmt.Fprintf(srw, "\n")
	if err != nil {
		return err
	}

	return srw.controller.Flush()
}

func (srw *StreamResponseWriter) WriteEvent(id string, data string) error {
	return srw.WriteTypedEvent(id, "", data)
}

func NewStatusResponseWriter(writer http.ResponseWriter) *StreamResponseWriter {
	return &StreamResponseWriter{
		ResponseWriter: writer,
		controller:     http.NewResponseController(writer),
		statusCode:     http.StatusOK,
	}
}

type HandlerContext struct {
	registry *ClientRegistry
}

func NewHandlerContext() *HandlerContext {
	return &HandlerContext{registry: NewClientRegistry()}
}

type HandlerFunc func(srw *StreamResponseWriter, request *http.Request)

func (f HandlerFunc) ServeHTTP(srw *StreamResponseWriter, request *http.Request) {
	f(srw, request)
}

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

	if client.LastEventId >= limit {
		return
	}

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

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
			if client.LastEventId >= limit {
				return
			}
		case msg := <-ch:
			err := srw.WriteTypedEvent("", "custom", msg)
			if err != nil {
				return
			}
		case <-request.Context().Done():
			return
		}
	}
}

func (ctx *HandlerContext) StatusHandler(srw *StreamResponseWriter, request *http.Request) {
	body, err := json.Marshal(ctx.registry.Snapshot())
	if err != nil {
		srw.WriteHeader(http.StatusInternalServerError)
		return
	}

	srw.Header().Set("Content-Type", "application/json")
	_, _ = srw.Write(body)
}

type messageRequest struct {
	ClientId string `json:"clientId"`
	Message  string `json:"message"`
}

type messageResponse struct {
	Delivered bool   `json:"delivered"`
	Reason    string `json:"reason,omitempty"`
}

func (ctx *HandlerContext) MessageHandler(srw *StreamResponseWriter, request *http.Request) {
	srw.Header().Set("Access-Control-Allow-Origin", "*")
	srw.Header().Set("Access-Control-Allow-Headers", "*")
	srw.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")

	if request.Method == http.MethodOptions {
		srw.WriteHeader(http.StatusNoContent)
		return
	}

	if request.Method != http.MethodPost {
		srw.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var body messageRequest
	if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
		srw.WriteHeader(http.StatusBadRequest)
		return
	}

	if body.Message == "" {
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

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultListenPort
	}

	// Read the token from AUTH_TOKEN_FILE, falling back to AUTH_TOKEN value.
	auth := NewAllowAllAuthValidator()
	token := os.Getenv("AUTH_TOKEN")
	tokenFile := os.Getenv("AUTH_TOKEN_FILE")
	if tokenFile != "" {
		bytes, err := os.ReadFile(tokenFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading token from '%s': %v; ignoring.\n", tokenFile, err)
		} else if fileToken := strings.TrimSpace(string(bytes)); fileToken != "" {
			token = fileToken
		} else {
			fmt.Fprintf(os.Stderr, "Token file '%s' is empty; ignoring.\n", tokenFile)
		}
	}

	if token != "" {
		auth = NewTokenAuthValidator(token)
	}

	// Define the middleware stack
	authMiddleware := NewAuthMiddleware(auth)
	loggingMiddleware := NewLoggingMiddleware(os.Stdout)
	adapt := func(handler HandlerFunc) http.HandlerFunc {
		return AdaptHandler(handler, loggingMiddleware, authMiddleware)
	}

	ctx := NewHandlerContext()
	http.Handle("/status", adapt(ctx.StatusHandler))
	http.Handle("/stream", adapt(ctx.StreamHandler))
	http.Handle("/message", adapt(ctx.MessageHandler))

	fmt.Printf("Starting sse-server at :%s ...\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		panic(err)
	}
}
