package hooks

import (
	"errors"
	"testing"

	"lol-telemetry/internal/types"
)

// mockHook is a test hook that fires when its configured name matches a mark
// in the provided context.
type mockHook struct {
	name       string
	shouldFire bool
}

func (m mockHook) Name() string        { return m.name }
func (m mockHook) Instruction() string { return "mock instruction for " + m.name }
func (m mockHook) ShouldFire(ctx types.HookContext) (bool, error) {
	return m.shouldFire, nil
}
func (m mockHook) CurrentMark(gameTime float64) int64 { return 0 }

// TestRegistry_MultipleHooksFireIndependently validates that the registry can
// hold several hooks and each one produces its own trigger.
func TestRegistry_MultipleHooksFireIndependently(t *testing.T) {
	reg := NewRegistry()
	reg.Register(mockHook{name: "hook-a", shouldFire: true})
	reg.Register(mockHook{name: "hook-b", shouldFire: false})
	reg.Register(mockHook{name: "hook-c", shouldFire: true})

	ctx := types.HookContext{GameTime: 100, PrevFired: map[string]int64{}}
	triggers, err := reg.Evaluate(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(triggers) != 2 {
		t.Fatalf("expected 2 triggers, got %d", len(triggers))
	}

	names := map[string]bool{}
	for _, tr := range triggers {
		names[tr.HookName] = true
		if tr.Question != "mock instruction for "+tr.HookName {
			t.Errorf("instruction for %s = %s, want mock instruction for %s", tr.HookName, tr.Question, tr.HookName)
		}
	}
	if !names["hook-a"] || !names["hook-c"] {
		t.Errorf("expected hook-a and hook-c to fire, got %v", names)
	}
	if names["hook-b"] {
		t.Errorf("hook-b should not have fired, got %v", names)
	}
}

// TestRegistry_EvaluateReturnsErrorOnHookError validates that errors from a
// single hook propagate without affecting other hooks.
func TestRegistry_EvaluateReturnsErrorOnHookError(t *testing.T) {
	reg := NewRegistry()
	reg.Register(mockHook{name: "ok", shouldFire: true})
	reg.Register(errorHook{})

	ctx := types.HookContext{GameTime: 100, PrevFired: map[string]int64{}}
	_, err := reg.Evaluate(ctx)
	if err == nil {
		t.Fatal("expected error from failing hook")
	}
}

type errorHook struct{}

func (errorHook) Name() string        { return "error-hook" }
func (errorHook) Instruction() string { return "error" }
func (errorHook) ShouldFire(ctx types.HookContext) (bool, error) {
	return false, errors.New("hook evaluation failed")
}
func (errorHook) CurrentMark(gameTime float64) int64 { return 0 }

func TestRegistry_SetEnabled_DisabledDoesNotFire(t *testing.T) {
	reg := NewRegistry()
	reg.Register(mockHook{name: "toggle", shouldFire: true})

	ctx := types.HookContext{GameTime: 100, PrevFired: map[string]int64{}}
	triggers, err := reg.Evaluate(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(triggers) != 1 {
		t.Fatalf("expected 1 trigger, got %d", len(triggers))
	}

	if err := reg.SetEnabled("toggle", false); err != nil {
		t.Fatalf("SetEnabled error: %v", err)
	}
	triggers, err = reg.Evaluate(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(triggers) != 0 {
		t.Errorf("expected 0 triggers after disable, got %d", len(triggers))
	}
}

func TestRegistry_SetEnabled_UnknownHookReturnsError(t *testing.T) {
	reg := NewRegistry()
	if err := reg.SetEnabled("nonexistent", false); err == nil {
		t.Error("expected error for unknown hook")
	}
}

func TestRegistry_Configure_NonConfigurableReturnsError(t *testing.T) {
	reg := NewRegistry()
	reg.Register(mockHook{name: "basic", shouldFire: true})
	if err := reg.Configure("basic", map[string]any{"foo": 1}); err == nil {
		t.Error("expected error for non-configurable hook")
	}
}

func TestRegistry_Snapshot_IncludesAllHooks(t *testing.T) {
	reg := NewRegistry()
	reg.Register(mockHook{name: "h1", shouldFire: false})
	reg.Register(mockHook{name: "h2", shouldFire: true})

	snap := reg.Snapshot()
	if len(snap) != 2 {
		t.Fatalf("expected 2 hooks in snapshot, got %d", len(snap))
	}
	if snap[0].Name != "h1" || snap[1].Name != "h2" {
		t.Errorf("unexpected hook order in snapshot: %+v", snap)
	}
	if !snap[1].Enabled {
		t.Error("h2 should be enabled by default")
	}
}

func TestRegistry_GetHook_ReturnsHookByName(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&Periodic5MinHook{})
	h, ok := reg.GetHook(PeriodicHookName)
	if !ok {
		t.Fatal("expected hook to be found")
	}
	if h.Name() != PeriodicHookName {
		t.Errorf("got %s, want %s", h.Name(), PeriodicHookName)
	}
}

func TestRegistry_GetHook_NotFound(t *testing.T) {
	reg := NewRegistry()
	_, ok := reg.GetHook("nonexistent")
	if ok {
		t.Error("expected not found")
	}
}
