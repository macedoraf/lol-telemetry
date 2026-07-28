package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"lol-telemetry/internal/features"
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
	lastReq   types.JudgeRequest
}

func (m *mockJudge) Evaluate(ctx context.Context, req types.JudgeRequest) (types.JudgeResponse, error) {
	m.calls++
	m.lastReq = req
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
	orch := NewOrchestrator(client, reg, b, j, nil, nil)

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
	j := &mockJudge{responses: []types.JudgeResponse{{Advice: "Recall."}, {Advice: "Push."}}}
	b := payload.NewBuilder("en")
	orch := NewOrchestrator(client, reg, b, j, nil, nil)

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
	orch := NewOrchestrator(client, reg, b, j, nil, nil)

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
	orch := NewOrchestrator(client, reg, b, j, nil, nil)

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

func TestOrchestrator_Tick_PipelineAddsFeaturesAndObjectives(t *testing.T) {
	data := gameAt(600)
	data.Events.Events = append(data.Events.Events, riotclient.Event{
		EventID:    1,
		EventName:  "DragonKill",
		EventTime:  312,
		DragonType: "Infernal",
		Stolen:     "True",
		KillerName: "EnemyAhri",
	})
	data.AllPlayers = append(data.AllPlayers, riotclient.AllPlayer{
		SummonerName: "EnemyAhri",
		ChampionName: "Ahri",
		Position:     "MIDDLE",
		Team:         "CHAOS",
		Scores:       riotclient.PlayerScores{Kills: 3, Deaths: 1, Assists: 2, CreepScore: 95},
		Items: []riotclient.Item{
			{ItemID: 3802, Consumable: false, Price: 1100},
		},
	})

	var err error
	client := newTestClient(&data, &err)
	reg := hooks.NewRegistry()
	reg.Register(&hooks.Periodic5MinHook{})
	j := &mockJudge{responses: []types.JudgeResponse{{Advice: "Push."}}}
	b := payload.NewBuilder("en")
	tracker := features.NewTracker()
	tracker.Add(data)
	orch := NewOrchestrator(client, reg, b, j, features.NewPipeline(), tracker)

	// Baseline at 300, then fire at 600.
	data.GameData.GameTime = 300
	_, _ = orch.Tick(context.Background())
	data.GameData.GameTime = 600
	resps, err := orch.Tick(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resps) != 1 {
		t.Fatalf("expected 1 response, got %d", len(resps))
	}
	if j.lastReq.Features == nil {
		t.Fatal("expected Features to be populated")
	}
	if j.lastReq.Objectives.Chaos.Dragons != 1 {
		t.Errorf("legacy Objectives Chaos Dragons = %d, want 1", j.lastReq.Objectives.Chaos.Dragons)
	}
	if j.lastReq.Objectives.Order.Dragons != 0 {
		t.Errorf("legacy Objectives Order Dragons = %d, want 0", j.lastReq.Objectives.Order.Dragons)
	}
	if j.lastReq.Features.Enemy.Objectives.Steals != 1 {
		t.Errorf("feature enemy steals = %d, want 1", j.lastReq.Features.Enemy.Objectives.Steals)
	}
}

func TestOrchestrator_Tick_NoPipeline_KeepsFeaturesNil(t *testing.T) {
	data := gameAt(600)
	var err error
	client := newTestClient(&data, &err)
	reg := hooks.NewRegistry()
	reg.Register(&hooks.Periodic5MinHook{})
	j := &mockJudge{responses: []types.JudgeResponse{{Advice: "Recall."}}}
	b := payload.NewBuilder("en")
	orch := NewOrchestrator(client, reg, b, j, nil, nil)

	// Baseline at 300, then fire at 600.
	data.GameData.GameTime = 300
	_, _ = orch.Tick(context.Background())
	data.GameData.GameTime = 600
	_, _ = orch.Tick(context.Background())

	if j.lastReq.Features != nil {
		t.Errorf("expected nil Features when pipeline disabled, got %+v", j.lastReq.Features)
	}
	if j.lastReq.Objectives.Order.Towers != 0 || j.lastReq.Objectives.Chaos.Towers != 0 {
		t.Errorf("expected zero legacy Objectives when pipeline disabled, got %+v", j.lastReq.Objectives)
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
		Events:   riotclient.Events{Events: []riotclient.Event{{EventID: 0, EventName: "GameStart", EventTime: 0}}},
		GameData: riotclient.GameData{GameMode: "CLASSIC", GameTime: seconds},
	}
}
