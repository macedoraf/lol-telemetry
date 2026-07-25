// Package types contains shared application models and states.
package types

// PlayerStats holds the metrics computed by the processor.
type PlayerStats struct {
	SummonerName string
	ChampionName string
	Level        int
	CurrentGold  float64
	GameTime     float64
	CSPerMin     float64
	GPM          float64
}

// DashboardState is the current state rendered by the TUI.
type DashboardState struct {
	Stats   PlayerStats
	Error   string
	Waiting bool
}
