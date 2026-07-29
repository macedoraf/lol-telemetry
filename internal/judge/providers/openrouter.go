package providers

const (
	// OpenRouterEndpoint is the OpenRouter API base URL.
	OpenRouterEndpoint = "https://openrouter.ai/api/v1"
	// OpenRouterDefaultModel is the model used when none is provided.
	OpenRouterDefaultModel = "openai/gpt-4o-mini"
)

// NewOpenRouterClient creates an OpenRouter client with the default model.
func NewOpenRouterClient(apiKey string) *Client {
	return NewOpenRouterClientWithModel(apiKey, OpenRouterDefaultModel)
}

// NewOpenRouterClientWithModel creates an OpenRouter client with the given model.
func NewOpenRouterClientWithModel(apiKey, model string) *Client {
	if model == "" {
		model = OpenRouterDefaultModel
	}
	return NewClient(OpenRouterEndpoint, apiKey, model)
}
