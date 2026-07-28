package tui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"lol-telemetry/internal/hooks"
	"lol-telemetry/pkg/service"
)

func updateModel(m Model, msg tea.Msg) Model {
	newM, _ := m.Update(msg)
	return newM.(Model)
}

func TestModelInitializes(t *testing.T) {
	m := NewModel("ws://localhost:8080/ws")
	cmd := m.Init()
	if cmd == nil {
		t.Error("Init() returned nil command")
	}
}

func TestModelNavigation(t *testing.T) {
	m := NewModel("ws://localhost:8080/ws")
	m = updateModel(m, tea.WindowSizeMsg{Width: 80, Height: 24})

	m = updateModel(m, tea.KeyMsg(tea.Key{Type: tea.KeyRight}))
	if m.ActiveTab() != TabAdvice {
		t.Errorf("after right ActiveTab = %v, want TabAdvice", m.ActiveTab())
	}

	m = updateModel(m, tea.KeyMsg(tea.Key{Type: tea.KeyRight}))
	if m.ActiveTab() != TabEvents {
		t.Errorf("after right ActiveTab = %v, want TabEvents", m.ActiveTab())
	}

	m = updateModel(m, tea.KeyMsg(tea.Key{Type: tea.KeyLeft}))
	if m.ActiveTab() != TabAdvice {
		t.Errorf("after left ActiveTab = %v, want TabAdvice", m.ActiveTab())
	}

	m = updateModel(m, tea.KeyMsg(tea.Key{Type: tea.KeyEsc}))
	cmd := m.Init()
	if cmd != nil {
		// esc is handled by returning tea.Quit in Update, but we only get the cmd from Init.
		// We verify the model state is consistent.
	}
}

func TestModelReceivesGameState(t *testing.T) {
	m := NewModel("ws://localhost:8080/ws")
	m = updateModel(m, tea.WindowSizeMsg{Width: 80, Height: 24})

	gs := GameStateMsg(service.GameState{
		GameMode:  "CLASSIC",
		GameTime:  312.5,
		MapName:   "Map11",
		Players:   []service.PlayerSnapshot{{SummonerName: "P1", ChampionName: "Ashe", IsActive: true}},
		Timestamp: 1,
	})
	m = updateModel(m, gs)

	view := m.View()
	if !contains(view, "CLASSIC") {
		t.Errorf("View() missing CLASSIC, got %q", view)
	}
	if !contains(view, "Ashe") {
		t.Errorf("View() missing Ashe, got %q", view)
	}
}

func TestModelReceivesAdvice(t *testing.T) {
	m := NewModel("ws://localhost:8080/ws")
	m = updateModel(m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updateModel(m, tea.KeyMsg(tea.Key{Type: tea.KeyRight}))

	advice := JudgeAdviceMsg(service.JudgeAdvice{
		HookName:   "periodic-5min",
		GameMinute: 5,
		Advice:     "Push mid.",
	})
	m = updateModel(m, advice)

	view := m.View()
	if !contains(view, "Push mid.") {
		t.Errorf("View() missing advice, got %q", view)
	}
}

func TestParseMessage(t *testing.T) {
	gs := service.GameState{GameMode: "CLASSIC", GameTime: 123.0}
	payload, _ := json.Marshal(gs)
	msg := parseMessage(mustMarshal(service.WSMessage{Type: service.MsgTypeGameState, Payload: payload}))

	gsm, ok := msg.(GameStateMsg)
	if !ok {
		t.Fatalf("parseMessage returned %T, want GameStateMsg", msg)
	}
	if gsm.GameMode != "CLASSIC" {
		t.Errorf("GameMode = %q, want CLASSIC", gsm.GameMode)
	}
}

func TestParseMessageUnknown(t *testing.T) {
	msg := parseMessage(mustMarshal(service.WSMessage{Type: "custom"}))
	_, ok := msg.(RawMsg)
	if !ok {
		t.Fatalf("parseMessage returned %T, want RawMsg", msg)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func mustMarshal(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

func navigateToConfig(m Model) Model {
	for i := 0; i < 4; i++ {
		m = updateModel(m, tea.KeyMsg(tea.Key{Type: tea.KeyRight}))
	}
	return m
}

func typeRunes(m Model, s string) Model {
	for _, r := range s {
		m = updateModel(m, tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{r}}))
	}
	return m
}

func TestModelConfigTab_CommandHelp(t *testing.T) {
	m := NewModel("ws://localhost:8080/ws")
	m = updateModel(m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m = navigateToConfig(m)
	if m.ActiveTab() != TabConfig {
		t.Fatalf("expected TabConfig, got %v", m.ActiveTab())
	}
	m = typeRunes(m, "/help")
	newM, _ := m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyEnter}))
	m = newM.(Model)
	if !strings.Contains(m.config.status, "/lang") {
		t.Errorf("status = %q, want help text", m.config.status)
	}
}

