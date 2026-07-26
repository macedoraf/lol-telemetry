// Package config centralizes environment-based configuration for the CLI.
package config

import (
	"fmt"
	"os"
)

const defaultModel = "openai/gpt-4o-mini"

// EnvConfig holds the Judge configuration loaded from environment variables.
type EnvConfig struct {
	APIKey string
	Model  string
}

// LoadEnvConfig reads OPENROUTER_API_KEY and OPENROUTER_MODEL from the environment.
func LoadEnvConfig() EnvConfig {
	model := os.Getenv("OPENROUTER_MODEL")
	if model == "" {
		model = defaultModel
	}
	return EnvConfig{
		APIKey: os.Getenv("OPENROUTER_API_KEY"),
		Model:  model,
	}
}

// Enabled reports whether the Judge is configured to run.
func (c EnvConfig) Enabled() bool {
	return c.APIKey != ""
}

// MaskedKey returns a masked version of the API key showing only the last 4 characters.
// If the key is empty or too short, a placeholder is returned.
func (c EnvConfig) MaskedKey() string {
	if c.APIKey == "" {
		return "(não configurada)"
	}
	if len(c.APIKey) <= 4 {
		return "****"
	}
	return "..." + c.APIKey[len(c.APIKey)-4:]
}

// String returns a human-readable summary of the configuration.
func (c EnvConfig) String() string {
	keyStatus := c.MaskedKey()
	if !c.Enabled() {
		return fmt.Sprintf("OPENROUTER_API_KEY: %s\nOPENROUTER_MODEL: %s\nStatus: desativado", keyStatus, c.Model)
	}
	return fmt.Sprintf("OPENROUTER_API_KEY: %s\nOPENROUTER_MODEL: %s\nStatus: ativado", keyStatus, c.Model)
}
