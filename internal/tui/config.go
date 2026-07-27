package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"lol-telemetry/pkg/service"
)

// ConfigState holds a snapshot of the runtime config for the Config tab.
type ConfigState struct {
	view    service.ConfigView
	err     string
	loading bool
}

func newConfigState() ConfigState {
	return ConfigState{loading: true}
}

func (s *ConfigState) Update(v service.ConfigView) {
	s.view = v
	s.err = ""
	s.loading = false
}

func (s *ConfigState) SetError(err string) {
	s.err = err
	s.loading = false
}

func (s *ConfigState) View(width, height int) string {
	if s.loading {
		return "Loading config..."
	}
	if s.err != "" {
		return errorStyle.Render("Config error: " + s.err)
	}
	var b strings.Builder
	b.WriteString(configTitleStyle.Render("Judge Configuration") + "\n")
	b.WriteString(fmt.Sprintf("  Language: %s\n", s.view.Judge.Language))
	b.WriteString(fmt.Sprintf("  Prompt Override: %s\n\n", promptOverrideLabel(s.view.Judge.PromptOverride)))

	b.WriteString(configTitleStyle.Render("Effective Prompt") + "\n")
	b.WriteString(wrap(s.view.Judge.EffectivePrompt, width-4, "    "))
	b.WriteString("\n\n")

	b.WriteString(configTitleStyle.Render("Hooks") + "\n")
	for _, h := range s.view.Hooks {
		status := "[  ]"
		statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555"))
		if h.Enabled {
			status = "[✓]"
			statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00"))
		}
		b.WriteString(fmt.Sprintf("  %s %-22s", statusStyle.Render(status), h.Name))
		if len(h.Params) > 0 {
			var parts []string
			for k, v := range h.Params {
				if n, ok := v.(float64); ok && n == 0 {
					continue
				}
				parts = append(parts, fmt.Sprintf("%s=%v", k, v))
			}
			if len(parts) > 0 {
				b.WriteString("  " + strings.Join(parts, ", "))
			}
		}
		b.WriteString("\n")
	}

	b.WriteString("\n" + helpStyle.Render("space toggle • l cycle language • p edit prompt • q quit"))

	return b.String()
}

var configTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#5A56E0"))

func promptOverrideLabel(prompt string) string {
	if prompt == "" {
		return "(default)"
	}
	if len(prompt) > 40 {
		return prompt[:37] + "..."
	}
	return prompt
}

func wrap(s string, width int, prefix string) string {
	if width <= 0 {
		return s
	}
	var out strings.Builder
	var line strings.Builder
	first := true
	for _, word := range strings.Fields(s) {
		if line.Len()+len(word)+1 > width && line.Len() > 0 {
			if !first {
				out.WriteString("\n")
			}
			out.WriteString(prefix)
			out.WriteString(line.String())
			line.Reset()
			first = false
		}
		if line.Len() > 0 {
			line.WriteByte(' ')
		}
		line.WriteString(word)
	}
	if line.Len() > 0 {
		if !first {
			out.WriteString("\n")
		}
		out.WriteString(prefix)
		out.WriteString(line.String())
	}
	return out.String()
}
