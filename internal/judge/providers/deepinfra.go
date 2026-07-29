package providers

import (
	"fmt"
	"os"
)

const (
	// DeepInfraDefaultEndpoint is the DeepInfra OpenAI-compatible base URL.
	DeepInfraDefaultEndpoint = "https://api.deepinfra.com/v1/openai"
	// DeepInfraDefaultModel is a cheap, capable default model on DeepInfra.
	DeepInfraDefaultModel = "deepseek-ai/DeepSeek-V3"
)

// NewDeepInfraClientFromEnv creates a DeepInfra client from environment variables.
// Required: DEEPINFRA_API_KEY.
// Optional: DEEPINFRA_BASE_URL, DEEPINFRA_MODEL.
func NewDeepInfraClientFromEnv() (*Client, error) {
	baseURL := os.Getenv("DEEPINFRA_BASE_URL")
	if baseURL == "" {
		baseURL = DeepInfraDefaultEndpoint
	}
	apiKey := os.Getenv("DEEPINFRA_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("DEEPINFRA_API_KEY is required for provider deepinfra")
	}
	model := os.Getenv("DEEPINFRA_MODEL")
	if model == "" {
		model = DeepInfraDefaultModel
	}
	return NewClient(baseURL, apiKey, model), nil
}
