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

// GameDataProvider abstracts the source of live game snapshots.
type GameDataProvider interface {
	GetGameData() (riotclient.AllGameData, error)
}

// Judge evaluates a JudgeRequest and returns a JudgeResponse.
type Judge interface {
	Evaluate(ctx context.Context, req types.JudgeRequest) (types.JudgeResponse, error)
}

// Orchestrator runs the periodic evaluation loop.
type Orchestrator struct {
	provider  GameDataProvider
	registry  *hooks.Registry
	builder   *payload.Builder
	judge     Judge
	prevFired map[string]int64
	lastErr   error
}

// NewOrchestrator creates a new orchestrator.
func NewOrchestrator(provider GameDataProvider, registry *hooks.Registry, builder *payload.Builder, j Judge) *Orchestrator {
	return &Orchestrator{
		provider:  provider,
		registry:  registry,
		builder:   builder,
		judge:     j,
		prevFired: make(map[string]int64),
	}
}

// Tick fetches game data, evaluates hooks, and invokes the Judge for each trigger.
func (o *Orchestrator) Tick(ctx context.Context) ([]types.JudgeResponse, error) {
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
		GameTime:  gameTime,
		PrevFired: o.prevFired,
	}
	triggers, err := o.registry.Evaluate(ctxHook)
	if err != nil {
		o.lastErr = err
		return nil, fmt.Errorf("evaluate hooks: %w", err)
	}

	var responses []types.JudgeResponse
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
		responses = append(responses, resp)
		mark := hooks.CurrentMark(gameTime)
		o.prevFired[trigger.HookName] = mark
	}

	o.lastErr = nil
	return responses, nil
}

// reset clears deduplication state between matches.
func (o *Orchestrator) reset() {
	for k := range o.prevFired {
		delete(o.prevFired, k)
	}
}
