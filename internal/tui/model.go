// Package tui implements the Bubble Tea interface for the lol-telemetry daemon client.
package tui

import (
	"fmt"
	"os"
	"os/exec"
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
	TabConfig
)

const tabCount = 5

// Model is the top-level Bubble Tea model for the daemon-connected CLI.
type Model struct {
	client       *WSClient
	configClient *ConfigClient
	connected    bool
	lastErr      string
	activeTab    Tab
	width        int
	height       int
	live         LiveState
	advice       AdviceState
	events       EventsState
	log          LogState
	config       ConfigState
}

// NewModel creates a new TUI model wired to the given WebSocket address.
func NewModel(wsAddr string) Model {
	return Model{
		client:       NewWSClient(wsAddr),
		configClient: NewConfigClient(wsAddr),
		config:       newConfigState(),
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
		keyStr := msg.String()
		switch keyStr {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "esc":
			if m.activeTab == TabConfig && m.config.input.Value() != "" {
				m.config.input.Reset()
				return m, nil
			}
			return m, tea.Quit
		}

		// When on the Config tab the input is focused. If the input is empty,
		// arrow keys and h/l still navigate tabs; otherwise they edit the command.
		if m.activeTab != TabConfig || m.config.input.Value() == "" {
			switch keyStr {
			case "right", "l", "tab":
				m.activeTab = (m.activeTab + 1) % tabCount
				return m, nil
			case "left", "h":
				m.activeTab = (m.activeTab - 1 + tabCount) % tabCount
				return m, nil
			}
		}

		if m.activeTab == TabConfig {
			if keyStr == "enter" {
				return m.handleConfigCommand()
			}
			var cmd tea.Cmd
			m.config.input, cmd = m.config.input.Update(msg)
			return m, cmd
		}

		return m, nil

	case ConnectedMsg:
		m.connected = true
		m.lastErr = ""
		return m, tea.Batch(m.client.Read(), loadConfigCmd(m.configClient))

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

	case ConfigLoadedMsg:
		m.config.Update(service.ConfigView(msg))
		return m, nil

	case ConfigSavedMsg:
		m.config.Update(service.ConfigView(msg))
		return m, nil

	case ConfigErrorMsg:
		m.config.SetError(msg.Error)
		return m, nil

	case PromptEditorResultMsg:
		if msg.Err != nil {
			m.config.SetError(msg.Err.Error())
			return m, nil
		}
		current := m.config.view.Judge.EffectivePrompt
		trimmed := strings.TrimSpace(msg.Content)
		if trimmed == "" || trimmed == strings.TrimSpace(current) {
			return m, loadConfigCmd(m.configClient)
		}
		patch := service.ConfigPatch{
			Judge: &service.JudgeConfigPatch{Prompt: &trimmed},
		}
		return m, patchConfigCmd(m.configClient, patch)
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
	case TabConfig:
		body = m.config.View(m.width, m.height-lipgloss.Height(status)-lipgloss.Height(tabs)-2)
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
	names := []string{"Live", "Advice", "Events", "Log", "Config"}
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

// handleConfigCommand parses a command typed in the Config tab input and
// returns the corresponding action.
func (m Model) handleConfigCommand() (Model, tea.Cmd) {
	cmdStr := strings.TrimSpace(m.config.input.Value())
	m.config.input.Reset()
	if cmdStr == "" {
		return m, nil
	}
	if !strings.HasPrefix(cmdStr, "/") {
		m.config.SetStatus("commands start with /")
		return m, nil
	}

	parts := strings.Fields(cmdStr[1:])
	if len(parts) == 0 {
		return m, nil
	}
	switch parts[0] {
	case "help":
		m.config.SetStatus("/lang, /language, /prompt")
		return m, nil
	case "lang", "language":
		if m.config.loading {
			return m, nil
		}
		next := nextLanguage(m.config.view.Judge.Language)
		patch := service.ConfigPatch{
			Judge: &service.JudgeConfigPatch{Language: &next},
		}
		return m, patchConfigCmd(m.configClient, patch)
	case "prompt":
		if m.config.loading {
			return m, nil
		}
		cmd, err := editPromptCmd(m.config.view.Judge.EffectivePrompt)
		if err != nil {
			m.config.SetError(err.Error())
			return m, nil
		}
		return m, cmd
	default:
		m.config.SetStatus(fmt.Sprintf("unknown command: %s", parts[0]))
		return m, nil
	}
}

// loadConfigCmd fetches the runtime config and sends it to the model.
func loadConfigCmd(client *ConfigClient) tea.Cmd {
	return func() tea.Msg {
		v, err := client.Get()
		if err != nil {
			return ConfigErrorMsg{Error: err.Error()}
		}
		return ConfigLoadedMsg(v)
	}
}

// patchConfigCmd sends a PATCH and forwards the result.
func patchConfigCmd(client *ConfigClient, patch service.ConfigPatch) tea.Cmd {
	return func() tea.Msg {
		v, err := client.Patch(patch)
		if err != nil {
			return ConfigErrorMsg{Error: err.Error()}
		}
		return ConfigSavedMsg(v)
	}
}

// editorRunner launches an external editor for the user to edit the prompt.
// It is a variable so tests can inject a fake runner.
var editorRunner = func(path string) tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		return func() tea.Msg {
			return fmt.Errorf("$EDITOR not set")
		}
	}
	return tea.ExecProcess(exec.Command(editor, path), func(err error) tea.Msg {
		if err != nil {
			return fmt.Errorf("editor exited: %w", err)
		}
		return nil
	})
}

// editPromptCmd writes the current effective prompt to a temp file and launches
// the user's $EDITOR. When the editor exits, the file content is read and sent
// back as a PromptEditorResultMsg.
func editPromptCmd(currentPrompt string) (tea.Cmd, error) {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		return nil, fmt.Errorf("$EDITOR not set; set it or use curl")
	}

	f, err := os.CreateTemp("", "lol-prompt-*.txt")
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}
	path := f.Name()
	if _, err := f.WriteString(currentPrompt); err != nil {
		f.Close()
		os.Remove(path)
		return nil, fmt.Errorf("write temp file: %w", err)
	}
	if err := f.Close(); err != nil {
		os.Remove(path)
		return nil, fmt.Errorf("close temp file: %w", err)
	}

	return func() tea.Msg {
		defer os.Remove(path)
		msg := editorRunner(path)()
		if err, ok := msg.(error); ok && err != nil {
			return PromptEditorResultMsg{Path: path, Err: err}
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return PromptEditorResultMsg{Path: path, Err: err}
		}
		return PromptEditorResultMsg{Path: path, Content: string(content)}
	}, nil
}

// nextLanguage cycles through supported languages.
func nextLanguage(current string) string {
	langs := []string{"en", "pt-BR", "es"}
	for i, l := range langs {
		if l == current {
			return langs[(i+1)%len(langs)]
		}
	}
	return "en"
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

