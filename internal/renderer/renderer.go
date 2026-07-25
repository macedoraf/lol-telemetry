// Package renderer implements the Bubble Tea TUI dashboard for active player stats.
package renderer

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"lol-telemetry/internal/types"
)

// Model is the Bubble Tea model for the telemetry dashboard.
type Model struct {
	state types.DashboardState
}

// tickMsg is used internally to keep the UI responsive.
type tickMsg struct{}

// NewModel creates a fresh dashboard model.
func NewModel() Model {
	return Model{}
}

// Init satisfies the tea.Model interface.
func (m Model) Init() tea.Cmd {
	return tick()
}

// Update satisfies the tea.Model interface.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case types.DashboardState:
		m.state = msg
		return m, tick()
	case tickMsg:
		return m, tick()
	}
	return m, nil
}

// View renders the dashboard.
func (m Model) View() string {
	if m.state.Waiting {
		return m.waitingView()
	}
	if m.state.Error != "" {
		return m.errorView()
	}
	return m.statsView()
}

func (m Model) statsView() string {
	s := m.state.Stats
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF6B6B")).Render("LoL Telemetry")
	section := lipgloss.NewStyle().MarginLeft(1).MarginTop(1)

	lines := []string{
		title,
		section.Render(fmt.Sprintf("Summoner: %s", s.SummonerName)),
		section.Render(fmt.Sprintf("Champion: %s", s.ChampionName)),
		section.Render(fmt.Sprintf("Level:    %d", s.Level)),
		section.Render(fmt.Sprintf("Gold:     %.0f", s.CurrentGold)),
		section.Render(fmt.Sprintf("CS/Min:   %.2f", s.CSPerMin)),
		section.Render(fmt.Sprintf("GPM:      %.2f", s.GPM)),
		section.Render(fmt.Sprintf("Game:     %.1fs", s.GameTime)),
		section.Render("Press q to quit"),
	}
	return strings.Join(lines, "\n") + "\n"
}

func (m Model) waitingView() string {
	msg := lipgloss.NewStyle().Foreground(lipgloss.Color("#F0E68C")).Render("Waiting for League of Legends match...")
	return msg + "\nPress q to quit\n"
}

func (m Model) errorView() string {
	msg := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Render(fmt.Sprintf("Error: %s", m.state.Error))
	return msg + "\nPress q to quit\n"
}

func tick() tea.Cmd {
	return tea.Tick(updateInterval, func(_ time.Time) tea.Msg {
		return tickMsg{}
	})
}

// updateInterval is the polling interval used for the UI tick.
const updateInterval = 1 * time.Second
