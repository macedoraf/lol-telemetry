package providers

import (
	"fmt"
	"strings"

	"lol-telemetry/internal/judge"
)

// New creates an LLM client from a Config, reading provider-specific env vars
// for DeepInfra and OpenAI. Supported providers: openrouter (default), deepinfra, openai.
func New(cfg Config) (judge.LLMClient, error) {
	provider := strings.ToLower(cfg.Provider)
	switch provider {
	case "", "openrouter":
		if cfg.OpenRouterKey == "" {
			return nil, fmt.Errorf("OPENROUTER_API_KEY is required for provider %s", provider)
		}
		return NewOpenRouterClientWithModel(cfg.OpenRouterKey, cfg.OpenRouterModel), nil
	case "deepinfra":
		return NewDeepInfraClientFromEnv()
	case "openai":
		return NewOpenAIClientFromEnv()
	default:
		return nil, fmt.Errorf("unknown judge provider: %s", cfg.Provider)
	}
}
