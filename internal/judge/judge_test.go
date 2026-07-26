package judge

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"lol-telemetry/internal/types"
)

type mockLLM struct {
	calledSystem string
	calledPrompt string
	response     string
	err          error
}

func (m *mockLLM) Complete(ctx context.Context, systemPrompt, prompt string) (string, error) {
	m.calledSystem = systemPrompt
	m.calledPrompt = prompt
	return m.response, m.err
}

func TestJudge_Evaluate(t *testing.T) {
	mock := &mockLLM{response: `{"advice":"Push mid and secure dragon.","reasoning":"Dragon up at 10:00, mid priority."}`}
	j := NewJudge(mock)

	req := types.JudgeRequest{
		GameMinute:   10,
		Question:     "What should I do now?",
		SystemPrompt: "You are a coach.",
		Matchup: types.LaneMatchup{
			Player: types.PlayerSnapshot{SummonerName: "ActivePlayer"},
		},
	}

	resp, err := j.Evaluate(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Advice != "Push mid and secure dragon." {
		t.Errorf("Advice = %s, want Push mid and secure dragon.", resp.Advice)
	}
	if resp.Reasoning != "Dragon up at 10:00, mid priority." {
		t.Errorf("Reasoning = %s, want Dragon up at 10:00, mid priority.", resp.Reasoning)
	}
	if mock.calledSystem != "You are a coach." {
		t.Errorf("system prompt = %s, want You are a coach.", mock.calledSystem)
	}
	if !strings.Contains(mock.calledPrompt, "What should I do now?") {
		t.Errorf("prompt does not contain question: %s", mock.calledPrompt)
	}
	if !strings.Contains(mock.calledPrompt, "ActivePlayer") {
		t.Errorf("prompt does not contain player: %s", mock.calledPrompt)
	}
}

func TestJudge_Evaluate_TruncatesLongResponse(t *testing.T) {
	longAdvice := strings.Repeat("a", 200)
	longReasoning := strings.Repeat("b", 400)
	mock := &mockLLM{response: fmt.Sprintf(`{"advice":"%s","reasoning":"%s"}`, longAdvice, longReasoning)}
	j := NewJudge(mock)

	req := types.JudgeRequest{Question: "q"}
	resp, err := j.Evaluate(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Advice) != 140 {
		t.Errorf("advice length = %d, want %d", len(resp.Advice), 140)
	}
	if len(resp.Reasoning) != 300 {
		t.Errorf("reasoning length = %d, want %d", len(resp.Reasoning), 300)
	}
}
