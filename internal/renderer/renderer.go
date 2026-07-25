// Package renderer implements the Bubble Tea debugger TUI for the Live Client
// Data API. It displays a route list, connection status, and a scrollable
// viewport of raw JSON responses.
package renderer

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"lol-telemetry/pkg/riotclient"
)

// route describes a pollable API endpoint and how to fetch its raw JSON.
type route struct {
	name  string
	fetch func(*riotclient.Client) (string, error)
}

// fetchMsg carries the result (or error) of a single route fetch.
type fetchMsg struct {
	route   string
	content string
	err     error
}

// tickMsg triggers the next periodic fetch.
type tickMsg struct{}

// Model is the Bubble Tea model for the debugger TUI.
type Model struct {
	client   *riotclient.Client
	routes   []route
	cursor   int
	focused  string // "sidebar" or "viewport"
	online   bool
	content  string
	lastErr  string
	viewport viewport.Model
	width    int
	height   int
	ready    bool
}

// NewDebugger creates a new debugger model wired to the given client.
func NewDebugger(client *riotclient.Client) Model {
	m := Model{
		client:  client,
		focused: "sidebar",
		routes:  defaultRoutes(),
	}
	m.viewport = viewport.New(0, 0)
	m.viewport.SetContent(m.placeholderContent())
	return m
}

// NewDebuggerWithRoutes creates a model with custom routes (used for testing).
func NewDebuggerWithRoutes(client *riotclient.Client, routes []route) Model {
	m := NewDebugger(client)
	m.routes = routes
	return m
}

func defaultRoutes() []route {
	return []route{
		{"allgamedata", func(c *riotclient.Client) (string, error) {
			data, err := c.GetGameData()
			if err != nil {
				return "", err
			}
			b, err := json.MarshalIndent(data, "", "  ")
			return string(b), err
		}},
		{"activeplayername", func(c *riotclient.Client) (string, error) {
			name, err := c.GetActivePlayerName()
			if err != nil {
				return "", err
			}
			b, err := json.MarshalIndent(name, "", "  ")
			return string(b), err
		}},
		{"activeplayerabilities", func(c *riotclient.Client) (string, error) {
			data, err := c.GetActivePlayerAbilities()
			if err != nil {
				return "", err
			}
			b, err := json.MarshalIndent(data, "", "  ")
			return string(b), err
		}},
		{"activeplayerrunes", func(c *riotclient.Client) (string, error) {
			data, err := c.GetActivePlayerRunes()
			if err != nil {
				return "", err
			}
			b, err := json.MarshalIndent(data, "", "  ")
			return string(b), err
		}},
		{"playerlist", func(c *riotclient.Client) (string, error) {
			data, err := c.GetPlayerList()
			if err != nil {
				return "", err
			}
			b, err := json.MarshalIndent(data, "", "  ")
			return string(b), err
		}},
		{"eventdata", func(c *riotclient.Client) (string, error) {
			data, err := c.GetEventData()
			if err != nil {
				return "", err
			}
			b, err := json.MarshalIndent(data, "", "  ")
			return string(b), err
		}},
		{"gamestats", func(c *riotclient.Client) (string, error) {
			data, err := c.GetGameStats()
			if err != nil {
				return "", err
			}
			b, err := json.MarshalIndent(data, "", "  ")
			return string(b), err
		}},
	}
}

// Init satisfies the tea.Model interface.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.tick(), m.fetch())
}

// Update satisfies the tea.Model interface.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.layout()
		m.ready = true

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		case "tab":
			if m.focused == "sidebar" {
				m.focused = "viewport"
			} else {
				m.focused = "sidebar"
			}
		case "up", "k":
			if m.focused == "sidebar" && m.cursor > 0 {
				m.cursor--
				cmds = append(cmds, m.fetch())
			}
		case "down", "j":
			if m.focused == "sidebar" && m.cursor < len(m.routes)-1 {
				m.cursor++
				cmds = append(cmds, m.fetch())
			}
		case "enter":
			if m.focused == "sidebar" {
				m.focused = "viewport"
				cmds = append(cmds, m.fetch())
			}
		case "pgup", "ctrl+u":
			if m.focused == "viewport" {
				m.viewport.LineUp(5)
			}
		case "pgdown", "ctrl+d":
			if m.focused == "viewport" {
				m.viewport.LineDown(5)
			}
		}

	case tickMsg:
		cmds = append(cmds, m.fetch())
		cmds = append(cmds, m.tick())

	case fetchMsg:
		// Ignore stale responses for a route that is no longer selected.
		if msg.route != m.currentRoute().name {
			return m, nil
		}
		if msg.err != nil {
			m.online = false
			m.lastErr = msg.err.Error()
			m.content = ""
			m.viewport.SetContent(m.errorView())
		} else {
			m.online = true
			m.lastErr = ""
			m.content = msg.content
			m.viewport.SetContent(msg.content)
			m.viewport.GotoTop()
		}
	}

	// Let the viewport handle its own internal messages (e.g., mouse wheel).
	var vpCmd tea.Cmd
	m.viewport, vpCmd = m.viewport.Update(msg)
	cmds = append(cmds, vpCmd)

	return m, tea.Batch(cmds...)
}

