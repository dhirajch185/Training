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
