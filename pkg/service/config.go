// Package service provides the background daemon and WebSocket API for lol-telemetry.
package service

import (
	"os"
	"time"
)

// LoadDaemonConfigFromEnv reads daemon configuration from environment variables.
// Defaults:
//   - port: 8080
//   - poll interval: 1s
//   - judge: enabled if OPENROUTER_API_KEY is set
//   - LoL base URL: https://127.0.0.1:2999/liveclientdata
func LoadDaemonConfigFromEnv() DaemonConfig {
	port := os.Getenv("LOL_DAEMON_PORT")
	if port == "" {
		port = "8080"
	}

	baseURL := os.Getenv("LOL_BASE_URL")
	if baseURL == "" {
		baseURL = "https://127.0.0.1:2999/liveclientdata"
	}

	interval := 1 * time.Second
	if v := os.Getenv("LOL_POLL_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			interval = d
		}
	}

	provider := os.Getenv("JUDGE_PROVIDER")
	if provider == "" {
		provider = "openrouter"
	}

	apiKey := os.Getenv("OPENROUTER_API_KEY")
	model := os.Getenv("OPENROUTER_MODEL")
	if model == "" {
		model = "openai/gpt-4o-mini"
	}

	judgeEnabled := os.Getenv("JUDGE_ENABLED") != "false"
	if provider == "openrouter" && apiKey == "" {
		judgeEnabled = false
	}

	debug := os.Getenv("LOL_DEBUG") == "true" || os.Getenv("LOL_DEBUG") == "1"

	judgeLanguage := os.Getenv("JUDGE_LANGUAGE")
	if judgeLanguage == "" {
		judgeLanguage = "en"
	}

	recordEnabled := os.Getenv("LOL_RECORD_ENABLED") == "true" || os.Getenv("LOL_RECORD_ENABLED") == "1"
	recordingsDir := os.Getenv("LOL_RECORDINGS_DIR")
	if recordingsDir == "" {
		recordingsDir = "./recordings"
	}

	featuresEnabled := os.Getenv("LOL_FEATURES_ENABLED") == "true" || os.Getenv("LOL_FEATURES_ENABLED") == "1"

	return DaemonConfig{
		Port:             port,
		BaseURL:          baseURL,
		PollInterval:     interval,
		JudgeEnabled:     judgeEnabled,
		JudgeProvider:    provider,
		OpenRouterKey:    apiKey,
		OpenRouterModel:  model,
		Debug:            debug,
		JudgeLanguage:    judgeLanguage,
		RecordEnabled:    recordEnabled,
		RecordingsDir:    recordingsDir,
		FeaturesEnabled:  featuresEnabled,
	}
}
