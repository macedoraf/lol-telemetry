// Package features builds time-series derived features from raw Live Client Data
// without modifying the original riotclient models.
package features

import (
	"fmt"
	"sync"

	"lol-telemetry/pkg/riotclient"
)

// Sample is a minimal per-tick snapshot extracted from AllGameData.
type Sample struct {
	GameTime float64
	GameMode string
	Players  []PlayerSample
}

// PlayerSample is a compact scalar view of one player at a tick.
type PlayerSample struct {
	SummonerName, ChampionName, Team, Position string
	Level, Kills, Deaths, Assists, CS, ItemsCompleted, ItemsGold int
	IsActive     bool
	IsDead       bool
	RespawnTimer float64
	Gold         float64
}

// Window is a read-only view of the accumulated time series.
type Window interface {
	Samples() []Sample
	Last() (Sample, bool)
	Since(seconds float64) []Sample
	Events() []riotclient.Event
}

// Tracker stores a ring buffer of samples and the latest event list.
// All reads return deep copies so concurrent callers cannot mutate the
// underlying slices or observe partial writes.
type Tracker struct {
	mu     sync.RWMutex
	cap    int
	samples []Sample
	head   int
	count  int
	events []riotclient.Event
	lastGameTime float64
}

// NewTracker returns a tracker with the default 1-hour @ 1-second capacity.
func NewTracker() *Tracker {
	const cap = 3600
	return &Tracker{cap: cap, samples: make([]Sample, cap)}
}

// Add extracts a sample from data and stores it in the ring buffer.
func (t *Tracker) Add(data riotclient.AllGameData) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.events = copyEvents(data.Events.Events)
	t.lastGameTime = data.GameData.GameTime

	s := extractSample(data)
	t.head = (t.head + 1) % t.cap
	t.samples[t.head] = s
	if t.count < t.cap {
		t.count++
	}
}

// Window returns a read-only snapshot of the current series.
func (t *Tracker) Window() Window {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return &window{
		samples: copySamples(t.samples, t.head, t.count, t.cap),
		events:  copyEvents(t.events),
	}
}

// Reset clears the accumulated samples and events.
func (t *Tracker) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.samples = make([]Sample, t.cap)
	t.head = 0
	t.count = 0
	t.events = nil
	t.lastGameTime = 0
}

// LastGameTime returns the most recent gameTime observed by the tracker.
func (t *Tracker) LastGameTime() float64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.lastGameTime
}

// window is the read-only snapshot returned by Tracker.Window.
type window struct {
	samples []Sample
	events  []riotclient.Event
}

func (w *window) Samples() []Sample {
	return copySamples(w.samples, len(w.samples)-1, len(w.samples), len(w.samples))
}

func (w *window) Last() (Sample, bool) {
	if len(w.samples) == 0 {
		return Sample{}, false
	}
	return copySample(w.samples[len(w.samples)-1]), true
}

func (w *window) Since(seconds float64) []Sample {
	last, ok := w.Last()
	if !ok {
		return nil
	}
	minTime := last.GameTime - seconds
	out := make([]Sample, 0, len(w.samples))
	for _, s := range w.samples {
		if s.GameTime >= minTime {
			out = append(out, copySample(s))
		}
	}
	return out
}

func (w *window) Events() []riotclient.Event {
	return copyEvents(w.events)
}

func copySamples(samples []Sample, head, count, cap int) []Sample {
	if count == 0 {
		return nil
	}
	start := (head - count + 1 + cap) % cap
	out := make([]Sample, count)
	for i := 0; i < count; i++ {
		idx := (start + i) % cap
		out[i] = copySample(samples[idx])
	}
	return out
}

func copySample(s Sample) Sample {
	out := s
	out.Players = make([]PlayerSample, len(s.Players))
	copy(out.Players, s.Players)
	return out
}

func copyEvents(events []riotclient.Event) []riotclient.Event {
	out := make([]riotclient.Event, len(events))
	for i, e := range events {
		out[i] = e
		if e.Assisters != nil {
			out[i].Assisters = make([]string, len(e.Assisters))
			copy(out[i].Assisters, e.Assisters)
		}
	}
	return out
}

func extractSample(data riotclient.AllGameData) Sample {
	players := make([]PlayerSample, 0, len(data.AllPlayers))
	activeName := data.ActivePlayer.SummonerName
	for _, p := range data.AllPlayers {
		players = append(players, PlayerSample{
			SummonerName:   p.SummonerName,
			ChampionName:   p.ChampionName,
			Team:           p.Team,
			Position:       p.Position,
			Level:          p.Level,
			Kills:          p.Scores.Kills,
			Deaths:         p.Scores.Deaths,
			Assists:        p.Scores.Assists,
			CS:             p.Scores.CreepScore,
			ItemsCompleted: riotclient.ItemCount(p.Items),
			ItemsGold:      itemsGold(p.Items),
			IsActive:       p.SummonerName == activeName,
			IsDead:         p.IsDead,
			RespawnTimer:   p.RespawnTimer,
			Gold:           goldFor(p, activeName, data.ActivePlayer.CurrentGold),
		})
	}
	return Sample{
		GameTime: data.GameData.GameTime,
		GameMode: data.GameData.GameMode,
		Players:  players,
	}
}

func itemsGold(items []riotclient.Item) int {
	sum := 0
	for _, it := range items {
		if it.ItemID != 0 && !it.Consumable {
			sum += it.Price
		}
	}
	return sum
}

func goldFor(p riotclient.AllPlayer, activeName string, activeGold float64) float64 {
	if p.SummonerName == activeName {
		return activeGold
	}
	return 0
}

// findActive returns the active player sample, or nil if none matches.
func findActive(s Sample) *PlayerSample {
	for i := range s.Players {
		if s.Players[i].IsActive {
			return &s.Players[i]
		}
	}
	return nil
}

// findPlayer returns a sample by summoner name, or nil if absent.
func findPlayer(s Sample, name string) *PlayerSample {
	for i := range s.Players {
		if s.Players[i].SummonerName == name {
			return &s.Players[i]
		}
	}
	return nil
}

// findOpponent returns the sample on the opposite team with the same position.
func findOpponent(s Sample, position, activeTeam string) (*PlayerSample, bool) {
	if position == "" {
		return nil, false
	}
	for i := range s.Players {
		p := &s.Players[i]
		if p.Team != activeTeam && p.Position == position {
			return p, true
		}
	}
	return nil, false
}

// oppositeTeam maps ORDER/CHAOS to the other side.
func oppositeTeam(team string) string {
	if team == "ORDER" {
		return "CHAOS"
	}
	return "ORDER"
}

// teamOf returns the team of the player with the given name, or "".
func teamOf(players []PlayerSample, name string) string {
	for _, p := range players {
		if p.SummonerName == name {
			return p.Team
		}
	}
	return ""
}

// formatGameTime returns mm:ss for a gameTime in seconds.
func formatGameTime(t float64) string {
	sec := int(t)
	m := sec / 60
	s := sec % 60
	return fmt.Sprintf("%02d:%02d", m, s)
}

// safeDiv prevents division by very small or zero denominators.
func safeDiv(n, d float64) float64 {
	if d < 1.0 {
		return 0
	}
	return n / d
}

// perMin converts a value accumulated over gameSeconds into per-minute.
func perMin(value, gameSeconds float64) float64 {
	return safeDiv(value, gameSeconds/60)
}
