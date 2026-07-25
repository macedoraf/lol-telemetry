package riotclient

import (
	"encoding/json"
	"os"
	"testing"
)

func TestUnmarshalAllGameData(t *testing.T) {
	payload, err := os.ReadFile("../../testdata/mocks/allgamedata.json")
	if err != nil {
		t.Fatalf("failed to read mock fixture: %v", err)
	}

	var data AllGameData
	if err := json.Unmarshal(payload, &data); err != nil {
		t.Fatalf("json.Unmarshal error = %v", err)
	}

	if data.ActivePlayer.SummonerName != "Riot Tuxedo" {
		t.Errorf("ActivePlayer.SummonerName = %q, want Riot Tuxedo", data.ActivePlayer.SummonerName)
	}
	if data.ActivePlayer.Level != 1 {
		t.Errorf("ActivePlayer.Level = %d, want 1", data.ActivePlayer.Level)
	}
	if data.ActivePlayer.ChampionStats.AbilityPower != 0 {
		t.Errorf("ActivePlayer.ChampionStats.AbilityPower = %f, want 0", data.ActivePlayer.ChampionStats.AbilityPower)
	}
	if data.ActivePlayer.ChampionStats.ResourceType != "MANA" {
		t.Errorf("ActivePlayer.ChampionStats.ResourceType = %q, want MANA", data.ActivePlayer.ChampionStats.ResourceType)
	}
	if data.ActivePlayer.Abilities.Q.DisplayName != "Disintegrate" {
		t.Errorf("ActivePlayer.Abilities.Q.DisplayName = %q, want Disintegrate", data.ActivePlayer.Abilities.Q.DisplayName)
	}
	if data.ActivePlayer.FullRunes.Keystone.ID != 8112 {
		t.Errorf("ActivePlayer.FullRunes.Keystone.ID = %d, want 8112", data.ActivePlayer.FullRunes.Keystone.ID)
	}
	if data.ActivePlayer.FullRunes.Keystone.DisplayName != "Electrocute" {
		t.Errorf("ActivePlayer.FullRunes.Keystone.DisplayName = %q, want Electrocute", data.ActivePlayer.FullRunes.Keystone.DisplayName)
	}
	if len(data.ActivePlayer.FullRunes.GeneralRunes) != 6 {
		t.Errorf("len(ActivePlayer.FullRunes.GeneralRunes) = %d, want 6", len(data.ActivePlayer.FullRunes.GeneralRunes))
	}
	if len(data.ActivePlayer.FullRunes.StatRunes) != 3 {
		t.Errorf("len(ActivePlayer.FullRunes.StatRunes) = %d, want 3", len(data.ActivePlayer.FullRunes.StatRunes))
	}
	if data.ActivePlayer.FullRunes.StatRunes[0].ID != 5008 {
		t.Errorf("ActivePlayer.FullRunes.StatRunes[0].ID = %d, want 5008", data.ActivePlayer.FullRunes.StatRunes[0].ID)
	}

	if len(data.AllPlayers) != 1 {
		t.Fatalf("len(AllPlayers) = %d, want 1", len(data.AllPlayers))
	}
	player := data.AllPlayers[0]
	if player.ChampionName != "Annie" {
		t.Errorf("AllPlayers[0].ChampionName = %q, want Annie", player.ChampionName)
	}
	if player.Runes.Keystone.ID != 8112 {
		t.Errorf("AllPlayers[0].Runes.Keystone.ID = %d, want 8112", player.Runes.Keystone.ID)
	}
	if player.Scores.Kills != 0 {
		t.Errorf("AllPlayers[0].Scores.Kills = %d, want 0", player.Scores.Kills)
	}
	if player.SummonerName != "Riot Tuxedo" {
		t.Errorf("AllPlayers[0].SummonerName = %q, want Riot Tuxedo", player.SummonerName)
	}
	if player.SummonerSpells.SummonerSpellOne.DisplayName != "Flash" {
		t.Errorf("AllPlayers[0].SummonerSpells.SummonerSpellOne.DisplayName = %q, want Flash", player.SummonerSpells.SummonerSpellOne.DisplayName)
	}
	if player.SummonerSpells.SummonerSpellTwo.DisplayName != "Ignite" {
		t.Errorf("AllPlayers[0].SummonerSpells.SummonerSpellTwo.DisplayName = %q, want Ignite", player.SummonerSpells.SummonerSpellTwo.DisplayName)
	}
	if player.Team != "ORDER" {
		t.Errorf("AllPlayers[0].Team = %q, want ORDER", player.Team)
	}
	if player.RiotID != "Riot Tuxedo#1234" {
		t.Errorf("AllPlayers[0].RiotID = %q, want Riot Tuxedo#1234", player.RiotID)
	}
	if player.RiotIDGameName != "Riot Tuxedo" {
		t.Errorf("AllPlayers[0].RiotIDGameName = %q, want Riot Tuxedo", player.RiotIDGameName)
	}
	if player.RiotIDTagLine != "1234" {
		t.Errorf("AllPlayers[0].RiotIDTagLine = %q, want 1234", player.RiotIDTagLine)
	}
	if len(player.Items) != 1 {
		t.Fatalf("len(AllPlayers[0].Items) = %d, want 1", len(player.Items))
	}
	if player.Items[0].DisplayName != "Doran's Ring" {
		t.Errorf("AllPlayers[0].Items[0].DisplayName = %q, want Doran's Ring", player.Items[0].DisplayName)
	}
	if player.Items[0].ItemID != 1056 {
		t.Errorf("AllPlayers[0].Items[0].ItemID = %d, want 1056", player.Items[0].ItemID)
	}

	if len(data.Events.Events) != 1 {
		t.Fatalf("len(Events.Events) = %d, want 1", len(data.Events.Events))
	}
	if data.Events.Events[0].EventName != "GameStart" {
		t.Errorf("Events.Events[0].EventName = %q, want GameStart", data.Events.Events[0].EventName)
	}

	if data.GameData.GameMode != "CLASSIC" {
		t.Errorf("GameData.GameMode = %q, want CLASSIC", data.GameData.GameMode)
	}
	if data.GameData.MapName != "Map11" {
		t.Errorf("GameData.MapName = %q, want Map11", data.GameData.MapName)
	}
	if data.GameData.MapTerrain != "Default" {
		t.Errorf("GameData.MapTerrain = %q, want Default", data.GameData.MapTerrain)
	}
}

