// Package tui implements the Bubble Tea interface for the lol-telemetry daemon client.
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"lol-telemetry/pkg/service"
)

// Tab identifies the active TUI panel.
type Tab int

const (
	TabLive Tab = iota
	TabAdvice
	TabEvents
	TabLog
)

const tabCount = 4

// Model is the top-level Bubble Tea model for the daemon-connected CLI.
type Model struct {
	client     *WSClient
	connected  bool
	lastErr    string
	activeTab  Tab
	width      int
	height     int
	live       LiveState
	advice     AdviceState
	events     EventsState
	log        LogState
}

// NewModel creates a new TUI model wired to the given WebSocket address.
func NewModel(wsAddr string) Model {
	return Model{
		client: NewWSClient(wsAddr),
	}
}

// Init starts the connection and the first read loop.
func (m Model) Init() tea.Cmd {
	return m.client.Connect(nil)
}

// Update handles user input and daemon messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		case "right", "l", "tab":
			m.activeTab = (m.activeTab + 1) % tabCount
		case "left", "h":
			m.activeTab = (m.activeTab - 1 + tabCount) % tabCount
		}
		return m, nil

	case ConnectedMsg:
		m.connected = true
		m.lastErr = ""
		return m, m.client.Read()

	case DisconnectedMsg:
		m.connected = false
		m.lastErr = msg.Error.Error()
		return m, nil

	case GameStateMsg:
		m.live.Update(service.GameState(msg))
		m.advice.SetGameMinute(service.GameState(msg).GameMinute())
		return m, m.client.Read()

	case JudgeAdviceMsg:
		m.advice.Update(service.JudgeAdvice(msg))
		return m, m.client.Read()

	case EventMsg:
		m.events.Update(service.EventMessage(msg))
		return m, m.client.Read()

	case RawMsg:
		m.log.Update(service.WSMessage(msg))
		return m, m.client.Read()
	}

	return m, nil
}

// View renders the full TUI.
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing..."
	}

	status := m.renderStatus()
	tabs := m.renderTabs()

	var body string
	switch m.activeTab {
	case TabLive:
		body = m.live.View(m.width, m.height-lipgloss.Height(status)-lipgloss.Height(tabs)-2)
	case TabAdvice:
		body = m.advice.View(m.width, m.height-lipgloss.Height(status)-lipgloss.Height(tabs)-2)
	case TabEvents:
		body = m.events.View(m.width, m.height-lipgloss.Height(status)-lipgloss.Height(tabs)-2)
	case TabLog:
		body = m.log.View(m.width, m.height-lipgloss.Height(status)-lipgloss.Height(tabs)-2)
	}

	return strings.Join([]string{status, "", tabs, "", body}, "\n")
}

func (m Model) renderStatus() string {
	status := "OFFLINE"
	color := offlineColor
	if m.connected {
		status = "ONLINE"
		color = onlineColor
	}

	indicator := lipgloss.NewStyle().Bold(true).Foreground(color).Render("● " + status)
	addr := addrStyle.Render(m.client.addr)
	help := helpStyle.Render("←/→ tab • q quit")

	parts := []string{indicator, addr}
	if m.lastErr != "" && !m.connected {
		parts = append(parts, errorStyle.Render(m.lastErr))
	}
	parts = append(parts, help)

	return lipgloss.JoinHorizontal(lipgloss.Center, parts...)
}

func (m Model) renderTabs() string {
	names := []string{"Live", "Advice", "Events", "Log"}
	var parts []string
	for i, name := range names {
		if Tab(i) == m.activeTab {
			parts = append(parts, activeTabStyle.Render(" "+name+" "))
		} else {
			parts = append(parts, tabStyle.Render(" "+name+" "))
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Left, parts...)
}

// LastError returns the last connection error for tests.
func (m Model) LastError() string {
	return m.lastErr
}

// ActiveTab returns the current tab for tests.
func (m Model) ActiveTab() Tab {
	return m.activeTab
}

var (
	onlineColor  = lipgloss.Color("#00FF00")
	offlineColor = lipgloss.Color("#FF0000")

	addrStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).MarginLeft(2)
	helpStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).MarginLeft(2)
	errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555")).MarginLeft(2)

	tabStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#333333"))
	activeTabStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#5A56E0")).Bold(true)
)

// Sprintf helper avoids unused import when iterating.
var _ = fmt.Sprintf