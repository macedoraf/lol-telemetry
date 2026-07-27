// Package orchestrator coordinates polling, hooks, payload building, and Judge evaluation.
package orchestrator

import (
	"context"
	"fmt"

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
	Advice     string
	Reasoning  string
}

// Orchestrator runs the periodic evaluation loop.
type Orchestrator struct {
	provider  *riotclient.Client
	registry  *hooks.Registry
	builder   *payload.Builder
	judge     Judge
	prevData  riotclient.AllGameData
	prevFired map[string]int64
	lastErr   error
}

// NewOrchestrator creates a new orchestrator.
func NewOrchestrator(provider *riotclient.Client, registry *hooks.Registry, builder *payload.Builder, j Judge) *Orchestrator {
	return &Orchestrator{
		provider:  provider,
		registry:  registry,
		builder:   builder,
		judge:     j,
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
		resp, err := o.judge.Evaluate(ctx, req)
		if err != nil {
			o.lastErr = err
			continue
		}
		results = append(results, Result{
			HookName:   trigger.HookName,
			GameMinute: req.GameMinute,
			Advice:     resp.Advice,
			Reasoning:  resp.Reasoning,
		})
		mark := hooks.CurrentMark(gameTime)
		o.prevFired[trigger.HookName] = mark
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
}
