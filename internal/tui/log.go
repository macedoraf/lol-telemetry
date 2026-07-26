// Package tui implements the Bubble Tea interface for the lol-telemetry daemon client.
package tui

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"lol-telemetry/pkg/service"
)

// LogState holds recent raw WebSocket messages.
type LogState struct {
	lines []string
}

// Update appends a raw message to the log.
func (s *LogState) Update(msg service.WSMessage) {
	compact, err := json.Marshal(msg)
	if err != nil {
		compact = []byte(fmt.Sprintf(`{"type":"%s"}`, msg.Type))
	}
	s.lines = append(s.lines, string(compact))
	if len(s.lines) > 100 {
		s.lines = s.lines[len(s.lines)-100:]
	}
}

// View renders the log panel.
func (s LogState) View(width, height int) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Raw Message Log") + "\n\n")
	if len(s.lines) == 0 {
		b.WriteString(placeholderStyle.Render("Waiting for messages..."))
	} else {
		for _, line := range s.lines {
			b.WriteString(truncate(line, width-6) + "\n")
		}
	}
	return contentStyle.Width(width - 4).Height(height).Render(b.String())
}

var (
	titleStyle       = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#5A56E0"))
	placeholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	contentStyle     = lipgloss.NewStyle().Padding(0, 1)
)

func truncate(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

// avoid unused strings import if any helper changes.
var _ = strings.Join
