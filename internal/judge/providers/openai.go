package providers

import (
	"fmt"
	"os"
)

// NewOpenAIClientFromEnv creates an OpenAI client from environment variables.
// Required: OPENAI_API_KEY.
// Optional: OPENAI_BASE_URL (default https://api.openai.com/v1), OPENAI_MODEL.
func NewOpenAIClientFromEnv() (*Client, error) {
	baseURL := os.Getenv("OPENAI_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY is required for provider openai")
	}
	model := os.Getenv("OPENAI_MODEL")
	if model == "" {
		model = "gpt-4o-mini"
	}
	return NewClient(baseURL, apiKey, model), nil
}
