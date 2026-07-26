// Package tui implements the Bubble Tea interface for the lol-telemetry daemon client.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"lol-telemetry/pkg/service"
)

// AdviceState holds the latest judge advice.
type AdviceState struct {
	advice     string
	reasoning  string
	hookName   string
	gameMinute int
}

// Update stores the latest advice.
func (s *AdviceState) Update(advice service.JudgeAdvice) {
	s.advice = advice.Advice
	s.reasoning = advice.Reasoning
	s.hookName = advice.HookName
	s.gameMinute = advice.GameMinute
}

// SetGameMinute updates the current minute from live state.
func (s *AdviceState) SetGameMinute(min int) {
	if s.gameMinute == 0 {
		s.gameMinute = min
	}
}

// View renders the advice panel.
func (s AdviceState) View(width, height int) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Judge Advice") + "\n\n")
	if s.advice == "" {
		b.WriteString(placeholderStyle.Render("Waiting for advice..."))
	} else {
		b.WriteString(fmt.Sprintf("Hook: %s\n", s.hookName))
		b.WriteString(fmt.Sprintf("Minute: %d\n\n", s.gameMinute))
		b.WriteString(adviceBoxStyle.Render(s.advice) + "\n")
		if s.reasoning != "" {
			b.WriteString("\nReason: " + s.reasoning)
		}
	}
	return contentStyle.Width(width - 4).Height(height).Render(b.String())
}

var adviceBoxStyle = lipgloss.NewStyle().
	Border(lipgloss.NormalBorder()).
	Padding(1, 2).
	Foreground(lipgloss.Color("#FFFFFF"))
