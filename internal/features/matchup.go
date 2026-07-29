package features

import (
	"lol-telemetry/internal/types"
)

type matchupTransformer struct{}

func (matchupTransformer) Name() string { return "matchup" }

func (matchupTransformer) Transform(w Window, fv *types.FeatureVector) {
	last, ok := w.Last()
	if !ok {
		return
	}
	active := findActive(last)
	if active == nil {
		return
	}
	opp, ok := findOpponent(last, active.Position, active.Team)
	if !ok {
		fv.Matchup = nil
		return
	}
	fv.Matchup = &types.MatchupFeatures{
		LevelDiff:        active.Level - opp.Level,
		CSDiff:           active.CS - opp.CS,
		ItemDiff:         active.ItemsCompleted - opp.ItemsCompleted,
		KillDiff:         active.Kills - opp.Kills,
		OpponentXPPerMin: perMin(float64(xpForLevel(opp.Level)), last.GameTime),
	}
}
