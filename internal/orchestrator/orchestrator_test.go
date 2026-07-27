package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"lol-telemetry/internal/hooks"
	"lol-telemetry/internal/judge/payload"
	"lol-telemetry/internal/types"
	"lol-telemetry/pkg/riotclient"
)

func newTestClient(data *riotclient.AllGameData, err *error) *riotclient.Client {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if *err != nil {
			http.Error(w, (*err).Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(*data)
	}))
	return riotclient.NewClientWithURL(server.URL)
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
	data := gameAt(300)
	var err error
	client := newTestClient(&data, &err)
	reg := hooks.NewRegistry()
	reg.Register(&hooks.Periodic5MinHook{})
	j := &mockJudge{responses: []types.JudgeResponse{{Advice: "Recall and buy."}}}
	b := payload.NewBuilder("en")
	orch := NewOrchestrator(client, reg, b, j)

	// First tick establishes baseline; no trigger yet.
	_, _ = orch.Tick(context.Background())

	data = gameAt(600)
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
	if resps[0].GameTime != 600 {
		t.Errorf("gameTime = %f, want 600", resps[0].GameTime)
	}
	if resps[0].Question == "" {
		t.Errorf("question is empty")
	}
	if j.calls != 1 {
		t.Errorf("judge calls = %d, want 1", j.calls)
	}
}

func TestOrchestrator_Tick_DoesNotDuplicate(t *testing.T) {
	data := gameAt(300)
	var err error
	client := newTestClient(&data, &err)
	reg := hooks.NewRegistry()
	reg.Register(&hooks.Periodic5MinHook{})
	j := &mockJudge{responses: []types.JudgeResponse{{Advice: "Recall and buy."}}}
	b := payload.NewBuilder("en")
	orch := NewOrchestrator(client, reg, b, j)

	_, _ = orch.Tick(context.Background())
	data = gameAt(600)
	_, _ = orch.Tick(context.Background())

	data = gameAt(610)
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
	data := gameAt(300)
	var err error
	client := newTestClient(&data, &err)
	reg := hooks.NewRegistry()
	reg.Register(&hooks.Periodic5MinHook{})
	j := &mockJudge{responses: []types.JudgeResponse{{Advice: "Recall."}, {Advice: "Push."}}}
	b := payload.NewBuilder("en")
	orch := NewOrchestrator(client, reg, b, j)

	_, _ = orch.Tick(context.Background())
	data = gameAt(600)
	_, _ = orch.Tick(context.Background())

	data = gameAt(0)
	_, _ = orch.Tick(context.Background())

	data = gameAt(600)
	_, _ = orch.Tick(context.Background())
	data = gameAt(900)
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
	data := gameAt(300)
	var err error
	client := newTestClient(&data, &err)
	reg := hooks.NewRegistry()
	reg.Register(&hooks.Periodic5MinHook{})
	j := &mockJudge{responses: []types.JudgeResponse{{Advice: "Recall."}, {Advice: "Push."}}}
	b := payload.NewBuilder("en")
	orch := NewOrchestrator(client, reg, b, j)

	_, _ = orch.Tick(context.Background())
	data = gameAt(600)
	_, _ = orch.Tick(context.Background())

	err = errors.New("api down")
	_, errOut := orch.Tick(context.Background())
	if errOut == nil {
		t.Fatal("expected error")
	}

	err = nil
	data = gameAt(600)
	_, _ = orch.Tick(context.Background())
	data = gameAt(900)
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
