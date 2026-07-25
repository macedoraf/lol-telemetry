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
	data, err := client.GetGameDataFromURL(baseURL)
	if err != nil {
		t.Fatalf("GetGameDataFromURL(%q) error = %v", baseURL, err)
	}

	if data.ActivePlayer.SummonerName == "" {
		t.Error("expected active player summoner name to be non-empty")
	}
	if data.GameData.GameTime <= 0 {
		t.Errorf("expected positive game time, got %f", data.GameData.GameTime)
	}
}
