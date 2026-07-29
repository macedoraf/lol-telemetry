// Package openrouter provides a minimal HTTP client for the OpenRouter API.
// It is now a thin wrapper around the generic OpenAI-compatible provider.
package openrouter

import (
	"context"
	"net/http"
	"time"

	"lol-telemetry/internal/judge/providers"
)

// DefaultEndpoint is the OpenRouter API base URL.
const DefaultEndpoint = providers.OpenRouterEndpoint

// DefaultModel is the model used when none is provided.
const DefaultModel = providers.OpenRouterDefaultModel

// Client is a thin LLM client using the OpenRouter / OpenAI format.
type Client struct {
	APIKey   string
	Endpoint string
	Model    string
	HTTP     *http.Client

	provider *providers.Client
}

// NewClient creates an OpenRouter client with the given API key and the default model.
func NewClient(apiKey string) *Client {
	return NewClientWithModel(apiKey, DefaultModel)
}

// NewClientWithModel creates an OpenRouter client with the given API key and model.
func NewClientWithModel(apiKey, model string) *Client {
	if model == "" {
		model = DefaultModel
	}
	c := &Client{
		APIKey:   apiKey,
		Endpoint: DefaultEndpoint,
		Model:    model,
		HTTP:     &http.Client{Timeout: 15 * time.Second},
	}
	c.provider = providers.NewClient(c.Endpoint, apiKey, model)
	c.provider.HTTP = c.HTTP
	return c
}

// Complete sends a prompt and returns the LLM text response.
func (c *Client) Complete(ctx context.Context, systemPrompt, prompt string) (string, error) {
	c.provider.BaseURL = c.Endpoint
	c.provider.APIKey = c.APIKey
	c.provider.Model = c.Model
	c.provider.HTTP = c.HTTP
	return c.provider.Complete(ctx, systemPrompt, prompt)
}
