package orchestrator

import (
	"context"
	"errors"
	"testing"

	"lol-telemetry/internal/hooks"
	"lol-telemetry/internal/judge/payload"
	"lol-telemetry/internal/types"
	"lol-telemetry/pkg/riotclient"
)

type mockProvider struct {
	data riotclient.AllGameData
	err  error
}

func (m *mockProvider) GetGameData() (riotclient.AllGameData, error) {
	return m.data, m.err
}

type mockJudge struct {
	responses []types.JudgeResponse
	err       error
	calls     int
}

func (m *mockJudge) Evaluate(ctx context.Context, req types.JudgeRequest) (types.JudgeResponse, error) {
	m.calls++
	if m.calls <= len(m.responses) {
		return m.responses[m.calls-1], m.err
	}
	return types.JudgeResponse{}, m.err
}

func TestOrchestrator_Tick_FiresAtFiveMinuteMark(t *testing.T) {
	provider := &mockProvider{data: gameAt(300)}
	reg := hooks.NewRegistry()
	reg.Register(hooks.Periodic5MinHook{})
	j := &mockJudge{responses: []types.JudgeResponse{{Advice: "Recall and buy."}}}
	b := payload.NewBuilder()
	orch := NewOrchestrator(provider, reg, b, j)

	// First tick establishes baseline; no trigger yet.
	_, _ = orch.Tick(context.Background())

	provider.data = gameAt(600)
	resps, err := orch.Tick(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resps) != 1 {
		t.Fatalf("expected 1 response, got %d", len(resps))
	}
	if resps[0].Advice != "Recall and buy." {
		t.Errorf("advice = %s, want Recall and buy.", resps[0].Advice)
	}
	if j.calls != 1 {
		t.Errorf("judge calls = %d, want 1", j.calls)
	}
}

func TestOrchestrator_Tick_DoesNotDuplicate(t *testing.T) {
	provider := &mockProvider{data: gameAt(300)}
	reg := hooks.NewRegistry()
	reg.Register(hooks.Periodic5MinHook{})
	j := &mockJudge{responses: []types.JudgeResponse{{Advice: "Recall and buy."}}}
	b := payload.NewBuilder()
	orch := NewOrchestrator(provider, reg, b, j)

	_, _ = orch.Tick(context.Background())
	provider.data = gameAt(600)
	_, _ = orch.Tick(context.Background())

	provider.data = gameAt(610)
	resps, err := orch.Tick(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resps) != 0 {
		t.Fatalf("expected 0 responses, got %d", len(resps))
	}
	if j.calls != 1 {
		t.Errorf("judge calls = %d, want 1", j.calls)
	}
}

func TestOrchestrator_Tick_ResetsWhenGameEnds(t *testing.T) {
	provider := &mockProvider{data: gameAt(300)}
	reg := hooks.NewRegistry()
	reg.Register(hooks.Periodic5MinHook{})
	j := &mockJudge{responses: []types.JudgeResponse{{Advice: "Recall."}, {Advice: "Push."}}}
	b := payload.NewBuilder()
	orch := NewOrchestrator(provider, reg, b, j)

	_, _ = orch.Tick(context.Background())
	provider.data = gameAt(600)
	_, _ = orch.Tick(context.Background())

	provider.data = gameAt(0)
	_, _ = orch.Tick(context.Background())

	provider.data = gameAt(600)
	_, _ = orch.Tick(context.Background())
	provider.data = gameAt(900)
	resps, err := orch.Tick(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resps) != 1 {
		t.Fatalf("expected 1 response after reset, got %d", len(resps))
	}
	if j.calls != 2 {
		t.Errorf("judge calls = %d, want 2", j.calls)
	}
}

func TestOrchestrator_Tick_APIErrorResets(t *testing.T) {
	provider := &mockProvider{data: gameAt(300)}
	reg := hooks.NewRegistry()
	reg.Register(hooks.Periodic5MinHook{})
	j := &mockJudge{responses: []types.JudgeResponse{{Advice: "Recall."}, {Advice: "Push."}}}
	b := payload.NewBuilder()
	orch := NewOrchestrator(provider, reg, b, j)

	_, _ = orch.Tick(context.Background())
	provider.data = gameAt(600)
	_, _ = orch.Tick(context.Background())

	provider.err = errors.New("api down")
	_, err := orch.Tick(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}

	provider.err = nil
	provider.data = gameAt(600)
	_, _ = orch.Tick(context.Background())
	provider.data = gameAt(900)
	resps, err := orch.Tick(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resps) != 1 {
		t.Fatalf("expected 1 response after reset, got %d", len(resps))
	}
}

func gameAt(seconds float64) riotclient.AllGameData {
	return riotclient.AllGameData{
		ActivePlayer: riotclient.ActivePlayer{
			SummonerName: "ActivePlayer",
			CurrentGold:  1000,
			Level:        1,
		},
		AllPlayers: []riotclient.AllPlayer{
			{
				SummonerName: "ActivePlayer",
				ChampionName: "Annie",
				Position:     "MIDDLE",
				Team:         "ORDER",
				Scores:       riotclient.PlayerScores{},
			},
		},
		GameData: riotclient.GameData{GameMode: "CLASSIC", GameTime: seconds},
	}
}
