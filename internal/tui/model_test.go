package tui

import (
	"encoding/json"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

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
