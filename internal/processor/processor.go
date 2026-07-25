// Package processor calculates core real-time metrics from Live Client Data.
package processor

import (
	"fmt"

	"lol-telemetry/internal/types"
	"lol-telemetry/pkg/riotclient"
)

// Calculate derives CS/Min and GPM from the raw API data.
// It returns an error when the game time is zero or negative.
func Calculate(data riotclient.AllGameData) (types.PlayerStats, error) {
	gameTime := data.GameData.GameTime
	if gameTime <= 0 {
		return types.PlayerStats{}, fmt.Errorf("invalid game time: %f", gameTime)
	}

	minutes := gameTime / 60.0
	ap := data.ActivePlayer
	totalCS := float64(ap.Scores.CreepScore + ap.Scores.NeutralMinionsKilled)

	stats := types.PlayerStats{
		SummonerName: ap.SummonerName,
		ChampionName: ap.ChampionName,
		Level:        ap.Level,
		CurrentGold:  ap.CurrentGold,
		GameTime:     gameTime,
		CSPerMin:     totalCS / minutes,
		GPM:          ap.CurrentGold / minutes,
	}
	return stats, nil
}
