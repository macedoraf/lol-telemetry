package features

import (
	"lol-telemetry/internal/types"
)

// Transformer computes one feature slice from the current window.
type Transformer interface {
	Name() string
	Transform(w Window, fv *types.FeatureVector)
}

// Pipeline runs all registered transformers in order.
type Pipeline struct {
	transformers []Transformer
}

// NewPipeline returns the default feature pipeline (F1–F6).
func NewPipeline() *Pipeline {
	return &Pipeline{
		transformers: []Transformer{
			&goldTransformer{},
			&xpTransformer{},
			&spikesTransformer{},
			&matchupTransformer{},
			&objectivesTransformer{},
		},
	}
}

// Compute builds a FeatureVector from the current window.
func (p *Pipeline) Compute(w Window) types.FeatureVector {
	fv := types.FeatureVector{}
	if samples := w.Samples(); len(samples) > 0 {
		last := samples[len(samples)-1]
		fv.GameMinute = int(last.GameTime) / 60
		fv.Samples = len(samples)
		fv.WindowSec = last.GameTime - samples[0].GameTime
	}
	for _, t := range p.transformers {
		t.Transform(w, &fv)
	}
	return fv
}

// teamFeaturesFor returns the ally or enemy team features based on team name.
func teamFeaturesFor(fv *types.FeatureVector, allyTeam, team string) *types.TeamFeatures {
	if team == allyTeam {
		return &fv.Team
	}
	return &fv.Enemy
}

// objectiveFor returns the ObjectiveCount of the ally or enemy team.
func objectiveFor(fv *types.FeatureVector, allyTeam, team string) *types.ObjectiveCount {
	if team == allyTeam {
		return &fv.Team.Objectives
	}
	return &fv.Enemy.Objectives
}
