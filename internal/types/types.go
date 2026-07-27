// Package types contains shared application models and states.
package types

import "lol-telemetry/pkg/riotclient"

// HookContext carries the raw game state and deduplication metadata for hooks.
type HookContext struct {
	Data      riotclient.AllGameData
	PrevData  riotclient.AllGameData
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
	SummonerName   string
	ChampionName   string
	Level          int
	Position       string
	Team           string
	Kills          int
	Deaths         int
	Assists        int
	CreepScore     int
	CurrentGold    float64
	Items          []ItemSnapshot
	IsDead         bool
	SummonerSpells []SpellSnapshot
	Runes          RuneSnapshot
	Abilities      []AbilitySnapshot
	Stats          StatsSnapshot
}

// ItemSnapshot is a compact item representation for the Judge.
type ItemSnapshot struct {
	DisplayName string
	ItemID      int
	Slot        int
}

// SpellSnapshot is a compact summoner spell representation for the Judge.
type SpellSnapshot struct {
	Name string
}

// RuneSnapshot is a compact rune representation for the Judge.
type RuneSnapshot struct {
	Keystone      string
	PrimaryTree   string
	SecondaryTree string
}

// AbilitySnapshot is a compact ability representation for the Judge.
type AbilitySnapshot struct {
	Name  string
	Level int
}

// StatsSnapshot is a compact stats representation for the Judge.
type StatsSnapshot struct {
	AttackDamage      float64
	AbilityPower      float64
	Armor             float64
	MagicResist       float64
	AttackSpeed       float64
	CritChance        float64
	HealthMax         float64
	HealthCurrent     float64
	MoveSpeed         float64
	CooldownReduction float64
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

// EventSnapshot is a compact game event for the Judge.
type EventSnapshot struct {
	Name string
	Time float64
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
	Events     []EventSnapshot
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
	Events       []EventSnapshot
}

// JudgeResponse carries the Judge's actionable advice and reasoning.
type JudgeResponse struct {
	Advice    string
	Reasoning string
}
