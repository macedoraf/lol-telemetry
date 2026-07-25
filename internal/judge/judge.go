// Package judge orchestrates the LLM evaluation for game hooks.
package judge

import (
	"context"
	"encoding/json"
	"fmt"

	"lol-telemetry/internal/types"
)

const maxAdviceLength = 140

// LLMClient abstracts the LLM provider call.
type LLMClient interface {
	Complete(ctx context.Context, systemPrompt, prompt string) (string, error)
}

// Judge turns a JudgeRequest into actionable advice via an LLM.
type Judge struct {
	client LLMClient
}

// NewJudge creates a Judge wired to the given LLM client.
func NewJudge(client LLMClient) *Judge {
	return &Judge{client: client}
}

// Evaluate sends the request to the LLM and returns a concise response.
func (j *Judge) Evaluate(ctx context.Context, req types.JudgeRequest) (types.JudgeResponse, error) {
	prompt, err := buildPrompt(req)
	if err != nil {
		return types.JudgeResponse{}, fmt.Errorf("build prompt: %w", err)
	}

	advice, err := j.client.Complete(ctx, req.SystemPrompt, prompt)
	if err != nil {
		return types.JudgeResponse{}, fmt.Errorf("llm completion: %w", err)
	}

	return types.JudgeResponse{Advice: truncate(advice, maxAdviceLength)}, nil
}

func buildPrompt(req types.JudgeRequest) (string, error) {
	payload, err := json.MarshalIndent(req, "", "  ")
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Game minute: %d\n\n%s\n\nQuestion: %s", req.GameMinute, payload, req.Question), nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
