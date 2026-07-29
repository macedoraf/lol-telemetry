package features

import (
	"fmt"

	"lol-telemetry/internal/types"
)

type spikesTransformer struct{}

func (spikesTransformer) Name() string { return "spikes" }

func (spikesTransformer) Transform(w Window, fv *types.FeatureVector) {
	last, ok := w.Last()
	if !ok {
		return
	}
	active := findActive(last)
	if active == nil {
		return
	}
	fv.Player.ItemsCompleted = active.ItemsCompleted

	samples := w.Since(60)
	if len(samples) < 2 {
		return
	}
	first := samples[0]
	allyTeam := active.Team

	fv.Team.ItemCompletions1m, fv.Team.LevelUps1m, fv.Team.Spikes =
		teamChanges(first, last, allyTeam)
	fv.Enemy.ItemCompletions1m, fv.Enemy.LevelUps1m, fv.Enemy.Spikes =
		teamChanges(first, last, oppositeTeam(allyTeam))
}

func teamChanges(first, last Sample, team string) (int, int, []string) {
	var itemDelta, levelDelta int
	var spikes []string
	for i := range last.Players {
		lp := &last.Players[i]
		if lp.Team != team {
			continue
		}
		fp := findPlayer(first, lp.SummonerName)
		if fp == nil {
			continue
		}
		if d := lp.ItemsCompleted - fp.ItemsCompleted; d > 0 {
			itemDelta += d
			spikes = append(spikes, fmt.Sprintf("%s completed item @%s", lp.ChampionName, formatGameTime(last.GameTime)))
		}
		if d := lp.Level - fp.Level; d > 0 {
			levelDelta += d
			spikes = append(spikes, fmt.Sprintf("%s level up to %d @%s", lp.ChampionName, lp.Level, formatGameTime(last.GameTime)))
		}
	}
	return itemDelta, levelDelta, spikes
}
