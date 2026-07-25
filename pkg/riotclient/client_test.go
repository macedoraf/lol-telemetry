package riotclient

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetGameData_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/allgamedata" {
			t.Errorf("expected path /allgamedata, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"activePlayer": {
				"summonerName": "Test",
				"level": 5,
				"currentGold": 1000,
				"championName": "Annie",
				"scores": {
					"creepScore": 30,
					"neutralMinionsKilled": 5
				}
			},
			"allPlayers": [],
			"events": {"Events": []},
			"gameData": {"gameTime": 300, "mapName": "SR", "mapNumber": 11}
		}`))
	}))
	defer server.Close()

	client := NewClient()
	client.BaseURL = server.URL
	data, err := client.GetGameData()
	if err != nil {
		t.Fatalf("GetGameData() error = %v", err)
	}

	if data.ActivePlayer.SummonerName != "Test" {
		t.Errorf("SummonerName = %s, want Test", data.ActivePlayer.SummonerName)
	}
	if data.GameData.GameTime != 300 {
		t.Errorf("GameTime = %f, want 300", data.GameData.GameTime)
	}
}

func TestGetGameData_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient()
	client.BaseURL = server.URL
	_, err := client.GetGameData()
	if err == nil {
		t.Fatal("GetGameData() expected error for 500 response")
	}
}

func TestGetGameData_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{not valid json}`))
	}))
	defer server.Close()

	client := NewClient()
	client.BaseURL = server.URL
	_, err := client.GetGameData()
	if err == nil {
		t.Fatal("GetGameData() expected error for malformed JSON")
	}
}