// View satisfies the tea.Model interface.
func (m Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	status := m.statusBar()
	sidebar := m.sidebarView()
	viewport := m.viewport.View()

	bodyHeight := m.height - lipgloss.Height(status) - 1
	if bodyHeight < 1 {
		bodyHeight = 1
	}

	// Ensure the viewport is sized correctly for the current body area.
	m.viewport.Width = m.width - sidebarWidth - 3
	m.viewport.Height = bodyHeight

	body := lipgloss.JoinHorizontal(
		lipgloss.Top,
		sidebar,
		viewportStyle.Render(viewport),
	)

	return status + "\n" + body
}

// currentRoute returns the currently selected route.
func (m Model) currentRoute() route {
	return m.routes[m.cursor]
}

// fetch returns a command that fetches the currently selected route.
func (m Model) fetch() tea.Cmd {
	r := m.currentRoute()
	return func() tea.Msg {
		content, err := r.fetch(m.client)
		return fetchMsg{route: r.name, content: content, err: err}
	}
}

// tick returns a command that waits for the configured polling interval.
func (m Model) tick() tea.Cmd {
	return tea.Tick(pollInterval, func(_ time.Time) tea.Msg {
		return tickMsg{}
	})
}

// layout recalculates viewport/sidebar dimensions.
func (m *Model) layout() {
	bodyHeight := m.height - 2
	if bodyHeight < 1 {
		bodyHeight = 1
	}
	m.viewport.Width = m.width - sidebarWidth - 3
	m.viewport.Height = bodyHeight
}

// sidebarView renders the route list.
func (m Model) sidebarView() string {
	var lines []string
	for i, r := range m.routes {
		item := fmt.Sprintf(" %s", r.name)
		if i == m.cursor {
			item = cursorStyle.Render("> " + r.name)
		} else if m.focused == "sidebar" {
			item = sidebarStyle.Render(item)
		} else {
			item = sidebarBlurStyle.Render(item)
		}
		lines = append(lines, item)
	}
	return lipgloss.NewStyle().
		Width(sidebarWidth).
		Height(m.height - 2).
		Border(lipgloss.NormalBorder()).
		BorderRight(true).
		Render(strings.Join(lines, "\n"))
}

// statusBar renders the online/offline indicator and help text.
func (m Model) statusBar() string {
	status := "OFFLINE"
	color := offlineColor
	if m.online {
		status = "ONLINE"
		color = onlineColor
	}

	indicator := lipgloss.NewStyle().
		Bold(true).
		Foreground(color).
		Render("● " + status)

	help := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		Render("↑/↓/j/k navigate • Tab switch focus • q/Esc quit")

	return lipgloss.JoinHorizontal(
		lipgloss.Center,
		indicator,
		lipgloss.NewStyle().Width(m.width-50).Render(""),
		help,
	)
}

// errorView returns the viewport content when a fetch fails.
func (m Model) errorView() string {
	msg := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FF0000")).
		Render("Error: " + m.lastErr)
	return msg + "\n\n" +
		lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Render("The Live Client Data API is unreachable. Ensure League of Legends is running in-game and try again.")
}

// placeholderContent returns the initial viewport text.
func (m Model) placeholderContent() string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		Render("Select a route and press Enter to inspect the raw JSON response.")
}

// Exported getters for tests.

// CurrentRouteName returns the name of the currently selected route.
func (m Model) CurrentRouteName() string {
	return m.currentRoute().name
}

// IsOnline reports the current connection status.
func (m Model) IsOnline() bool {
	return m.online
}

// CurrentContent returns the last successfully fetched content.
func (m Model) CurrentContent() string {
	return m.content
}

// LastError returns the last fetch error message (empty if none).
func (m Model) LastError() string {
	return m.lastErr
}

// Focus returns the current focus area.
func (m Model) Focus() string {
	return m.focused
}

const (
	pollInterval = 1 * time.Second
	sidebarWidth = 26
	onlineColor  = lipgloss.Color("#00FF00")
	offlineColor = lipgloss.Color("#FF0000")
)

var (
	cursorStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#5A56E0"))

	sidebarStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF"))

	sidebarBlurStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888"))

	viewportStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		Padding(0, 1)
)
