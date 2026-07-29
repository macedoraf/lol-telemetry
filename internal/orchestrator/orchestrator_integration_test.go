package orchestrator

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"lol-telemetry/internal/hooks"
	"lol-telemetry/internal/judge/payload"
	"lol-telemetry/internal/types"
	"lol-telemetry/pkg/riotclient"
)

// TestIntegration_EndToEnd_FiresAtFiveMinuteMark validates the full pipeline
// from a mocked Live Client Data API to the Judge via the Orchestrator.
func TestIntegration_EndToEnd_FiresAtFiveMinuteMark(t *testing.T) {
	call := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/liveclientdata/allgamedata" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		data := fullGameDataSnapshot()
		if call == 0 {
			data.GameData.GameTime = 300 // baseline
		} else {
			data.GameData.GameTime = 600 // first trigger
		}
		call++
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(data)
	}))
	defer server.Close()

	client := riotclient.NewClient()
	client.BaseURL = server.URL + "/liveclientdata"

	reg := hooks.NewRegistry()
	reg.Register(&hooks.Periodic5MinHook{})

	judge := &mockJudge{responses: []types.JudgeResponse{{Advice: "Secure dragon and push mid."}}}
	builder := payload.NewBuilder("en")
	orch := NewOrchestrator(client, reg, builder, judge, nil, nil)

	_, _ = orch.Tick(context.Background())
	resps, err := orch.Tick(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resps) != 1 {
		t.Fatalf("expected 1 response, got %d", len(resps))
	}
	if resps[0].Advice != "Secure dragon and push mid." {
		t.Errorf("advice = %s, want Secure dragon and push mid.", resps[0].Advice)
	}
	if judge.calls != 1 {
		t.Errorf("judge calls = %d, want 1", judge.calls)
	}
}

// TestIntegration_EndToEnd_LateStartFiresAtNextMark validates that the hook
// ignores past marks when the system starts mid-match.
func TestIntegration_EndToEnd_LateStartFiresAtNextMark(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := fullGameDataSnapshot()
		data.GameData.GameTime = 610 // 10:10 — first future mark is 600 already passed, next is 900
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(data)
	}))
	defer server.Close()

	client := riotclient.NewClient()
	client.BaseURL = server.URL + "/liveclientdata"

	reg := hooks.NewRegistry()
	reg.Register(&hooks.Periodic5MinHook{})

	judge := &mockJudge{responses: []types.JudgeResponse{{Advice: "Late start advice."}}}
	builder := payload.NewBuilder("en")
	orch := NewOrchestrator(client, reg, builder, judge, nil, nil)

	// First tick at 10:10 should NOT fire because 10:00 mark is in the past
	// relative to start and no previous mark was recorded.
	resps, err := orch.Tick(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resps) != 0 {
		t.Fatalf("expected 0 responses for late start at 10:10, got %d", len(resps))
	}
}

// TestIntegration_EndToEnd_APIErrorDoesNotCrash validates resilience when the
// Live Client Data API returns an error.
func TestIntegration_EndToEnd_APIErrorDoesNotCrash(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("api unavailable"))
	}))
	defer server.Close()

	client := riotclient.NewClient()
	client.BaseURL = server.URL + "/liveclientdata"

	reg := hooks.NewRegistry()
	reg.Register(&hooks.Periodic5MinHook{})

	judge := &mockJudge{}
	builder := payload.NewBuilder("en")
	orch := NewOrchestrator(client, reg, builder, judge, nil, nil)

	_, err := orch.Tick(context.Background())
	if err == nil {
		t.Fatal("expected error when Live Client API fails")
	}
	if judge.calls != 0 {
		t.Errorf("judge calls = %d, want 0", judge.calls)
	}
}

func fullGameDataSnapshot() riotclient.AllGameData {
	return riotclient.AllGameData{
		ActivePlayer: riotclient.ActivePlayer{
			SummonerName: "ActivePlayer",
			CurrentGold:  2500,
			Level:        6,
		},
		AllPlayers: []riotclient.AllPlayer{
			{
				SummonerName: "ActivePlayer",
				ChampionName: "Annie",
				Level:        6,
				Position:     "MIDDLE",
				Team:         "ORDER",
				Scores: riotclient.PlayerScores{
					Kills:      1,
					Deaths:     0,
					Assists:    2,
					CreepScore: 60,
				},
				Items: []riotclient.Item{
					{DisplayName: "Doran's Ring", ItemID: 1056, Slot: 0},
				},
			},
			{
				SummonerName: "Opponent",
				ChampionName: "Zed",
				Level:        6,
				Position:     "MIDDLE",
				Team:         "CHAOS",
				Scores: riotclient.PlayerScores{
					Kills:      0,
					Deaths:     1,
					Assists:    0,
					CreepScore: 55,
				},
				Items: []riotclient.Item{
					{DisplayName: "Long Sword", ItemID: 1036, Slot: 0},
				},
			},
		},
		GameData: riotclient.GameData{
			GameMode: "CLASSIC",
			GameTime: 300,
		},
	}
}
