package payload

import (
	"strings"
	"testing"

	"lol-telemetry/pkg/riotclient"
)

func TestBuilder_Build(t *testing.T) {
	b := NewBuilder("en")
	data := fullGameData()

	req, err := b.Build(data, "test question")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if req.GameMinute != 10 {
		t.Errorf("GameMinute = %d, want 10", req.GameMinute)
	}
	if req.Question != "test question" {
		t.Errorf("Question = %s, want test question", req.Question)
	}
	if req.SystemPrompt == "" {
		t.Error("SystemPrompt is empty")
	}
	if req.Matchup.Player.SummonerName != "ActivePlayer" {
		t.Errorf("player summoner name = %s, want ActivePlayer", req.Matchup.Player.SummonerName)
	}
	if req.Matchup.Opponent.SummonerName != "Opponent" {
		t.Errorf("opponent summoner name = %s, want Opponent", req.Matchup.Opponent.SummonerName)
	}
	if req.KDA.Player.Kills != 2 {
		t.Errorf("player kills = %d, want 2", req.KDA.Player.Kills)
	}
	if req.KDA.Opponent.Deaths != 2 {
		t.Errorf("opponent deaths = %d, want 2", req.KDA.Opponent.Deaths)
	}
	if len(req.Items.Player) != 1 {
		t.Errorf("player items = %d, want 1", len(req.Items.Player))
	}
	if len(req.Items.Opponent) != 1 {
		t.Errorf("opponent items = %d, want 1", len(req.Items.Opponent))
	}
}

func TestBuilder_Build_OpponentNotIdentified(t *testing.T) {
	b := NewBuilder("en")
	data := fullGameData()
	// Remove active player position so opponent cannot be matched.
	data.AllPlayers[0].Position = ""
	data.ActivePlayer.SummonerName = "ActivePlayer"

	req, err := b.Build(data, "test question")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if req.Matchup.Opponent.SummonerName != "opponent not identified" {
		t.Errorf("opponent = %s, want opponent not identified", req.Matchup.Opponent.SummonerName)
	}
}

func TestBuilder_Build_InvalidGameTime(t *testing.T) {
	b := NewBuilder("en")
	data := fullGameData()
	data.GameData.GameTime = 0

	_, err := b.Build(data, "test question")
	if err == nil {
		t.Fatal("expected error for invalid game time")
	}
}

func TestBuilder_DefaultLanguage(t *testing.T) {
	b := NewBuilder("")
	if b.Language() != "en" {
		t.Errorf("default language = %q, want en", b.Language())
	}
}

func TestBuilder_SetLanguage(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"en", "en"},
		{"pt-BR", "pt-BR"},
		{"es", "es"},
		{"fr", "en"},
		{"", "en"},
		{"EN", "en"},
	}
	for _, tc := range tests {
		b := NewBuilder(tc.input)
		if b.Language() != tc.want {
			t.Errorf("NewBuilder(%q).Language() = %q, want %q", tc.input, b.Language(), tc.want)
		}
	}
}

func TestBuilder_LanguagePromptContainsDirective(t *testing.T) {
	tests := []struct {
		lang     string
		contains string
	}{
		{"en", "Respond entirely in English"},
		{"pt-BR", "Respond entirely in Brazilian Portuguese"},
		{"es", "Respond entirely in Spanish"},
	}
	for _, tc := range tests {
		b := NewBuilder(tc.lang)
		req, err := b.Build(fullGameData(), "?")
		if err != nil {
			t.Fatalf("lang=%s: Build error: %v", tc.lang, err)
		}
		if !strings.Contains(req.SystemPrompt, tc.contains) {
			t.Errorf("lang=%s: SystemPrompt missing %q\n got: %s", tc.lang, tc.contains, req.SystemPrompt)
		}
		if !strings.Contains(req.SystemPrompt, "JSON keys must remain in English") {
			t.Errorf("lang=%s: SystemPrompt missing JSON keys directive", tc.lang)
		}
	}
}

