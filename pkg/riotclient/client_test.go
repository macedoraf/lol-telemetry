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
				"abilities": {"E": {}, "Passive": {}, "Q": {}, "R": {}, "W": {}},
				"championStats": {},
				"fullRunes": {
					"generalRunes": [],
					"keystone": {},
					"primaryRuneTree": {},
					"secondaryRuneTree": {},
					"statRunes": []
				}
			},
			"allPlayers": [],
			"events": {"Events": []},
			"gameData": {"gameMode": "CLASSIC", "gameTime": 300, "mapName": "SR", "mapNumber": 11, "mapTerrain": "Default"}
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

func TestGetActivePlayerName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/activeplayername" {
			t.Errorf("expected path /activeplayername, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`"Riot Tuxedo#TXC1"`))
	}))
	defer server.Close()

	client := NewClient()
	client.BaseURL = server.URL
	name, err := client.GetActivePlayerName()
	if err != nil {
		t.Fatalf("GetActivePlayerName() error = %v", err)
	}
	if name != "Riot Tuxedo#TXC1" {
		t.Errorf("GetActivePlayerName() = %q, want Riot Tuxedo#TXC1", name)
	}
}

func TestGetActivePlayerAbilities(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/activeplayerabilities" {
			t.Errorf("expected path /activeplayerabilities, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"E": {"displayName": "Molten Shield", "id": "AnnieE"},
			"Passive": {"displayName": "Pyromania", "id": "AnniePassive"},
			"Q": {"displayName": "Disintegrate", "id": "AnnieQ"},
			"R": {"displayName": "Summon: Tibbers", "id": "AnnieR"},
			"W": {"displayName": "Incinerate", "id": "AnnieW"}
		}`))
	}))
	defer server.Close()

	client := NewClient()
	client.BaseURL = server.URL
	abilities, err := client.GetActivePlayerAbilities()
	if err != nil {
		t.Fatalf("GetActivePlayerAbilities() error = %v", err)
	}
	if abilities.Q.DisplayName != "Disintegrate" {
		t.Errorf("Q.DisplayName = %q, want Disintegrate", abilities.Q.DisplayName)
	}
	if abilities.Passive.ID != "AnniePassive" {
		t.Errorf("Passive.ID = %q, want AnniePassive", abilities.Passive.ID)
	}
}

func TestGetActivePlayerRunes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/activeplayerrunes" {
			t.Errorf("expected path /activeplayerrunes, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"generalRunes": [],
			"keystone": {"displayName": "Electrocute", "id": 8112},
			"primaryRuneTree": {"displayName": "Domination", "id": 8100},
			"secondaryRuneTree": {"displayName": "Sorcery", "id": 8200},
			"statRunes": []
		}`))
	}))
	defer server.Close()

	client := NewClient()
	client.BaseURL = server.URL
	runes, err := client.GetActivePlayerRunes()
	if err != nil {
		t.Fatalf("GetActivePlayerRunes() error = %v", err)
	}
	if runes.Keystone.ID != 8112 {
		t.Errorf("Keystone.ID = %d, want 8112", runes.Keystone.ID)
	}
	if runes.SecondaryRuneTree.DisplayName != "Sorcery" {
		t.Errorf("SecondaryRuneTree.DisplayName = %q, want Sorcery", runes.SecondaryRuneTree.DisplayName)
	}
}

func TestGetPlayerList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/playerlist" {
			t.Errorf("expected path /playerlist, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{
			"championName": "Annie",
			"isBot": false,
			"isDead": false,
			"items": [],
			"level": 1,
			"position": "MIDDLE",
			"rawChampionName": "game_character_displayname_Annie",
			"respawnTimer": 0.0,
			"runes": {},
			"scores": {"assists": 0, "creepScore": 0, "deaths": 0, "kills": 0, "wardScore": 0},
			"skinID": 0,
			"summonerName": "Riot Tuxedo",
			"riotId": "Riot Tuxedo#TXC1",
			"riotIdGameName": "Riot Tuxedo",
			"riotIdTagLine": "TXC1",
			"summonerSpells": {},
			"team": "ORDER"
		}]`))
	}))
	defer server.Close()

	client := NewClient()
	client.BaseURL = server.URL
	players, err := client.GetPlayerList()
	if err != nil {
		t.Fatalf("GetPlayerList() error = %v", err)
	}
	if len(players) != 1 {
		t.Fatalf("len(players) = %d, want 1", len(players))
	}
	if players[0].RiotID != "Riot Tuxedo#TXC1" {
		t.Errorf("RiotID = %q, want Riot Tuxedo#TXC1", players[0].RiotID)
	}
	if players[0].RiotIDTagLine != "TXC1" {
		t.Errorf("RiotIDTagLine = %q, want TXC1", players[0].RiotIDTagLine)
	}
}

func TestGetPlayerScores(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/playerscores" {
			t.Errorf("expected path /playerscores, got %s", r.URL.Path)
		}
		riotId := r.URL.Query().Get("riotId")
		if riotId != "Riot Tuxedo#TXC1" {
			t.Errorf("riotId query = %q, want Riot Tuxedo#TXC1", riotId)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"assists": 1, "creepScore": 42, "deaths": 2, "kills": 3, "wardScore": 0.5}`))
	}))
	defer server.Close()

	client := NewClient()
	client.BaseURL = server.URL
	scores, err := client.GetPlayerScores("Riot Tuxedo#TXC1")
	if err != nil {
		t.Fatalf("GetPlayerScores() error = %v", err)
	}
	if scores.Kills != 3 {
		t.Errorf("Kills = %d, want 3", scores.Kills)
	}
	if scores.CreepScore != 42 {
		t.Errorf("CreepScore = %d, want 42", scores.CreepScore)
	}
}

