package service

import (
	"testing"

	"lol-telemetry/pkg/riotclient"
)

func TestNewGameStateFromAllGameData(t *testing.T) {
	data := riotclient.AllGameData{
		ActivePlayer: riotclient.ActivePlayer{
			SummonerName: "Player1",
			CurrentGold:  1234.5,
		},
		AllPlayers: []riotclient.AllPlayer{
			{
				SummonerName: "Player1",
				ChampionName: "Ashe",
				Team:         "ORDER",
				Position:     "BOTTOM",
				Level:        5,
				Scores: riotclient.PlayerScores{
					Kills:      1,
					Deaths:     2,
					Assists:    3,
					CreepScore: 42,
				},
				Items: []riotclient.Item{
					{ItemID: 1055, DisplayName: "Doran's Blade", Slot: 0, CanUse: true},
				},
				Runes: riotclient.PlayerRunes{
					Keystone:          riotclient.RuneTree{DisplayName: "Press the Attack"},
					PrimaryRuneTree:   riotclient.RuneTree{DisplayName: "Precision"},
					SecondaryRuneTree: riotclient.RuneTree{DisplayName: "Inspiration"},
				},
			},
		},
		Events: riotclient.Events{
			Events: []riotclient.Event{
				{EventID: 1, EventName: "GameStart", EventTime: 0.0},
			},
		},
		GameData: riotclient.GameData{
			GameMode: "CLASSIC",
			GameTime: 600.0,
			MapName:  "Map11",
		},
	}

	gs := NewGameStateFromAllGameData(data, "Player1")

	if gs.GameMode != "CLASSIC" {
		t.Errorf("GameMode = %q, want CLASSIC", gs.GameMode)
	}
	if gs.GameTime != 600.0 {
		t.Errorf("GameTime = %f, want 600.0", gs.GameTime)
	}
	if len(gs.Players) != 1 {
		t.Fatalf("Players len = %d, want 1", len(gs.Players))
	}
	p := gs.Players[0]
	if !p.IsActive {
		t.Errorf("IsActive = false, want true")
	}
	if p.CurrentGold != 1234 {
		t.Errorf("CurrentGold = %d, want 1234", p.CurrentGold)
	}
	if p.Runes.Keystone != "Press the Attack" {
		t.Errorf("Keystone = %q, want Press the Attack", p.Runes.Keystone)
	}
	if len(gs.Events) != 1 || gs.Events[0].EventName != "GameStart" {
		t.Errorf("Events not mapped correctly")
	}
	if gs.GameMinute() != 10 {
		t.Errorf("GameMinute() = %d, want 10", gs.GameMinute())
	}
}

func TestBroadcastMessageJSON(t *testing.T) {
	hub := NewHub()
	msg := []byte(`{"type":"test"}`)
	// Ensure Broadcast does not panic and the message is queued.
	hub.Broadcast(msg)
	if len(hub.broadcast) != 1 {
		t.Errorf("broadcast queue len = %d, want 1", len(hub.broadcast))
	}
}

func TestEventMessageRoundTrip(t *testing.T) {
	original := EventMessage{
		EventID:   2,
		EventName: "ChampionKill",
		EventTime: 123.4,
	}
	if original.EventName != "ChampionKill" {
		t.Errorf("EventName = %q", original.EventName)
	}
}
