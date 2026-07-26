// Package tips implements the Bubble Tea panel that displays the latest Judge
// advice and the current environment configuration.
package tips

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"lol-telemetry/internal/config"
)

// UpdateAdviceMsg carries a new Judge advice to be displayed.
type UpdateAdviceMsg struct {
	Advice string
}

// BackMsg is emitted when the user asks to return to the main menu.
type BackMsg struct{}

// Model is the Bubble Tea model for the game tips panel.
type Model struct {
	cfg    config.EnvConfig
	advice string
	width  int
	height int
	ready  bool
}

// NewModel creates a new tips panel with the given configuration.
func NewModel(cfg config.EnvConfig) Model {
	return Model{cfg: cfg}
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

	case UpdateAdviceMsg:
		m.advice = msg.Advice

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, func() tea.Msg { return BackMsg{} }
		}
	}

	return m, nil
}

// View satisfies the tea.Model interface.
func (m Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	title := titleStyle.Render("Dicas do Jogo")

	advice := m.advice
	if advice == "" {
		advice = "Aguardando dica do Judge..."
	}
	adviceBox := adviceStyle.Render("Última dica:\n" + advice)

	status := statusStyle.Render(m.cfg.String())

	if !m.cfg.Enabled() {
		status = statusStyle.Render(m.cfg.String() + "\n\nConfigure OPENROUTER_API_KEY para ativar as dicas.")
	}

	help := helpStyle.Render("Esc/q voltar")

	body := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		adviceBox,
		"",
		status,
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

// CurrentAdvice returns the most recent advice shown in the panel.
func (m Model) CurrentAdvice() string {
	return m.advice
}

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#5A56E0")).
			MarginBottom(1)

	adviceStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			Padding(1, 2).
			Width(60)

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Padding(1, 2).
			Width(60)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			MarginTop(1)
)

func placeholderContent() string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		Render("Aguardando dica do Judge...")
}

// avoid unused import warning if lipgloss strings is not used elsewhere.
var _ = strings.Join
