package features

import (
	"lol-telemetry/internal/types"
)

// XP_TABLE maps Summoner's Rift level to cumulative XP.
// XP_TABLE[1] = 0 by construction so the cold start is absorbed.
// ponytail: approximation — real XP is not exposed by the Live Client API.
// Typical error is <5% after the laning phase.
var XP_TABLE = []int{
	0,   // padding for 0
	0,   // 1
	280, // 2
	660, // 3
	1140,
	1720,
	2400,
	3180,
	4060,
	5040,
	6120,
	7300,
	8580,
	9960,
	11440,
	13020,
	14700,
	16480,
	18360,
}

// xpForLevel returns the estimated cumulative XP for a level.
func xpForLevel(level int) int {
	if level <= 1 {
		return 0
	}
	if level >= len(XP_TABLE) {
		return XP_TABLE[len(XP_TABLE)-1]
	}
	return XP_TABLE[level]
}

type xpTransformer struct{}

func (xpTransformer) Name() string { return "xp" }

func (xpTransformer) Transform(w Window, fv *types.FeatureVector) {
	last, ok := w.Last()
	if !ok {
		return
	}
	active := findActive(last)
	if active == nil {
		return
	}
	fv.Player.XPPerMin = perMin(float64(xpForLevel(active.Level)), last.GameTime)
	fv.Player.XPDelta1m = xpDelta(w, 60, active)
	fv.Player.XPDelta5m = xpDelta(w, 300, active)
	fv.Team.AvgXPPerMin = avgXPPerMin(last.Players, active.Team, last.GameTime)
	fv.Enemy.AvgXPPerMin = avgXPPerMin(last.Players, oppositeTeam(active.Team), last.GameTime)
}

func xpDelta(w Window, seconds float64, active *PlayerSample) float64 {
	samples := w.Since(seconds)
	if len(samples) < 2 {
		return 0
	}
	first := findActive(samples[0])
	last := findActive(samples[len(samples)-1])
	if first == nil || last == nil {
		return 0
	}
	return float64(xpForLevel(last.Level) - xpForLevel(first.Level))
}

func avgXPPerMin(players []PlayerSample, team string, gameTime float64) float64 {
	var sum float64
	var n int
	for _, p := range players {
		if p.Team == team {
			sum += perMin(float64(xpForLevel(p.Level)), gameTime)
			n++
		}
	}
	if n == 0 {
		return 0
	}
	return sum / float64(n)
}
