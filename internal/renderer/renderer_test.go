package renderer

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"lol-telemetry/pkg/riotclient"
)

func updateModel(m Model, msg tea.Msg) Model {
	newM, _ := m.Update(msg)
	return newM.(Model)
}

func keyMsg(t *testing.T, key string) tea.KeyMsg {
	t.Helper()
	switch key {
	case "down":
		return tea.KeyMsg(tea.Key{Type: tea.KeyDown})
	case "up":
		return tea.KeyMsg(tea.Key{Type: tea.KeyUp})
	case "tab":
		return tea.KeyMsg(tea.Key{Type: tea.KeyTab})
	case "esc":
		return tea.KeyMsg(tea.Key{Type: tea.KeyEsc})
	case "ctrl+c":
		return tea.KeyMsg(tea.Key{Type: tea.KeyCtrlC})
	default:
		return tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune(key)})
	}
}

func TestDebuggerViewShowsRoutes(t *testing.T) {
	m := NewDebuggerWithRoutes(nil, []route{
		{name: "allgamedata", fetch: func(*riotclient.Client) (string, error) { return "{}", nil }},
		{name: "eventdata", fetch: func(*riotclient.Client) (string, error) { return "{}", nil }},
	})
	m = updateModel(m, tea.WindowSizeMsg{Width: 80, Height: 24})

	view := m.View()
	for _, want := range []string{"allgamedata", "eventdata", "OFFLINE", "●"} {
		if !strings.Contains(view, want) {
			t.Errorf("View() missing %q", want)
		}
	}
}

func TestDebuggerNavigation(t *testing.T) {
	m := NewDebuggerWithRoutes(nil, []route{
		{name: "route1", fetch: func(*riotclient.Client) (string, error) { return "route1-body", nil }},
		{name: "route2", fetch: func(*riotclient.Client) (string, error) { return "route2-body", nil }},
	})
	m = updateModel(m, tea.WindowSizeMsg{Width: 80, Height: 24})

	if got := m.CurrentRouteName(); got != "route1" {
		t.Fatalf("initial route = %q, want route1", got)
	}

	m = updateModel(m, keyMsg(t, "down"))
	if got := m.CurrentRouteName(); got != "route2" {
		t.Errorf("after down route = %q, want route2", got)
	}

	m = updateModel(m, keyMsg(t, "up"))
	if got := m.CurrentRouteName(); got != "route1" {
		t.Errorf("after up route = %q, want route1", got)
	}
}

func TestDebuggerFocusSwitch(t *testing.T) {
	m := NewDebuggerWithRoutes(nil, []route{
		{name: "route1", fetch: func(*riotclient.Client) (string, error) { return "body", nil }},
	})
	m = updateModel(m, tea.WindowSizeMsg{Width: 80, Height: 24})

	if got := m.Focus(); got != "sidebar" {
		t.Fatalf("initial focus = %q, want sidebar", got)
	}

	m = updateModel(m, keyMsg(t, "tab"))
	if got := m.Focus(); got != "viewport" {
		t.Errorf("after tab focus = %q, want viewport", got)
	}
}

func TestDebuggerQuitKeys(t *testing.T) {
	m := NewDebuggerWithRoutes(nil, []route{
		{name: "route1", fetch: func(*riotclient.Client) (string, error) { return "body", nil }},
	})
	m = updateModel(m, tea.WindowSizeMsg{Width: 80, Height: 24})

	for _, key := range []string{"q", "esc", "ctrl+c"} {
		_, cmd := m.Update(keyMsg(t, key))
		if cmd == nil {
			t.Errorf("key %q: expected tea.Quit command, got nil", key)
			continue
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Errorf("key %q: expected tea.QuitMsg, got another message", key)
		}
	}
}

func TestDebuggerFetchResultUpdatesContent(t *testing.T) {
	m := NewDebuggerWithRoutes(nil, []route{
		{name: "route1", fetch: func(*riotclient.Client) (string, error) { return "route1-body", nil }},
	})
	m = updateModel(m, tea.WindowSizeMsg{Width: 80, Height: 24})

	m = updateModel(m, fetchMsg{route: "route1", content: "{\"ok\":true}"})
	if !m.IsOnline() {
		t.Errorf("IsOnline() = false, want true")
	}
	if got := m.CurrentContent(); got != "{\"ok\":true}" {
		t.Errorf("CurrentContent() = %q, want {\"ok\":true}", got)
	}
	if got := m.LastError(); got != "" {
		t.Errorf("LastError() = %q, want empty", got)
	}
}

func TestDebuggerFetchErrorMarksOffline(t *testing.T) {
	m := NewDebuggerWithRoutes(nil, []route{
		{name: "route1", fetch: func(*riotclient.Client) (string, error) { return "", nil }},
	})
	m = updateModel(m, tea.WindowSizeMsg{Width: 80, Height: 24})

	m = updateModel(m, fetchMsg{route: "route1", err: errors.New("connection refused")})
	if m.IsOnline() {
		t.Errorf("IsOnline() = true, want false")
	}
	if got := m.LastError(); got != "connection refused" {
		t.Errorf("LastError() = %q, want connection refused", got)
	}
}

func TestDebuggerStaleFetchResultIgnored(t *testing.T) {
	m := NewDebuggerWithRoutes(nil, []route{
		{name: "route1", fetch: func(*riotclient.Client) (string, error) { return "", nil }},
		{name: "route2", fetch: func(*riotclient.Client) (string, error) { return "", nil }},
	})
	m = updateModel(m, tea.WindowSizeMsg{Width: 80, Height: 24})

	m = updateModel(m, keyMsg(t, "down")) // now route2
	m = updateModel(m, fetchMsg{route: "route1", content: "stale"})

	if got := m.CurrentContent(); got != "" {
		t.Errorf("stale result should be ignored, got content %q", got)
	}
}

func TestDebuggerTickProducesCommands(t *testing.T) {
	m := NewDebuggerWithRoutes(nil, []route{
		{name: "route1", fetch: func(*riotclient.Client) (string, error) { return "", nil }},
	})
	m = updateModel(m, tea.WindowSizeMsg{Width: 80, Height: 24})

	_, cmd := m.Update(tickMsg{})
	if cmd == nil {
		t.Fatal("tickMsg did not produce a command")
	}
	// A tick should schedule both a fetch and the next tick. The exact
	// command is a tea.Batch; just verify the command is non-nil.
	msg := cmd()
	if msg == nil {
		t.Error("tickMsg command returned nil message")
	}
}