func TestUnmarshalAllPlayer_MissingOptionalFields(t *testing.T) {
	payload := []byte(`{
		"activePlayer": {
			"summonerName": "Test",
			"level": 1,
			"currentGold": 0,
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
		"allPlayers": [{
			"championName": "Annie",
			"isBot": false,
			"isDead": false,
			"items": [],
			"level": 1,
			"position": "",
			"rawChampionName": "game_character_displayname_Annie",
			"respawnTimer": 0,
			"runes": {
				"keystone": {},
				"primaryRuneTree": {},
				"secondaryRuneTree": {}
			},
			"scores": {"assists": 0, "creepScore": 0, "deaths": 0, "kills": 0, "wardScore": 0},
			"skinID": 0,
			"summonerName": "Riot Tuxedo",
			"summonerSpells": {"summonerSpellOne": {}, "summonerSpellTwo": {}},
			"team": "ORDER"
		}],
		"events": {"Events": []},
		"gameData": {"gameMode": "CLASSIC", "gameTime": 0, "mapName": "Map11", "mapNumber": 11, "mapTerrain": "Default"}
	}`)

	var data AllGameData
	if err := json.Unmarshal(payload, &data); err != nil {
		t.Fatalf("json.Unmarshal error = %v", err)
	}

	if data.AllPlayers[0].RiotID != "" {
		t.Errorf("RiotID = %q, want empty", data.AllPlayers[0].RiotID)
	}
	if data.AllPlayers[0].RiotIDGameName != "" {
		t.Errorf("RiotIDGameName = %q, want empty", data.AllPlayers[0].RiotIDGameName)
	}
	if data.AllPlayers[0].RiotIDTagLine != "" {
		t.Errorf("RiotIDTagLine = %q, want empty", data.AllPlayers[0].RiotIDTagLine)
	}
	if len(data.AllPlayers[0].Items) != 0 {
		t.Errorf("len(Items) = %d, want 0", len(data.AllPlayers[0].Items))
	}
}