func TestBuilder_PromptOverride(t *testing.T) {
	tests := []struct {
		name      string
		override  string
		wantErr   bool
		wantContains string
	}{
		{"default no override", "", false, "You are a League of Legends tactical assistant"},
		{"valid override", "Focus only on jungle pathing.", false, "Focus only on jungle pathing"},
		{"whitespace only", "   ", true, ""},
		{"too long", strings.Repeat("x", 4001), true, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := NewBuilder("en")
			err := b.SetPromptOverride(tc.override)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.Contains(b.EffectivePrompt(), tc.wantContains) {
				t.Errorf("EffectivePrompt missing %q\n got: %s", tc.wantContains, b.EffectivePrompt())
			}
			if !strings.Contains(b.EffectivePrompt(), "Respond entirely in English") {
				t.Errorf("EffectivePrompt missing language directive")
			}
		})
	}
}

func TestBuilder_PromptOverride_Clear(t *testing.T) {
	b := NewBuilder("en")
	if err := b.SetPromptOverride("Custom prompt."); err != nil {
		t.Fatalf("SetPromptOverride: %v", err)
	}
	if !strings.Contains(b.EffectivePrompt(), "Custom prompt.") {
		t.Fatalf("override not applied")
	}
	if err := b.SetPromptOverride(""); err != nil {
		t.Fatalf("clear override: %v", err)
	}
	if strings.Contains(b.EffectivePrompt(), "Custom prompt.") {
		t.Errorf("override still present after clear")
	}
	if !strings.Contains(b.EffectivePrompt(), "You are a League of Legends tactical assistant") {
		t.Errorf("default prompt not restored")
	}
}

func TestBuilder_PromptOverride_LanguageChange(t *testing.T) {
	b := NewBuilder("en")
	if err := b.SetPromptOverride("Override."); err != nil {
		t.Fatalf("SetPromptOverride: %v", err)
	}
	b.SetLanguage("pt-BR")
	if !strings.Contains(b.EffectivePrompt(), "Override.") {
		t.Errorf("override lost after language change")
	}
	if !strings.Contains(b.EffectivePrompt(), "Respond entirely in Brazilian Portuguese") {
		t.Errorf("language directive not updated")
	}
}

func TestNormalizeLanguage(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"en", "en"},
		{"pt-BR", "pt-BR"},
		{"es", "es"},
		{"fr", "en"},
		{"", "en"},
		{"pt_br", "en"},
		{"ENGLISH", "en"},
	}
	for _, tc := range tests {
		got := NormalizeLanguage(tc.input)
		if got != tc.want {
			t.Errorf("NormalizeLanguage(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func fullGameData() riotclient.AllGameData {
	return riotclient.AllGameData{
		ActivePlayer: riotclient.ActivePlayer{
			SummonerName: "ActivePlayer",
			CurrentGold:  1234.5,
			Level:        7,
		},
		AllPlayers: []riotclient.AllPlayer{
			{
				SummonerName: "ActivePlayer",
				ChampionName: "Annie",
				Level:        7,
				Position:     "MIDDLE",
				Team:         "ORDER",
				Scores: riotclient.PlayerScores{
					Kills:      2,
					Deaths:     1,
					Assists:    3,
					CreepScore: 80,
				},
				Items: []riotclient.Item{
					{DisplayName: "Doran's Ring", ItemID: 1056, Slot: 0},
				},
			},
			{
				SummonerName: "Opponent",
				ChampionName: "Zed",
				Level:        8,
				Position:     "MIDDLE",
				Team:         "CHAOS",
				Scores: riotclient.PlayerScores{
					Kills:      1,
					Deaths:     2,
					Assists:    0,
					CreepScore: 90,
				},
				Items: []riotclient.Item{
					{DisplayName: "Long Sword", ItemID: 1036, Slot: 0},
				},
			},
		},
		GameData: riotclient.GameData{
			GameMode: "CLASSIC",
			GameTime: 600,
		},
	}
}