func TestGetPlayerScores_URLEncoding(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		riotId := r.URL.Query().Get("riotId")
		if riotId != "Riot Tuxedo#TXC1" {
			t.Errorf("riotId query = %q, want Riot Tuxedo#TXC1", riotId)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"assists": 0, "creepScore": 0, "deaths": 0, "kills": 0, "wardScore": 0}`))
	}))
	defer server.Close()

	client := NewClient()
	client.BaseURL = server.URL
	_, err := client.GetPlayerScores("Riot Tuxedo#TXC1")
	if err != nil {
		t.Fatalf("GetPlayerScores() error = %v", err)
	}
}

func TestGetPlayerSummonerSpells(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/playersummonerspells" {
			t.Errorf("expected path /playersummonerspells, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"summonerSpellOne": {"displayName": "Flash"},
			"summonerSpellTwo": {"displayName": "Ignite"}
		}`))
	}))
	defer server.Close()

	client := NewClient()
	client.BaseURL = server.URL
	spells, err := client.GetPlayerSummonerSpells("Riot Tuxedo#TXC1")
	if err != nil {
		t.Fatalf("GetPlayerSummonerSpells() error = %v", err)
	}
	if spells.SummonerSpellOne.DisplayName != "Flash" {
		t.Errorf("SummonerSpellOne.DisplayName = %q, want Flash", spells.SummonerSpellOne.DisplayName)
	}
	if spells.SummonerSpellTwo.DisplayName != "Ignite" {
		t.Errorf("SummonerSpellTwo.DisplayName = %q, want Ignite", spells.SummonerSpellTwo.DisplayName)
	}
}

func TestGetPlayerMainRunes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/playermainrunes" {
			t.Errorf("expected path /playermainrunes, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"keystone": {"displayName": "Electrocute", "id": 8112},
			"primaryRuneTree": {"displayName": "Domination", "id": 8100},
			"secondaryRuneTree": {"displayName": "Sorcery", "id": 8200}
		}`))
	}))
	defer server.Close()

	client := NewClient()
	client.BaseURL = server.URL
	runes, err := client.GetPlayerMainRunes("Riot Tuxedo#TXC1")
	if err != nil {
		t.Fatalf("GetPlayerMainRunes() error = %v", err)
	}
	if runes.Keystone.ID != 8112 {
		t.Errorf("Keystone.ID = %d, want 8112", runes.Keystone.ID)
	}
}

func TestGetPlayerItems(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/playeritems" {
			t.Errorf("expected path /playeritems, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"displayName": "Warding Totem (Trinket)", "itemID": 3340, "slot": 6}]`))
	}))
	defer server.Close()

	client := NewClient()
	client.BaseURL = server.URL
	items, err := client.GetPlayerItems("Riot Tuxedo#TXC1")
	if err != nil {
		t.Fatalf("GetPlayerItems() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].ItemID != 3340 {
		t.Errorf("ItemID = %d, want 3340", items[0].ItemID)
	}
}

func TestGetEventData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/eventdata" {
			t.Errorf("expected path /eventdata, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"Events": [{"EventID": 0, "EventName": "GameStart", "EventTime": 0.5}]}`))
	}))
	defer server.Close()

	client := NewClient()
	client.BaseURL = server.URL
	events, err := client.GetEventData()
	if err != nil {
		t.Fatalf("GetEventData() error = %v", err)
	}
	if len(events.Events) != 1 {
		t.Fatalf("len(events.Events) = %d, want 1", len(events.Events))
	}
	if events.Events[0].EventName != "GameStart" {
		t.Errorf("EventName = %q, want GameStart", events.Events[0].EventName)
	}
}

func TestGetGameStats(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/gamestats" {
			t.Errorf("expected path /gamestats, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"gameMode": "CLASSIC", "gameTime": 123.4, "mapName": "Map11", "mapNumber": 11, "mapTerrain": "Default"}`))
	}))
	defer server.Close()

	client := NewClient()
	client.BaseURL = server.URL
	stats, err := client.GetGameStats()
	if err != nil {
		t.Fatalf("GetGameStats() error = %v", err)
	}
	if stats.GameMode != "CLASSIC" {
		t.Errorf("GameMode = %q, want CLASSIC", stats.GameMode)
	}
	if stats.MapNumber != 11 {
		t.Errorf("MapNumber = %d, want 11", stats.MapNumber)
	}
}
