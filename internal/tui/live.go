// Package tui implements the Bubble Tea interface for the lol-telemetry daemon client.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"lol-telemetry/pkg/service"
)

// LiveState holds the latest game state snapshot.
type LiveState struct {
	gs service.GameState
}

// Update stores the latest game state.
func (s *LiveState) Update(gs service.GameState) {
	s.gs = gs
}

// View renders the live panel.
func (s LiveState) View(width, height int) string {
	if s.gs.GameMode == "" && s.gs.GameTime == 0 {
		return placeholderStyle.Render("Waiting for game state...")
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render("Live Game State") + "\n\n")
	b.WriteString(fmt.Sprintf("Mode: %s\n", s.gs.GameMode))
	b.WriteString(fmt.Sprintf("Map:  %s\n", s.gs.MapName))
	b.WriteString(fmt.Sprintf("Time: %.1f (minute %d)\n\n", s.gs.GameTime, s.gs.GameMinute()))

	if len(s.gs.Players) > 0 {
		b.WriteString(playerHeaderStyle.Render(fmt.Sprintf("%-16s %-12s %-6s %-8s %3s %3s %3s %5s %4s",
			"Summoner", "Champion", "Team", "Pos", "K", "D", "A", "CS", "Gold")) + "\n")
		for _, p := range s.gs.Players {
			active := " "
			if p.IsActive {
				active = "*"
			}
			b.WriteString(fmt.Sprintf("%s%-15s %-12s %-6s %-8s %3d %3d %3d %5d %4d\n",
				active, truncate(p.SummonerName, 15), truncate(p.ChampionName, 12),
				truncate(p.Team, 6), truncate(p.Position, 8),
				p.Kills, p.Deaths, p.Assists, p.CS, p.CurrentGold))
		}
	}

	return contentStyle.Width(width - 4).Height(height).Render(b.String())
}

var playerHeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#5A56E0"))
