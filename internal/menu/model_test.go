package menu

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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
	case "esc":
		return tea.KeyMsg(tea.Key{Type: tea.KeyEsc})
	case "ctrl+c":
		return tea.KeyMsg(tea.Key{Type: tea.KeyCtrlC})
	default:
		return tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune(key)})
	}
}

func TestMenuViewShowsChoices(t *testing.T) {
	m := NewModel()
	m = updateModel(m, tea.WindowSizeMsg{Width: 80, Height: 24})

	view := m.View()
	for _, want := range []string{"Rotas do SDK", "Dicas do Jogo"} {
		if !strings.Contains(view, want) {
			t.Errorf("View() missing %q", want)
		}
	}
}

func TestMenuNavigation(t *testing.T) {
	m := NewModel()
	m = updateModel(m, tea.WindowSizeMsg{Width: 80, Height: 24})

	if got := m.CurrentChoice(); got != ChoiceRoutes {
		t.Fatalf("initial choice = %q, want %q", got, ChoiceRoutes)
	}

	m = updateModel(m, keyMsg(t, "down"))
	if got := m.CurrentChoice(); got != ChoiceTips {
		t.Errorf("after down choice = %q, want %q", got, ChoiceTips)
	}

	m = updateModel(m, keyMsg(t, "up"))
	if got := m.CurrentChoice(); got != ChoiceRoutes {
		t.Errorf("after up choice = %q, want %q", got, ChoiceRoutes)
	}
}

func TestMenuJAndKNavigation(t *testing.T) {
	m := NewModel()
	m = updateModel(m, tea.WindowSizeMsg{Width: 80, Height: 24})

	m = updateModel(m, keyMsg(t, "j"))
	if got := m.CurrentChoice(); got != ChoiceTips {
		t.Errorf("after j choice = %q, want %q", got, ChoiceTips)
	}

	m = updateModel(m, keyMsg(t, "k"))
	if got := m.CurrentChoice(); got != ChoiceRoutes {
		t.Errorf("after k choice = %q, want %q", got, ChoiceRoutes)
	}
}

func TestMenuSelectEmitsMessage(t *testing.T) {
	m := NewModel()
	m = updateModel(m, tea.WindowSizeMsg{Width: 80, Height: 24})

	_, cmd := m.Update(keyMsg(t, "enter"))
	if cmd == nil {
		t.Fatal("enter did not produce a command")
	}

	msg := cmd()
	selectMsg, ok := msg.(SelectMsg)
	if !ok {
		t.Fatalf("expected SelectMsg, got %T", msg)
	}
	if selectMsg.Choice != ChoiceRoutes {
		t.Errorf("selectMsg.Choice = %q, want %q", selectMsg.Choice, ChoiceRoutes)
	}
}

func TestMenuQuitKeys(t *testing.T) {
	m := NewModel()
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

func TestMenuCursorStaysWithinBounds(t *testing.T) {
	m := NewModel()
	m = updateModel(m, tea.WindowSizeMsg{Width: 80, Height: 24})

	m = updateModel(m, keyMsg(t, "up"))
	if got := m.CurrentChoice(); got != ChoiceRoutes {
		t.Errorf("up from first should stay at %q, got %q", ChoiceRoutes, got)
	}

	m = updateModel(m, keyMsg(t, "down"))
	m = updateModel(m, keyMsg(t, "down"))
	if got := m.CurrentChoice(); got != ChoiceTips {
		t.Errorf("down past last should stay at %q, got %q", ChoiceTips, got)
	}
}
