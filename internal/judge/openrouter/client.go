// Package openrouter provides a minimal HTTP client for the OpenRouter API.
package openrouter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// DefaultEndpoint is the OpenRouter chat completions endpoint.
const DefaultEndpoint = "https://openrouter.ai/api/v1/chat/completions"

// Client is a thin LLM client using the OpenRouter / OpenAI format.
type Client struct {
	APIKey   string
	Endpoint string
	Model    string
	HTTP     *http.Client
}

// NewClient creates an OpenRouter client with the given API key.
func NewClient(apiKey string) *Client {
	return &Client{
		APIKey:   apiKey,
		Endpoint: DefaultEndpoint,
		Model:    "openai/gpt-4o-mini",
		HTTP: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// Complete sends a prompt and returns the LLM text response.
func (c *Client) Complete(ctx context.Context, systemPrompt, prompt string) (string, error) {
	body := map[string]any{
		"model": c.Model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": prompt},
		},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("unexpected status %d: %s", resp.StatusCode, body)
	}

	var result completionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("empty choices in response")
	}
	return result.Choices[0].Message.Content, nil
}

type completionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}
