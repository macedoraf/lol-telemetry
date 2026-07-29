package providers

// Config selects and parametrizes an LLM provider without depending on the
// daemon's configuration package.
type Config struct {
	Provider string
	// OpenRouter credentials.
	OpenRouterKey   string
	OpenRouterModel string
}
