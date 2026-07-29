package providers

import (
	"testing"
)

func TestNew_DefaultsToOpenRouter(t *testing.T) {
	_, err := New(Config{Provider: "", OpenRouterKey: "key"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNew_OpenRouterMissingKey(t *testing.T) {
	_, err := New(Config{Provider: "openrouter"})
	if err == nil {
		t.Fatal("expected error when OPENROUTER_API_KEY is missing")
	}
}

func TestNew_UnknownProvider(t *testing.T) {
	_, err := New(Config{Provider: "unknown"})
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
}

func TestNew_OpenAIRequiresKey(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	_, err := New(Config{Provider: "openai"})
	if err == nil {
		t.Fatal("expected error when OPENAI_API_KEY is missing")
	}
}

func TestNew_DeepInfraRequiresKey(t *testing.T) {
	t.Setenv("DEEPINFRA_API_KEY", "")
	_, err := New(Config{Provider: "deepinfra"})
	if err == nil {
		t.Fatal("expected error when DEEPINFRA_API_KEY is missing")
	}
}

func TestNew_DeepInfraCustomBaseURL(t *testing.T) {
	t.Setenv("DEEPINFRA_BASE_URL", "http://localhost:9999/v1/openai")
	t.Setenv("DEEPINFRA_API_KEY", "key")
	t.Setenv("DEEPINFRA_MODEL", "custom-model")
	c, err := New(Config{Provider: "deepinfra"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	client := c.(*Client)
	if client.BaseURL != "http://localhost:9999/v1/openai" {
		t.Errorf("BaseURL = %q, want custom endpoint", client.BaseURL)
	}
	if client.Model != "custom-model" {
		t.Errorf("Model = %q, want custom-model", client.Model)
	}
}
