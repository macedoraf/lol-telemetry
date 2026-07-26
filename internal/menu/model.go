// Package menu implements the main feature menu for the lol-cli TUI.
package menu

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Choice identifiers used by the menu and consumers.
const (
	ChoiceRoutes = "routes"
	ChoiceTips   = "tips"
)

// SelectMsg is emitted when the user confirms a menu option.
type SelectMsg struct {
	Choice string
}

// Model is the Bubble Tea model for the feature menu.
type Model struct {
	choices []choice
	cursor  int
	width   int
	height  int
	ready   bool
}

type choice struct {
	label string
	value string
}

// NewModel creates a new menu model with the default feature options.
func NewModel() Model {
	return Model{
		choices: []choice{
			{label: "Rotas do SDK", value: ChoiceRoutes},
			{label: "Dicas do Jogo", value: ChoiceTips},
		},
	}
}

// Init satisfies the tea.Model interface.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update satisfies the tea.Model interface.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
		case "enter":
			return m, func() tea.Msg {
				return SelectMsg{Choice: m.choices[m.cursor].value}
			}
		}
	}

	return m, nil
}

// View satisfies the tea.Model interface.
func (m Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	title := titleStyle.Render("lol-telemetry — Features")

	var items []string
	for i, c := range m.choices {
		item := fmt.Sprintf("  %s", c.label)
		if i == m.cursor {
			item = cursorStyle.Render("> " + c.label)
		} else {
			item = itemStyle.Render(item)
		}
		items = append(items, item)
	}

	help := helpStyle.Render("↑/↓/j/k navigate • Enter select • q/Esc quit")

	body := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		strings.Join(items, "\n"),
		"",
		help,
	)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		body,
	)
}

// CurrentChoice returns the value of the currently selected menu item.
func (m Model) CurrentChoice() string {
	return m.choices[m.cursor].value
}

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#5A56E0")).
			MarginBottom(1)

	cursorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#5A56E0"))

	itemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			MarginTop(1)
)
