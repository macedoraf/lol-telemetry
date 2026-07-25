// Package hooks contains trigger conditions that decide when to invoke the Judge.
package hooks

import (
	"fmt"

	"lol-telemetry/internal/types"
)

// PeriodicHook fires once for every absolute 5-minute mark of the game clock.
const PeriodicHookName = "periodic-5min"

// Periodic5MinHook is the hook that triggers at 5, 10, 15... minutes.
type Periodic5MinHook struct {
}

// Name returns the hook identifier.
func (Periodic5MinHook) Name() string {
	return PeriodicHookName
}

// Instruction returns the question asked to the Judge.
func (Periodic5MinHook) Instruction() string {
	return "Evaluate the current macro state and give one actionable sentence of advice."
}

// ShouldFire returns true when the game clock crosses a new 5-minute mark.
// On the first positive observation (no prior baseline) it returns false so
// the orchestrator can establish the baseline without firing retroactively.
func (Periodic5MinHook) ShouldFire(ctx types.HookContext) (bool, error) {
	if ctx.GameTime <= 0 {
		return false, nil
	}
	mark := CurrentMark(ctx.GameTime)
	if mark == 0 {
		return false, nil
	}
	last, ok := ctx.PrevFired[PeriodicHookName]
	if !ok {
		// First observation: establish baseline, do not fire.
		return false, nil
	}
	if last >= mark {
		return false, nil
	}
	return true, nil
}

// CurrentMark returns the greatest absolute 5-minute mark (in seconds) that
// gameTime has reached. Marks start at 5 minutes (300 seconds).
func CurrentMark(gameTime float64) int64 {
	if gameTime < 300 {
		return 0
	}
	return int64(gameTime/300) * 300
}

// CurrentMark returns the current absolute 5-minute mark for the hook.
func (Periodic5MinHook) CurrentMark(gameTime float64) int64 {
	return CurrentMark(gameTime)
}

// Hook is the interface implemented by all Judge triggers.
type Hook interface {
	Name() string
	ShouldFire(ctx types.HookContext) (bool, error)
	Instruction() string
	// CurrentMark returns the absolute mark (in seconds) for this hook at the
	// given game time. It is used by the orchestrator to establish a baseline
	// when the system first detects an active match.
	CurrentMark(gameTime float64) int64
}

// Registry keeps registered hooks and can evaluate them.
type Registry struct {
	hooks []Hook
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// Register adds a hook to the registry.
func (r *Registry) Register(h Hook) {
	r.hooks = append(r.hooks, h)
}

// Hooks returns the registered hooks.
func (r *Registry) Hooks() []Hook {
	return r.hooks
}

// Evaluate runs every hook against the given context and returns all triggers.
func (r *Registry) Evaluate(ctx types.HookContext) ([]types.Trigger, error) {
	var triggers []types.Trigger
	for _, h := range r.hooks {
		fire, err := h.ShouldFire(ctx)
		if err != nil {
			return nil, fmt.Errorf("hook %s: %w", h.Name(), err)
		}
		if fire {
			triggers = append(triggers, types.Trigger{
				HookName: h.Name(),
				Question: h.Instruction(),
			})
		}
	}
	return triggers, nil
}
