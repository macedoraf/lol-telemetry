package features

import (
	"lol-telemetry/internal/types"
)

// GoldSpikeThreshold is the raw gold delta in one minute considered a spike.
const GoldSpikeThreshold = 1200

// StartingGold returns the initial gold for the given game mode.
// Unknown modes return 0, meaning no correction is applied.
func StartingGold(mode string) float64 {
	switch mode {
	case "CLASSIC":
		return 500
	case "ARAM":
		return 1400
	default:
		return 0
	}
}

type goldTransformer struct{}

func (goldTransformer) Name() string { return "gold" }

func (goldTransformer) Transform(w Window, fv *types.FeatureVector) {
	last, ok := w.Last()
	if !ok {
		return
	}
	active := findActive(last)
	if active == nil {
		return
	}
	corrected := active.Gold - StartingGold(last.GameMode)
	if corrected < 0 {
		corrected = 0
	}
	fv.Player.GoldPerMin = perMin(corrected, last.GameTime)
	fv.Player.GoldDelta1m = goldDelta(w, 60, active)
	fv.Player.GoldDelta5m = goldDelta(w, 300, active)
	fv.Player.GoldSpike1m = fv.Player.GoldDelta1m > GoldSpikeThreshold
}

func goldDelta(w Window, seconds float64, active *PlayerSample) float64 {
	samples := w.Since(seconds)
	if len(samples) < 2 {
		return 0
	}
	first := findActive(samples[0])
	last := findActive(samples[len(samples)-1])
	if first == nil || last == nil {
		return 0
	}
	return last.Gold - first.Gold
}