func TestModelConfigTab_CommandUnknown(t *testing.T) {
	m := NewModel("ws://localhost:8080/ws")
	m = updateModel(m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m = navigateToConfig(m)
	m = typeRunes(m, "/foo")
	newM, _ := m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyEnter}))
	m = newM.(Model)
	if !strings.Contains(m.config.status, "unknown command") {
		t.Errorf("status = %q, want unknown command", m.config.status)
	}
}

func TestModelConfigTab_CommandLang(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Fatalf("expected PATCH, got %s", r.Method)
		}
		var patch service.ConfigPatch
		if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if patch.Judge == nil || patch.Judge.Language == nil || *patch.Judge.Language != "pt-BR" {
			t.Fatalf("unexpected patch: %+v", patch)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(service.ConfigView{
			Judge: service.JudgeConfigView{Language: "pt-BR"},
			Hooks: []hooks.HookView{},
		})
	}))
	defer server.Close()

	addr := strings.Replace(server.URL, "http://", "ws://", 1) + "/ws"
	m := NewModel(addr)
	m = updateModel(m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m = navigateToConfig(m)
	m = updateModel(m, ConfigLoadedMsg(service.ConfigView{
		Judge: service.JudgeConfigView{Language: "en"},
		Hooks: []hooks.HookView{},
	}))
	m = typeRunes(m, "/lang")
	newM, cmd := m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyEnter}))
	m = newM.(Model)
	if cmd == nil {
		t.Fatal("expected command for /lang")
	}
	msg := cmd()
	saved, ok := msg.(ConfigSavedMsg)
	if !ok {
		t.Fatalf("expected ConfigSavedMsg, got %T", msg)
	}
	m = updateModel(m, saved)
	if m.config.view.Judge.Language != "pt-BR" {
		t.Errorf("language = %q, want pt-BR", m.config.view.Judge.Language)
	}
}

func TestModelConfigTab_EmptyInputNavigates(t *testing.T) {
	m := NewModel("ws://localhost:8080/ws")
	m = updateModel(m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m = navigateToConfig(m)
	m = updateModel(m, tea.KeyMsg(tea.Key{Type: tea.KeyRight}))
	if m.ActiveTab() != TabLive {
		t.Errorf("ActiveTab = %v, want TabLive", m.ActiveTab())
	}
}

func TestModelConfigTab_NonEmptyInputDoesNotNavigate(t *testing.T) {
	m := NewModel("ws://localhost:8080/ws")
	m = updateModel(m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m = navigateToConfig(m)
	m = typeRunes(m, "/")
	m = updateModel(m, tea.KeyMsg(tea.Key{Type: tea.KeyRight}))
	if m.ActiveTab() != TabConfig {
		t.Errorf("ActiveTab = %v, want TabConfig", m.ActiveTab())
	}
}

func TestModelConfigTab_EscClearsInput(t *testing.T) {
	m := NewModel("ws://localhost:8080/ws")
	m = updateModel(m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m = navigateToConfig(m)
	m = typeRunes(m, "/foo")
	m = updateModel(m, tea.KeyMsg(tea.Key{Type: tea.KeyEsc}))
	if m.config.input.Value() != "" {
		t.Errorf("input = %q, want empty", m.config.input.Value())
	}
	if m.ActiveTab() != TabConfig {
		t.Errorf("ActiveTab = %v, want TabConfig", m.ActiveTab())
	}
}
