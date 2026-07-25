package renderer

import (
	"strings"
	"testing"

	"lol-telemetry/internal/types"
)

func TestStatsView(t *testing.T) {
	m := Model{state: types.DashboardState{
		Stats: types.PlayerStats{
			SummonerName: "Player",
			ChampionName: "Ashe",
			Level:        12,
			CurrentGold:  3500,
			GameTime:     600,
			CSPerMin:     14.0,
			GPM:          350.0,
		},
	}}

	view := m.View()
	for _, want := range []string{"LoL Telemetry", "Player", "Ashe", "14.00", "350.00"} {
		if !strings.Contains(view, want) {
			t.Errorf("View() missing %q", want)
		}
	}
}

func TestErrorView(t *testing.T) {
	m := Model{state: types.DashboardState{Error: "boom"}}
	view := m.View()
	if !strings.Contains(view, "boom") {
		t.Errorf("View() missing error message")
	}
}

func TestWaitingView(t *testing.T) {
	m := Model{state: types.DashboardState{Waiting: true}}
	view := m.View()
	if !strings.Contains(view, "Waiting for League of Legends match...") {
		t.Errorf("View() missing waiting message")
	}
}
