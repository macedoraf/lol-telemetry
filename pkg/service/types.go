// Package service provides the background daemon and WebSocket API for lol-telemetry.
package service

import (
	"encoding/json"
	"time"

	"lol-telemetry/pkg/riotclient"
)

// WSMessage is the envelope for all WebSocket messages.
type WSMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
	Seq     int64           `json:"seq"`
	Ts      int64           `json:"ts"` // unix milliseconds
}

// Message types.
const (
	MsgTypeGameState   = "game_state"
	MsgTypeJudgeAdvice = "judge_advice"
	MsgTypeEvent       = "event"
	MsgTypeError       = "error"
	MsgTypeHello       = "hello"
)

// GameState represents the current game snapshot sent at each poll interval.
type GameState struct {
	GameTime  float64          `json:"gameTime"`
	GameMode  string           `json:"gameMode"`
	MapName   string           `json:"mapName"`
	Players   []PlayerSnapshot `json:"players"`
	Events    []EventMessage   `json:"events,omitempty"`
	Timestamp int64            `json:"timestamp"`
}

// PlayerSnapshot is a minimal player view for overlays.
type PlayerSnapshot struct {
	SummonerName string     `json:"summonerName"`
	ChampionName string     `json:"championName"`
	Team         string     `json:"team"`
	Position     string     `json:"position"`
	Level        int        `json:"level"`
	CS           int        `json:"cs"`
	Kills        int        `json:"kills"`
	Deaths       int        `json:"deaths"`
	Assists      int        `json:"assists"`
	CurrentGold  int        `json:"currentGold"`
	Items        []ItemSnap `json:"items"`
	Runes        RunesSnap  `json:"runes"`
	IsActive     bool       `json:"isActive"`
}

// ItemSnap represents a single item in the overlay view.
type ItemSnap struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Slot     int    `json:"slot"`
	CanUse   bool   `json:"canUse"`
	Cooldown int    `json:"cooldown"`
}

// RunesSnap captures keystone and primary runes.
type RunesSnap struct {
	Keystone  string   `json:"keystone"`
	Primary   []string `json:"primary"`
	Secondary []string `json:"secondary"`
}

// JudgeAdvice is sent when a hook triggers the Judge.
type JudgeAdvice struct {
	HookName   string `json:"hookName"`
	GameMinute int    `json:"gameMinute"`
	Advice     string `json:"advice"`
	Reasoning  string `json:"reasoning"`
	Timestamp  int64  `json:"timestamp"`
}

// EventMessage wraps a game event for real-time broadcasting.
type EventMessage struct {
	EventID   int     `json:"eventID"`
	EventName string  `json:"eventName"`
	EventTime float64 `json:"eventTime"`
	Timestamp int64   `json:"timestamp"`
}

// HelloMessage is sent on connection.
type HelloMessage struct {
	Version  string `json:"version"`
	ServerTS int64  `json:"serverTs"`
	Protocol string `json:"protocol"`
}

// ErrorMessage represents a server error.
type ErrorMessage struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// NewGameStateFromAllGameData converts riotclient.AllGameData to GameState.
func NewGameStateFromAllGameData(data riotclient.AllGameData, activePlayerName string) GameState {
	gs := GameState{
		GameTime:  data.GameData.GameTime,
		GameMode:  data.GameData.GameMode,
		MapName:   data.GameData.MapName,
		Players:   make([]PlayerSnapshot, 0, len(data.AllPlayers)),
		Events:    toEventMessages(data.Events.Events),
		Timestamp: time.Now().UnixMilli(),
	}

	for _, p := range data.AllPlayers {
		items := make([]ItemSnap, 0, len(p.Items))
		for _, it := range p.Items {
			items = append(items, ItemSnap{
				ID:       it.ItemID,
				Name:     it.DisplayName,
				Slot:     it.Slot,
				CanUse:   it.CanUse,
				Cooldown: 0,
			})
		}

		runes := RunesSnap{}
		runes.Keystone = p.Runes.Keystone.DisplayName
		runes.Primary = append(runes.Primary, p.Runes.PrimaryRuneTree.DisplayName)
		runes.Secondary = append(runes.Secondary, p.Runes.SecondaryRuneTree.DisplayName)

		isActive := p.SummonerName == activePlayerName
		gold := 0
		if isActive {
			gold = int(data.ActivePlayer.CurrentGold)
		}

		snap := PlayerSnapshot{
			SummonerName: p.SummonerName,
			ChampionName: p.ChampionName,
			Team:         p.Team,
			Position:     p.Position,
			Level:        p.Level,
			CS:           p.Scores.CreepScore,
			Kills:        p.Scores.Kills,
			Deaths:       p.Scores.Deaths,
			Assists:      p.Scores.Assists,
			CurrentGold:  gold,
			Items:        items,
			Runes:        runes,
			IsActive:     isActive,
		}
		gs.Players = append(gs.Players, snap)
	}

	return gs
}

func toEventMessages(events []riotclient.Event) []EventMessage {
	out := make([]EventMessage, 0, len(events))
	for _, ev := range events {
		out = append(out, EventMessage{
			EventID:   ev.EventID,
			EventName: ev.EventName,
			EventTime: ev.EventTime,
			Timestamp: time.Now().UnixMilli(),
		})
	}
	return out
}

// GameMinute returns the current game minute from GameState.
func (gs GameState) GameMinute() int {
	return int(gs.GameTime) / 60
}
