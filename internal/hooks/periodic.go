// Package hooks contains trigger conditions that decide when to invoke the Judge.
package hooks

import (
	"fmt"

	"lol-telemetry/internal/types"
)

// PeriodicHook fires once for every absolute 5-minute mark of the game clock.
const PeriodicHookName = "periodic-5min"

// Periodic5MinHook is the hook that triggers at configurable intervals.
type Periodic5MinHook struct {
	IntervalSeconds int
}

// Name returns the hook identifier.
func (Periodic5MinHook) Name() string {
	return PeriodicHookName
}

// Instruction returns the question asked to the Judge.
func (Periodic5MinHook) Instruction() string {
	return "Evaluate the current macro state and give one actionable sentence of advice."
}

// interval returns the effective interval; 0 means use default.
func (h *Periodic5MinHook) interval() int {
	if h.IntervalSeconds > 0 {
		return h.IntervalSeconds
	}
	return 300
}

// ShouldFire returns true when the game clock crosses a new mark.
func (h *Periodic5MinHook) ShouldFire(ctx types.HookContext) (bool, error) {
	if ctx.GameTime <= 0 {
		return false, nil
	}
	mark := h.mark(ctx.GameTime)
	if mark == 0 {
		return false, nil
	}
	last, ok := ctx.PrevFired[PeriodicHookName]
	if !ok {
		return false, nil
	}
	if last >= mark {
		return false, nil
	}
	return true, nil
}

// CurrentMark returns the current absolute mark for the hook.
func (h *Periodic5MinHook) CurrentMark(gameTime float64) int64 {
	return h.mark(gameTime)
}

func (h *Periodic5MinHook) mark(gameTime float64) int64 {
	interval := h.interval()
	if gameTime < float64(interval) {
		return 0
	}
	return int64(gameTime/float64(interval)) * int64(interval)
}

// Configure implements Configurable.
func (h *Periodic5MinHook) Configure(params map[string]any) error {
	if v, ok := params["intervalSeconds"]; ok {
		f, ok := toFloat(v)
		if !ok || f < 60 {
			return fmt.Errorf("intervalSeconds must be a number >= 60, got %v", v)
		}
		h.IntervalSeconds = int(f)
	}
	return nil
}

// Spec implements Configurable.
func (h *Periodic5MinHook) Spec() map[string]ParamSpec {
	return map[string]ParamSpec{
		"intervalSeconds": {Type: "int", Default: 300, Min: 60},
	}
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	}
	return 0, false
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

type hookEntry struct {
	hook    Hook
	enabled bool
}

// Registry keeps registered hooks and can evaluate them.
type Registry struct {
	entries []hookEntry
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// Register adds a hook to the registry (enabled by default).
func (r *Registry) Register(h Hook) {
	r.entries = append(r.entries, hookEntry{hook: h, enabled: true})
}

// Hooks returns the registered hooks (both enabled and disabled).
func (r *Registry) Hooks() []Hook {
	out := make([]Hook, len(r.entries))
	for i, e := range r.entries {
		out[i] = e.hook
	}
	return out
}

// GetHook returns the hook with the given name.
func (r *Registry) GetHook(name string) (Hook, bool) {
	for _, e := range r.entries {
		if e.hook.Name() == name {
			return e.hook, true
		}
	}
	return nil, false
}

// SetEnabled enables or disables a hook by name.
func (r *Registry) SetEnabled(name string, enabled bool) error {
	for i := range r.entries {
		if r.entries[i].hook.Name() == name {
			r.entries[i].enabled = enabled
			return nil
		}
	}
	return fmt.Errorf("hook %q not found", name)
}

// Configure applies params to a configurable hook.
func (r *Registry) Configure(name string, params map[string]any) error {
	for _, e := range r.entries {
		if e.hook.Name() == name {
			c, ok := e.hook.(Configurable)
			if !ok {
				return fmt.Errorf("hook %q does not support configuration", name)
			}
			return c.Configure(params)
		}
	}
	return fmt.Errorf("hook %q not found", name)
}

// Snapshot returns a view of all hooks and their current state.
func (r *Registry) Snapshot() []HookView {
	out := make([]HookView, len(r.entries))
	for i, e := range r.entries {
		v := HookView{Name: e.hook.Name(), Enabled: e.enabled}
		if c, ok := e.hook.(Configurable); ok {
			v.Schema = c.Spec()
			v.Params = readParams(e.hook)
		}
		out[i] = v
	}
	return out
}

func readParams(h Hook) map[string]any {
	switch h := h.(type) {
	case *Periodic5MinHook:
		return map[string]any{"intervalSeconds": float64(h.IntervalSeconds)}
	case *RecallHook:
		return map[string]any{
			"goldThreshold":      h.GoldThreshold,
			"minGameTimeSeconds": h.MinGameTimeSeconds,
		}
	case *LaningPhaseEndHook:
		return map[string]any{"markSeconds": h.MarkSeconds}
	}
	return nil
}

// HookView is a serializable view of a hook's runtime state.
type HookView struct {
	Name    string              `json:"name"`
	Enabled bool                `json:"enabled"`
	Params  map[string]any      `json:"params,omitempty"`
	Schema  map[string]ParamSpec `json:"schema,omitempty"`
}

// HookPatch is a partial update for a hook's runtime state.
type HookPatch struct {
	Name    string         `json:"name"`
	Enabled *bool          `json:"enabled,omitempty"`
	Params  map[string]any `json:"params,omitempty"`
}

// Evaluate runs every enabled hook against the given context and returns all triggers.
func (r *Registry) Evaluate(ctx types.HookContext) ([]types.Trigger, error) {
	var triggers []types.Trigger
	for _, e := range r.entries {
		if !e.enabled {
			continue
		}
		fire, err := e.hook.ShouldFire(ctx)
		if err != nil {
			return nil, fmt.Errorf("hook %s: %w", e.hook.Name(), err)
		}
		if fire {
			triggers = append(triggers, types.Trigger{
				HookName: e.hook.Name(),
				Question: e.hook.Instruction(),
			})
		}
	}
	return triggers, nil
}
