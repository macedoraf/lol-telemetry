//go:build integration

package riotclient

import (
	"os"
	"testing"
)

// TestGetGameData_Integration validates the riotclient against a real HTTPS
// mock of the Live Client Data API. It is intended to run inside the Docker
// Compose test stack where MOCK_API_URL points to the mock-api service.
func TestGetGameData_Integration(t *testing.T) {
	baseURL := os.Getenv("MOCK_API_URL")
	if baseURL == "" {
		t.Skip("MOCK_API_URL not set; skipping integration test")
	}

	client := NewClient()
	client.BaseURL = baseURL
	data, err := client.GetGameData()
	if err != nil {
		t.Fatalf("GetGameData() error = %v", err)
	}

	if data.ActivePlayer.SummonerName == "" {
		t.Error("expected active player summoner name to be non-empty")
	}
	if data.GameData.GameTime <= 0 {
		t.Errorf("expected positive game time, got %f", data.GameData.GameTime)
	}
}

func TestEndpointMethods_Integration(t *testing.T) {
	baseURL := os.Getenv("MOCK_API_URL")
	if baseURL == "" {
		t.Skip("MOCK_API_URL not set; skipping integration test")
	}

	client := NewClient()
	client.BaseURL = baseURL

	name, err := client.GetActivePlayerName()
	if err != nil {
		t.Fatalf("GetActivePlayerName() error = %v", err)
	}
	if name == "" {
		t.Error("expected active player name to be non-empty")
	}

	abilities, err := client.GetActivePlayerAbilities()
	if err != nil {
		t.Fatalf("GetActivePlayerAbilities() error = %v", err)
	}
	if abilities.Q.ID == "" {
		t.Error("expected active player Q ability ID to be non-empty")
	}

	runes, err := client.GetActivePlayerRunes()
	if err != nil {
		t.Fatalf("GetActivePlayerRunes() error = %v", err)
	}
	if runes.Keystone.ID == 0 {
		t.Error("expected active player keystone ID to be non-zero")
	}

	players, err := client.GetPlayerList()
	if err != nil {
		t.Fatalf("GetPlayerList() error = %v", err)
	}
	if len(players) == 0 {
		t.Fatal("expected non-empty player list")
	}
	riotId := players[0].RiotID
	if riotId == "" {
		riotId = players[0].SummonerName
	}

	scores, err := client.GetPlayerScores(riotId)
	if err != nil {
		t.Fatalf("GetPlayerScores(%q) error = %v", riotId, err)
	}
	if scores.CreepScore < 0 {
		t.Errorf("expected non-negative creep score, got %d", scores.CreepScore)
	}

	spells, err := client.GetPlayerSummonerSpells(riotId)
	if err != nil {
		t.Fatalf("GetPlayerSummonerSpells(%q) error = %v", riotId, err)
	}
	if spells.SummonerSpellOne.DisplayName == "" {
		t.Error("expected summoner spell one display name to be non-empty")
	}

	mainRunes, err := client.GetPlayerMainRunes(riotId)
	if err != nil {
		t.Fatalf("GetPlayerMainRunes(%q) error = %v", riotId, err)
	}
	if mainRunes.Keystone.ID == 0 {
		t.Error("expected main runes keystone ID to be non-zero")
	}

	items, err := client.GetPlayerItems(riotId)
	if err != nil {
		t.Fatalf("GetPlayerItems(%q) error = %v", riotId, err)
	}
	if len(items) == 0 {
		t.Error("expected at least one item")
	}

	events, err := client.GetEventData()
	if err != nil {
		t.Fatalf("GetEventData() error = %v", err)
	}
	if len(events.Events) == 0 {
		t.Error("expected at least one event")
	}

	stats, err := client.GetGameStats()
	if err != nil {
		t.Fatalf("GetGameStats() error = %v", err)
	}
	if stats.GameMode == "" {
		t.Error("expected game mode to be non-empty")
	}
}
