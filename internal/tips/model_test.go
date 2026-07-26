package tips

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"lol-telemetry/internal/config"
)

func updateModel(m Model, msg tea.Msg) Model {
	newM, _ := m.Update(msg)
	return newM.(Model)
}

func keyMsg(t *testing.T, key string) tea.KeyMsg {
	t.Helper()
	switch key {
	case "esc":
		return tea.KeyMsg(tea.Key{Type: tea.KeyEsc})
	case "ctrl+c":
		return tea.KeyMsg(tea.Key{Type: tea.KeyCtrlC})
	default:
		return tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune(key)})
	}
}

func TestTipsViewShowsWaitingMessage(t *testing.T) {
	m := NewModel(config.EnvConfig{})
	m = updateModel(m, tea.WindowSizeMsg{Width: 80, Height: 24})

	view := m.View()
	if !strings.Contains(view, "Aguardando dica do Judge...") {
		t.Errorf("View() missing waiting message, got %q", view)
	}
	if !strings.Contains(view, "desativado") {
		t.Errorf("View() missing disabled status, got %q", view)
	}
}

func TestTipsViewShowsDisabledHint(t *testing.T) {
	m := NewModel(config.EnvConfig{})
	m = updateModel(m, tea.WindowSizeMsg{Width: 80, Height: 24})

	view := m.View()
	if !strings.Contains(view, "Configure OPENROUTER_API_KEY") {
		t.Errorf("View() missing setup hint, got %q", view)
	}
}

func TestTipsUpdateAdvice(t *testing.T) {
	m := NewModel(config.EnvConfig{APIKey: "sk-secret", Model: "openai/gpt-4o-mini"})
	m = updateModel(m, tea.WindowSizeMsg{Width: 80, Height: 24})

	m = updateModel(m, UpdateAdviceMsg{Advice: "Push mid."})
	if got := m.CurrentAdvice(); got != "Push mid." {
		t.Errorf("CurrentAdvice() = %q, want Push mid.", got)
	}

	view := m.View()
	if !strings.Contains(view, "Push mid.") {
		t.Errorf("View() should display advice, got %q", view)
	}
}

func TestTipsBackKeys(t *testing.T) {
	m := NewModel(config.EnvConfig{})
	m = updateModel(m, tea.WindowSizeMsg{Width: 80, Height: 24})

	for _, key := range []string{"q", "esc", "ctrl+c"} {
		_, cmd := m.Update(keyMsg(t, key))
		if cmd == nil {
			t.Errorf("key %q: expected BackMsg command, got nil", key)
			continue
		}
		msg := cmd()
		if _, ok := msg.(BackMsg); !ok {
			t.Errorf("key %q: expected BackMsg, got %T", key, msg)
		}
	}
}

func TestTipsViewMasksAPIKey(t *testing.T) {
	m := NewModel(config.EnvConfig{APIKey: "sk-1234567890abcdef", Model: "openai/gpt-4o-mini"})
	m = updateModel(m, tea.WindowSizeMsg{Width: 80, Height: 24})

	view := m.View()
	if strings.Contains(view, "sk-1234567890abcdef") {
		t.Errorf("View() should not expose raw key, got %q", view)
	}
	if !strings.Contains(view, "...cdef") {
		t.Errorf("View() should show masked suffix, got %q", view)
	}
}
