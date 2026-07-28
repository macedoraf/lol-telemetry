// Package orchestrator coordinates polling, hooks, payload building, and Judge evaluation.
package orchestrator

import (
	"context"
	"fmt"

	"lol-telemetry/internal/features"
	"lol-telemetry/internal/hooks"
	"lol-telemetry/internal/judge/payload"
	"lol-telemetry/internal/types"
	"lol-telemetry/pkg/riotclient"
)

// Judge evaluates a JudgeRequest and returns a JudgeResponse.
// ponytail: kept as a local interface so tests can inject a fake LLM.
type Judge interface {
	Evaluate(ctx context.Context, req types.JudgeRequest) (types.JudgeResponse, error)
}

// Result pairs the hook that fired with the Judge response.
type Result struct {
	HookName   string
	GameMinute int
	GameTime   float64
	Question   string
	Advice     string
	Reasoning  string
}

// Orchestrator runs the periodic evaluation loop.
type Orchestrator struct {
	provider  *riotclient.Client
	registry  *hooks.Registry
	builder   *payload.Builder
	judge     Judge
	pipeline  *features.Pipeline
	tracker   *features.Tracker
	prevData  riotclient.AllGameData
	prevFired map[string]int64
	lastErr   error
}

// NewOrchestrator creates a new orchestrator.
// pipeline and tracker may be nil to keep the legacy JudgeRequest unchanged.
func NewOrchestrator(provider *riotclient.Client, registry *hooks.Registry, builder *payload.Builder, j Judge, pipeline *features.Pipeline, tracker *features.Tracker) *Orchestrator {
	return &Orchestrator{
		provider:  provider,
		registry:  registry,
		builder:   builder,
		judge:     j,
		pipeline:  pipeline,
		tracker:   tracker,
		prevFired: make(map[string]int64),
	}
}

// Tick fetches game data, evaluates hooks, and invokes the Judge for each trigger.
func (o *Orchestrator) Tick(ctx context.Context) ([]Result, error) {
	data, err := o.provider.GetGameData()
	if err != nil {
		o.reset()
		o.lastErr = err
		return nil, fmt.Errorf("fetch game data: %w", err)
	}

	gameTime := data.GameData.GameTime
	if gameTime <= 0 {
		o.reset()
		return nil, nil
	}

	// First observation after reset: establish a baseline for each hook so
	// that past marks are not processed retroactively.
	if len(o.prevFired) == 0 {
		for _, h := range o.registry.Hooks() {
			o.prevFired[h.Name()] = h.CurrentMark(gameTime)
		}
	}

	ctxHook := types.HookContext{
		Data:      data,
		PrevData:  o.prevData,
		GameTime:  gameTime,
		PrevFired: o.prevFired,
	}
	triggers, err := o.registry.Evaluate(ctxHook)
	// Keep the latest data for transition detection in the next tick.
	o.prevData = data
	if err != nil {
		o.lastErr = err
		return nil, fmt.Errorf("evaluate hooks: %w", err)
	}

	if o.judge == nil {
		return nil, nil
	}

	var results []Result
	for _, trigger := range triggers {
		req, err := o.builder.Build(data, trigger.Question)
		if err != nil {
			o.lastErr = err
			continue
		}
		if o.pipeline != nil && o.tracker != nil {
			fv := o.pipeline.Compute(o.tracker.Window())
			req.Features = &fv
			if active, ok := riotclient.FindActivePlayer(data); ok {
				req.Objectives = objectivesFromFeatures(fv, active.Team)
			}
		}
		resp, err := o.judge.Evaluate(ctx, req)
		if err != nil {
			o.lastErr = err
			continue
		}
		results = append(results, Result{
			HookName:   trigger.HookName,
			GameMinute: req.GameMinute,
			GameTime:   gameTime,
			Question:   trigger.Question,
			Advice:     resp.Advice,
			Reasoning:  resp.Reasoning,
		})
		if hook, ok := o.registry.GetHook(trigger.HookName); ok {
			o.prevFired[trigger.HookName] = hook.CurrentMark(gameTime)
		}
	}

	o.lastErr = nil
	return results, nil
}

// reset clears deduplication state between matches.
func (o *Orchestrator) reset() {
	for k := range o.prevFired {
		delete(o.prevFired, k)
	}
	o.prevData = riotclient.AllGameData{}
	if o.tracker != nil {
		o.tracker.Reset()
	}
}

// objectivesFromFeatures maps the new feature objectives back to the legacy
// TeamObjectives layout. It only runs when the feature pipeline is enabled.
func objectivesFromFeatures(fv types.FeatureVector, allyTeam string) types.TeamObjectives {
	ally := toObjectiveState(fv.Team.Objectives)
	enemy := toObjectiveState(fv.Enemy.Objectives)
	if allyTeam == "ORDER" {
		return types.TeamObjectives{Order: ally, Chaos: enemy}
	}
	return types.TeamObjectives{Order: enemy, Chaos: ally}
}

func toObjectiveState(o types.ObjectiveCount) types.ObjectiveState {
	return types.ObjectiveState{
		Towers:  o.Towers,
		Dragons: o.Dragons,
		Barons:  o.Barons,
		Heralds: o.Heralds,
	}
}

// ResetHook clears the deduplication mark for a hook, preventing retroactive fire
// after configuration changes.
func (o *Orchestrator) ResetHook(name string) {
	if hook, ok := o.registry.GetHook(name); ok {
		if data, err := o.provider.GetGameData(); err == nil {
			o.prevFired[name] = hook.CurrentMark(data.GameData.GameTime)
		}
	}
}
