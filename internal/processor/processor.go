// Package processor calculates core real-time metrics from Live Client Data.
package processor

import (
	"fmt"

	"lol-telemetry/internal/types"
	"lol-telemetry/pkg/riotclient"
)

// Calculate derives CS/Min and GPM from the raw API data.
// It returns an error when the game time is zero or negative.
// The active player's champion name and creep score are looked up from the
// allPlayers list by summoner name, because the /allgamedata activePlayer
// block does not include those fields.
func Calculate(data riotclient.AllGameData) (types.PlayerStats, error) {
	gameTime := data.GameData.GameTime
	if gameTime <= 0 {
		return types.PlayerStats{}, fmt.Errorf("invalid game time: %f", gameTime)
	}

	minutes := gameTime / 60.0
	ap := data.ActivePlayer

	var championName string
	var totalCS int
	for _, p := range data.AllPlayers {
		if p.SummonerName == ap.SummonerName {
			championName = p.ChampionName
			totalCS = p.Scores.CreepScore
			break
		}
	}

	stats := types.PlayerStats{
		SummonerName: ap.SummonerName,
		ChampionName: championName,
		Level:        ap.Level,
		CurrentGold:  ap.CurrentGold,
		GameTime:     gameTime,
		CSPerMin:     float64(totalCS) / minutes,
		GPM:          ap.CurrentGold / minutes,
	}
	return stats, nil
}
