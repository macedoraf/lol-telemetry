package payload

import (
	"testing"

	"lol-telemetry/pkg/riotclient"
)

func TestBuilder_Build(t *testing.T) {
	b := NewBuilder()
	data := fullGameData()

	req, err := b.Build(data, "test question")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if req.GameMinute != 10 {
		t.Errorf("GameMinute = %d, want 10", req.GameMinute)
	}
	if req.Question != "test question" {
		t.Errorf("Question = %s, want test question", req.Question)
	}
	if req.SystemPrompt == "" {
		t.Error("SystemPrompt is empty")
	}
	if req.Matchup.Player.SummonerName != "ActivePlayer" {
		t.Errorf("player summoner name = %s, want ActivePlayer", req.Matchup.Player.SummonerName)
	}
	if req.Matchup.Opponent.SummonerName != "Opponent" {
		t.Errorf("opponent summoner name = %s, want Opponent", req.Matchup.Opponent.SummonerName)
	}
	if req.KDA.Player.Kills != 2 {
		t.Errorf("player kills = %d, want 2", req.KDA.Player.Kills)
	}
	if req.KDA.Opponent.Deaths != 2 {
		t.Errorf("opponent deaths = %d, want 2", req.KDA.Opponent.Deaths)
	}
	if len(req.Items.Player) != 1 {
		t.Errorf("player items = %d, want 1", len(req.Items.Player))
	}
	if len(req.Items.Opponent) != 1 {
		t.Errorf("opponent items = %d, want 1", len(req.Items.Opponent))
	}
}

func TestBuilder_Build_OpponentNotIdentified(t *testing.T) {
	b := NewBuilder()
	data := fullGameData()
	// Remove active player position so opponent cannot be matched.
	data.AllPlayers[0].Position = ""
	data.ActivePlayer.SummonerName = "ActivePlayer"

	req, err := b.Build(data, "test question")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if req.Matchup.Opponent.SummonerName != "opponent not identified" {
		t.Errorf("opponent = %s, want opponent not identified", req.Matchup.Opponent.SummonerName)
	}
}

func TestBuilder_Build_InvalidGameTime(t *testing.T) {
	b := NewBuilder()
	data := fullGameData()
	data.GameData.GameTime = 0

	_, err := b.Build(data, "test question")
	if err == nil {
		t.Fatal("expected error for invalid game time")
	}
}

func fullGameData() riotclient.AllGameData {
	return riotclient.AllGameData{
		ActivePlayer: riotclient.ActivePlayer{
			SummonerName: "ActivePlayer",
			CurrentGold:  1234.5,
			Level:        7,
		},
		AllPlayers: []riotclient.AllPlayer{
			{
				SummonerName: "ActivePlayer",
				ChampionName: "Annie",
				Level:        7,
				Position:     "MIDDLE",
				Team:         "ORDER",
				Scores: riotclient.PlayerScores{
					Kills:      2,
					Deaths:     1,
					Assists:    3,
					CreepScore: 80,
				},
				Items: []riotclient.Item{
					{DisplayName: "Doran's Ring", ItemID: 1056, Slot: 0},
				},
			},
			{
				SummonerName: "Opponent",
				ChampionName: "Zed",
				Level:        8,
				Position:     "MIDDLE",
				Team:         "CHAOS",
				Scores: riotclient.PlayerScores{
					Kills:      1,
					Deaths:     2,
					Assists:    0,
					CreepScore: 90,
				},
				Items: []riotclient.Item{
					{DisplayName: "Long Sword", ItemID: 1036, Slot: 0},
				},
			},
		},
		GameData: riotclient.GameData{
			GameMode: "CLASSIC",
			GameTime: 600,
		},
	}
}
