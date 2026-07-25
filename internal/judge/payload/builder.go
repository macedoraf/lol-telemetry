// Package payload builds the Judge request from raw Live Client Data.
package payload

import (
	"fmt"

	"lol-telemetry/internal/types"
	"lol-telemetry/pkg/riotclient"
)

// Builder transforms riotclient.AllGameData into a types.JudgeRequest.
type Builder struct{}

// NewBuilder returns a new payload builder.
func NewBuilder() *Builder {
	return &Builder{}
}

// Build creates a JudgeRequest from the current game snapshot.
func (b *Builder) Build(data riotclient.AllGameData, question string) (types.JudgeRequest, error) {
	gameTime := data.GameData.GameTime
	if gameTime <= 0 {
		return types.JudgeRequest{}, fmt.Errorf("invalid game time: %f", gameTime)
	}

	active, ok := findActivePlayer(data)
	if !ok {
		return types.JudgeRequest{}, fmt.Errorf("active player not found")
	}

	opponent := findOpponent(data, active.Position, active.Team)
	identified := opponent.SummonerName != ""

	req := types.JudgeRequest{
		GameMinute: int(gameTime) / 60,
		Matchup: types.LaneMatchup{
			Player:   snapshotFromPlayer(active),
			Opponent: snapshotFromOpponent(opponent, identified),
		},
		KDA: types.PlayerKDA{
			Player: types.KDA{
				Kills:   active.Scores.Kills,
				Deaths:  active.Scores.Deaths,
				Assists: active.Scores.Assists,
			},
		},
		Gold: types.GoldSnapshot{
			Player: data.ActivePlayer.CurrentGold,
		},
		Items: types.ItemSnapshotPair{
			Player: itemsFromRiotItems(active.Items),
		},
		Objectives: types.TeamObjectives{},
		GameState: types.GameSnapshot{
			GameMode: data.GameData.GameMode,
			GameTime: gameTime,
		},
		Question:     question,
		SystemPrompt: defaultSystemPrompt(),
	}

	if identified {
		req.KDA.Opponent = types.KDA{
			Kills:   opponent.Scores.Kills,
			Deaths:  opponent.Scores.Deaths,
			Assists: opponent.Scores.Assists,
		}
		req.Items.Opponent = itemsFromRiotItems(opponent.Items)
	}

	return req, nil
}

func findActivePlayer(data riotclient.AllGameData) (riotclient.AllPlayer, bool) {
	name := data.ActivePlayer.SummonerName
	for _, p := range data.AllPlayers {
		if p.SummonerName == name {
			return p, true
		}
	}
	return riotclient.AllPlayer{}, false
}

func findOpponent(data riotclient.AllGameData, position, activeTeam string) riotclient.AllPlayer {
	if position == "" {
		return riotclient.AllPlayer{}
	}
	for _, p := range data.AllPlayers {
		if p.Team != activeTeam && p.Position == position {
			return p
		}
	}
	return riotclient.AllPlayer{}
}

func snapshotFromPlayer(p riotclient.AllPlayer) types.PlayerSnapshot {
	return types.PlayerSnapshot{
		SummonerName: p.SummonerName,
		ChampionName: p.ChampionName,
		Level:        p.Level,
		Position:     p.Position,
		Team:         p.Team,
		Kills:        p.Scores.Kills,
		Deaths:       p.Scores.Deaths,
		Assists:      p.Scores.Assists,
		CreepScore:   p.Scores.CreepScore,
		Items:        itemsFromRiotItems(p.Items),
		IsDead:       p.IsDead,
	}
}

func snapshotFromOpponent(p riotclient.AllPlayer, identified bool) types.PlayerSnapshot {
	if !identified {
		return types.PlayerSnapshot{SummonerName: "opponent not identified"}
	}
	return snapshotFromPlayer(p)
}

func itemsFromRiotItems(items []riotclient.Item) []types.ItemSnapshot {
	out := make([]types.ItemSnapshot, 0, len(items))
	for _, it := range items {
		out = append(out, types.ItemSnapshot{
			DisplayName: it.DisplayName,
			ItemID:      it.ItemID,
			Slot:        it.Slot,
		})
	}
	return out
}

func defaultSystemPrompt() string {
	return "You are a League of Legends tactical assistant. Analyze the current match state and respond with a single short actionable sentence (max 140 characters). Focus on macro: objectives, recalls, power spikes, rotations or risk warnings. Do not give long explanations, numbered lists, or repeat obvious data. Be direct, like a coach in the player's ear."
}
