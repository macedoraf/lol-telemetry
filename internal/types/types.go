// Package types contains shared application models and states.
package types

import "lol-telemetry/pkg/riotclient"

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

// HookContext carries the raw game state and deduplication metadata for hooks.
type HookContext struct {
	Data      riotclient.AllGameData
	GameTime  float64
	PrevFired map[string]int64
}

// Trigger is emitted by a hook when it decides the Judge should run.
type Trigger struct {
	HookName string
	Question string
}

// LaneMatchup describes the player and their lane opponent.
type LaneMatchup struct {
	Player   PlayerSnapshot
	Opponent PlayerSnapshot
}

// PlayerSnapshot is a compact view of a player for the Judge.
type PlayerSnapshot struct {
	SummonerName string
	ChampionName string
	Level        int
	Position     string
	Team         string
	Kills        int
	Deaths       int
	Assists      int
	CreepScore   int
	CurrentGold  float64
	Items        []ItemSnapshot
	IsDead       bool
}

// ItemSnapshot is a compact item representation for the Judge.
type ItemSnapshot struct {
	DisplayName string
	ItemID      int
	Slot        int
}

// GoldSnapshot holds gold information for the player and opponent.
type GoldSnapshot struct {
	Player   float64
	Opponent float64
}

// PlayerKDA holds KDA information for the player and opponent.
type PlayerKDA struct {
	Player   KDA
	Opponent KDA
}

// KDA is a compact kill/death/assist tuple.
type KDA struct {
	Kills   int
	Deaths  int
	Assists int
}

// ItemSnapshotPair holds items for the player and opponent.
type ItemSnapshotPair struct {
	Player   []ItemSnapshot
	Opponent []ItemSnapshot
}

// ObjectiveState holds team objectives.
type ObjectiveState struct {
	Towers  int `json:"towers"`
	Dragons int `json:"dragons"`
	Barons  int `json:"barons"`
	Heralds int `json:"heralds"`
}

// TeamObjectives groups objectives by team.
type TeamObjectives struct {
	Order ObjectiveState
	Chaos ObjectiveState
}

// GameSnapshot captures the overall match state.
type GameSnapshot struct {
	GameMode   string
	GameTime   float64
	ScoreOrder int
	ScoreChaos int
	AliveOrder int
	AliveChaos int
	Objectives TeamObjectives
}

// JudgeRequest is the payload delivered to the Judge for evaluation.
type JudgeRequest struct {
	GameMinute   int
	Matchup      LaneMatchup
	KDA          PlayerKDA
	Gold         GoldSnapshot
	Items        ItemSnapshotPair
	Objectives   TeamObjectives
	GameState    GameSnapshot
	Question     string
	SystemPrompt string
}

// JudgeResponse carries the Judge's actionable advice.
type JudgeResponse struct {
	Advice string
}
