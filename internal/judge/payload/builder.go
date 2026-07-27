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

	active, ok := riotclient.FindActivePlayer(data)
	if !ok {
		return types.JudgeRequest{}, fmt.Errorf("active player not found")
	}

	opponent := riotclient.FindOpponent(data, active.Position, active.Team)
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
			Events:   eventsFromRiotEvents(data.Events.Events),
		},
		Question:     question,
		SystemPrompt: defaultSystemPrompt(),
		Events:       eventsFromRiotEvents(data.Events.Events),
	}

	req.Matchup.Player.Abilities = abilitiesFromRiotAbilities(data.ActivePlayer.Abilities)
	req.Matchup.Player.Stats = statsFromRiotStats(data.ActivePlayer.ChampionStats)

	if identified {
		req.KDA.Opponent = types.KDA{
			Kills:   opponent.Scores.Kills,
			Deaths:  opponent.Scores.Deaths,
			Assists: opponent.Scores.Assists,
		}
		req.Items.Opponent = itemsFromRiotItems(opponent.Items)
		req.Matchup.Opponent.Abilities = []types.AbilitySnapshot{}
	}

	return req, nil
}

func snapshotFromPlayer(p riotclient.AllPlayer) types.PlayerSnapshot {
	return types.PlayerSnapshot{
		SummonerName:   p.SummonerName,
		ChampionName:   p.ChampionName,
		Level:          p.Level,
		Position:       p.Position,
		Team:           p.Team,
		Kills:          p.Scores.Kills,
		Deaths:         p.Scores.Deaths,
		Assists:        p.Scores.Assists,
		CreepScore:     p.Scores.CreepScore,
		Items:          itemsFromRiotItems(p.Items),
		IsDead:         p.IsDead,
		SummonerSpells: spellsFromRiotSpells(p.SummonerSpells),
		Runes:          runesFromRiotRunes(p.Runes),
	}
}

func spellsFromRiotSpells(spells riotclient.SummonerSpells) []types.SpellSnapshot {
	return []types.SpellSnapshot{
		{Name: spells.SummonerSpellOne.DisplayName},
		{Name: spells.SummonerSpellTwo.DisplayName},
	}
}

func runesFromRiotRunes(runes riotclient.PlayerRunes) types.RuneSnapshot {
	return types.RuneSnapshot{
		Keystone:      runes.Keystone.DisplayName,
		PrimaryTree:   runes.PrimaryRuneTree.DisplayName,
		SecondaryTree: runes.SecondaryRuneTree.DisplayName,
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

func abilitiesFromRiotAbilities(abilities riotclient.Abilities) []types.AbilitySnapshot {
	return []types.AbilitySnapshot{
		{Name: abilities.Passive.DisplayName, Level: abilities.Passive.AbilityLevel},
		{Name: abilities.Q.DisplayName, Level: abilities.Q.AbilityLevel},
		{Name: abilities.W.DisplayName, Level: abilities.W.AbilityLevel},
		{Name: abilities.E.DisplayName, Level: abilities.E.AbilityLevel},
		{Name: abilities.R.DisplayName, Level: abilities.R.AbilityLevel},
	}
}

func statsFromRiotStats(stats riotclient.ChampionStats) types.StatsSnapshot {
	return types.StatsSnapshot{
		AttackDamage:      stats.AttackDamage,
		AbilityPower:      stats.AbilityPower,
		Armor:             stats.Armor,
		MagicResist:       stats.MagicResist,
		AttackSpeed:       stats.AttackSpeed,
		CritChance:        stats.CritChance,
		HealthMax:         stats.MaxHealth,
		HealthCurrent:     stats.CurrentHealth,
		MoveSpeed:         stats.MoveSpeed,
		CooldownReduction: stats.CooldownReduction,
	}
}

func eventsFromRiotEvents(events []riotclient.Event) []types.EventSnapshot {
	out := make([]types.EventSnapshot, 0, len(events))
	for _, e := range events {
		out = append(out, types.EventSnapshot{Name: e.EventName, Time: e.EventTime})
	}
	return out
}

func defaultSystemPrompt() string {
	return "You are a League of Legends tactical assistant. Analyze the current match state and respond ONLY with valid JSON: {\"advice\": \"...\", \"reasoning\": \"...\"}. Advice: single short actionable sentence (max 140 characters). Reasoning: one short sentence citing specific evidence from the data. Focus on macro: objectives, recalls, power spikes, rotations or risk warnings. Be direct, like a coach in the player's ear."
}
