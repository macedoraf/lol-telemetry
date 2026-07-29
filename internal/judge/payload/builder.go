// Package payload builds the Judge request from raw Live Client Data.
package payload

import (
	"fmt"
	"strings"
	"sync/atomic"

	"lol-telemetry/internal/types"
	"lol-telemetry/pkg/riotclient"
)

var languageNames = map[string]string{
	"en":    "English",
	"pt-BR": "Brazilian Portuguese",
	"es":    "Spanish",
}

// Builder transforms riotclient.AllGameData into a types.JudgeRequest.
type Builder struct {
	lang           atomic.Value // stores string, always normalized
	promptOverride atomic.Value // stores string, may be empty
}

// NewBuilder returns a new payload builder with the given language.
func NewBuilder(language string) *Builder {
	b := &Builder{}
	b.SetLanguage(language)
	return b
}

// SetLanguage changes the output language thread-safely.
func (b *Builder) SetLanguage(lang string) {
	b.lang.Store(NormalizeLanguage(lang))
}

// Language returns the current normalized language code.
func (b *Builder) Language() string {
	v := b.lang.Load()
	if v == nil {
		return "en"
	}
	return v.(string)
}

// NormalizeLanguage validates a language code and returns a known value.
func NormalizeLanguage(lang string) string {
	if _, ok := languageNames[lang]; ok {
		return lang
	}
	return "en"
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

	opponent, identified := riotclient.FindOpponent(data, active.Position, active.Team)

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
		SystemPrompt: b.systemPrompt(),
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

// SetPromptOverride sets a custom system prompt base. Empty string clears the
// override and restores the default prompt. Returns an error for whitespace-only
// or overly long prompts.
func (b *Builder) SetPromptOverride(prompt string) error {
	trimmed := strings.TrimSpace(prompt)
	if prompt != "" && trimmed == "" {
		return fmt.Errorf("prompt cannot be whitespace-only")
	}
	if len(trimmed) > 4000 {
		return fmt.Errorf("prompt exceeds 4000 characters")
	}
	b.promptOverride.Store(trimmed)
	return nil
}

// PromptOverride returns the user-supplied prompt override, or "" if none.
func (b *Builder) PromptOverride() string {
	v := b.promptOverride.Load()
	if v == nil {
		return ""
	}
	return v.(string)
}

// EffectivePrompt returns the prompt that will be sent to the LLM, including
// the language directive.
func (b *Builder) EffectivePrompt() string {
	return b.systemPrompt()
}

// systemPrompt returns the base prompt with a language directive appended.
func (b *Builder) systemPrompt() string {
	base := defaultSystemPrompt()
	if override := b.PromptOverride(); override != "" {
		base = override
	}
	langName := languageNames[b.Language()]
	return base + "\nRespond entirely in " + langName + ". JSON keys must remain in English."
}
